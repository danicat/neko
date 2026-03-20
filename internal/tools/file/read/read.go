// Package read implements the read_file tool.
package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/rag"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/core/shared"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	ShouldShowDoc(language, pkg string) bool
	RAG() *rag.Engine
}

// Register registers the read_file tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["read_file"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return readHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters for the read_file tool.
type Params struct {
	File      string `json:"file" jsonschema:"The path to the file to read"`
	Outline   bool   `json:"outline,omitempty" jsonschema:"Optional: if true, returns the structure (AST) only"`
	StartLine int    `json:"start_line,omitempty" jsonschema:"Optional: start reading from this line number"`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"Optional: stop reading at this line number"`
}

func readHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	absPath, err := roots.Global.Validate(args.File)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}
	args.File = absPath

	be := s.ForFile(ctx, absPath)

	// Outline Mode
	if args.Outline && args.StartLine == 0 {
		var out string
		var lspClient *lsp.Client
		if be != nil {
			if cmd, cmdArgs, ok := be.LSPCommand(); ok {
				workspaceRoot, _ := roots.Global.Validate(".")
				if client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions()); err == nil {
					lspClient = client
					symbols, err := lspClient.DocumentSymbol(ctx, absPath)
					if err == nil && len(symbols) > 0 {
						out = lsp.FormatSymbols(symbols)
					}
				}
			}
		}

		// Fallback to backend parser if LSP failed or not available
		if out == "" && be != nil {
			out, err = be.Outline(ctx, absPath)
			if err != nil {
				return errorResult(fmt.Sprintf("failed to generate outline: %v", err)), nil, nil
			}
		}

		if out == "" {
			return errorResult("outline not supported for this file type"), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# File: %s (Outline)\n\n", absPath))
		sb.WriteString("```")
		sb.WriteString(langTag(absPath))
		sb.WriteString("\n")
		sb.WriteString(out)
		sb.WriteString("\n```\n")

		if lspClient != nil {
			lspClient.DidClose(ctx, absPath)
		}

		// Show third-party imports if available
		if be != nil {
			if imports, err := be.ParseImports(ctx, absPath); err == nil && len(imports) > 0 {
				sb.WriteString("\n## Third-Party Imports\n")
				for _, imp := range imports {
					sb.WriteString(fmt.Sprintf("- %s\n", imp))
				}
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	}

	// Read Content
	//nolint:gosec // G304
	content, err := os.ReadFile(absPath)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), nil, nil
	}

	original := string(content)

	// Warm-start LSP
	var lspClient *lsp.Client
	if be != nil {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot, _ := roots.Global.Validate(".")
			if client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions()); err == nil {
				lspClient = client
				lspClient.DidOpen(ctx, absPath, original)
			}
		}
	}

	startLine := args.StartLine
	if startLine <= 0 {
		startLine = 1
	}
	endLine := args.EndLine

	startOffset, endOffset, err := shared.GetLineOffsets(original, startLine, endLine)
	if err != nil {
		return errorResult(fmt.Sprintf("line range error: %v", err)), nil, nil
	}

	viewContent := original[startOffset:endOffset]
	lines := strings.Split(viewContent, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && !strings.HasSuffix(viewContent, "\n") {
		lines = lines[:len(lines)-1]
	}

	var contentWithLines strings.Builder
	for i, line := range lines {
		lineNum := startLine + i
		annotatedLine := line
		if lspClient != nil {
			// Fetch symbols for this specific line
			symbols, _ := lspClient.DocumentSymbol(ctx, absPath)
			for _, s := range symbols {
				if s.Range.Start.Line+1 == lineNum {
					if s.Kind == 12 || s.Kind == 6 || s.Kind == 13 { // Func, Method, Variable
						hover, _ := lspClient.Hover(ctx, absPath, lineNum, s.Range.Start.Character+1)
						if hover != nil {
							sig := lsp.HoverText(hover)
							sigLines := strings.Split(sig, "\n")
							if len(sigLines) > 0 {
								cleanSig := strings.TrimSpace(strings.Trim(sigLines[0], "`"))
								// Skip common primitives
								if cleanSig != "string" && cleanSig != "int" && cleanSig != "error" && cleanSig != "bool" {
									annotatedLine = fmt.Sprintf("%s <NEKO>type: %s</NEKO>", annotatedLine, cleanSig)
								}
							}
						}
					}
				}
			}
		}
		contentWithLines.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, annotatedLine))
	}

	isPartial := args.StartLine > 1 || args.EndLine > 0

	var sb strings.Builder
	rangeInfo := ""
	if isPartial {
		rangeInfo = fmt.Sprintf(" (Lines %d-%d)", startLine, startLine+len(lines)-1)
	}
	sb.WriteString(fmt.Sprintf("# File: %s%s\n\n", args.File, rangeInfo))

	sb.WriteString("```")
	sb.WriteString(langTag(absPath))
	sb.WriteString("\n")
	sb.WriteString(contentWithLines.String())
	sb.WriteString("```\n\n")

	if args.Outline && be == nil {
		sb.WriteString("*Note: Outline not available for this file type. Showing full content instead.*\n\n")
	}

	if isPartial {
		sb.WriteString("*Note: Partial read - analysis skipped.*\n\n")
	}

	if lspClient != nil {
		sb.WriteString("--- \n💡 **NOTE**: Lines containing `<NEKO>...</NEKO>` are virtual semantic annotations. They do not exist on disk and will be automatically ignored during edits.\n\n")
		// relinquished session
		lspClient.DidClose(ctx, absPath)
	}

	// Import analysis for full reads
	if !isPartial && be != nil {
		imports, parseErr := be.ParseImports(ctx, absPath)
		if parseErr == nil && len(imports) > 0 {
			if importDocs, err := be.ImportDocs(ctx, imports); err == nil && len(importDocs) > 0 {
				var filteredDocs []string
				for i, pd := range importDocs {
					pkg := strings.Trim(imports[i], "\"")
					if s.ShouldShowDoc(be.Name(), pkg) {
						filteredDocs = append(filteredDocs, pd)
					}
				}

				if len(filteredDocs) > 0 {
					sb.WriteString("## Imported Packages\n")
					for _, pd := range filteredDocs {
						sb.WriteString(pd + "\n")
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func langTag(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".py", ".pyi":
		return "python"
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	case ".toml":
		return "toml"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
