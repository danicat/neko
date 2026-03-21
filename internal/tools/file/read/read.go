// Package read implements the read_file tool.
package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/core/shared"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// identRe matches Go identifiers for type info extraction.
var identRe = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	ShouldShowDoc(language, pkg string) bool
	HasSeenTypeInfo(name string) bool
	ProjectRoot() string
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
	var absPath string
	if args.File == "" || args.File == "." {
		absPath = s.ProjectRoot()
		if absPath == "" {
			absPath, _ = filepath.Abs(".")
		}
	} else {
		var err error
		absPath, err = filepath.Abs(args.File)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
	}

	if err := roots.Global.Validate(absPath); err != nil {
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
				workspaceRoot := s.ProjectRoot()
				if workspaceRoot == "" {
					workspaceRoot, _ = filepath.Abs(".")
				}
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
			var err error
			out, err = be.Outline(ctx, absPath)
			if err != nil {
				return errorResult(fmt.Sprintf("failed to generate outline: %v", err)), nil, nil
			}
		}

		if out == "" {
			return errorResult("outline not supported for this file type"), nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# File: %s (Outline)\n\n", absPath)
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
					fmt.Fprintf(&sb, "- %s\n", imp)
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
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
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
	var typeInfoEntries []string

	// Build the line-by-line view and collect identifiers for Type Info
	var idents []struct {
		name string
		line int
		col  int
	}
	// Keywords to skip
	keywords := map[string]bool{
		"func": true, "var": true, "type": true, "struct": true, "interface": true,
		"package": true, "import": true, "return": true, "if": true, "else": true,
		"for": true, "range": true, "go": true, "select": true, "case": true,
		"default": true, "switch": true, "defer": true, "map": true, "chan": true,
		"nil": true, "true": true, "false": true, "err": true, "error": true,
		"string": true, "int": true, "bool": true, "ctx": true, "context": true,
		"float32": true, "float64": true, "byte": true, "rune": true, "uint": true,
		"int64": true, "uint64": true, "int32": true, "uint32": true,
		"make": true, "new": true, "len": true, "cap": true, "append": true, "copy": true,
		"delete": true, "close": true, "panic": true, "recover": true, "complex": true,
		"real": true, "imag": true, "print": true, "println": true,
	}

	for i, line := range lines {
		lineNum := startLine + i
		fmt.Fprintf(&contentWithLines, "%4d | %s\n", lineNum, line)

		if lspClient != nil {
			matches := identRe.FindAllStringIndex(line, -1)
			for _, m := range matches {
				name := line[m[0]:m[1]]
				if !keywords[name] {
					idents = append(idents, struct {
						name string
						line int
						col  int
					}{name, lineNum, m[0] + 1})
				}
			}
		}
	}

	// Resolve identifiers using LSP
	if lspClient != nil {
		for _, id := range idents {
			if s.HasSeenTypeInfo(id.name) {
				continue
			}

			// 1. Check if it's external
			locs, err := lspClient.Definition(ctx, absPath, id.line, id.col)
			isExternal := false
			isStdLib := false
			if err == nil && len(locs) > 0 {
				for _, loc := range locs {
					if !strings.HasSuffix(loc.URI, filepath.Base(absPath)) {
						isExternal = true
					}
					if be != nil && be.IsStdLibURI(loc.URI) {
						isStdLib = true
					}
				}
			}

			if isExternal && !isStdLib {
				text, err := lspClient.EnhancedHover(ctx, absPath, id.line, id.col)
				if err == nil && text != "" {
					formattedInfo := formatTypeInfo(id.line, id.name, text)
					if formattedInfo != "" {
						typeInfoEntries = append(typeInfoEntries, formattedInfo)
					}
				}
			}
		}
	}

	isPartial := args.StartLine > 1 || args.EndLine > 0

	var sb strings.Builder
	rangeInfo := ""
	if isPartial {
		rangeInfo = fmt.Sprintf(" (Lines %d-%d)", startLine, startLine+len(lines)-1)
	}
	fmt.Fprintf(&sb, "# File: %s%s\n\n", args.File, rangeInfo)

	sb.WriteString("```")
	sb.WriteString(langTag(absPath))
	sb.WriteString("\n")
	sb.WriteString(contentWithLines.String())
	sb.WriteString("```\n\n")

	if len(typeInfoEntries) > 0 {
		sb.WriteString("## Type Info\n")
		for _, entry := range typeInfoEntries {
			sb.WriteString(entry + "\n")
		}
		sb.WriteString("\n")
	}

	if args.Outline && be == nil {
		sb.WriteString("*Note: Outline not available for this file type. Showing full content instead.*\n\n")
	}

	if isPartial {
		sb.WriteString("*Note: Partial read - analysis skipped.*\n\n")
	}

	if lspClient != nil {
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

func formatTypeInfo(lineNum int, name string, hoverText string) string {
	// Simple Markdown parser for hover text
	lines := strings.Split(hoverText, "\n")

	var inCodeBlock bool
	var codeLines []string
	var docLines []string

	for _, l := range lines {
		if strings.HasPrefix(l, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			codeLines = append(codeLines, l)
		} else {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "---") && !strings.HasPrefix(l, "[`") {
				docLines = append(docLines, l)
			}
		}
	}

	// 1. Extract base type/signature
	var baseType string
	var fields []string
	var methods []string

	inStruct := false
	for _, cl := range codeLines {
		cl = strings.TrimSpace(cl)
		if cl == "" {
			continue
		}

		// Strip trailing comments for cleaner parsing (e.g. gopls // size=40)
		if idx := strings.Index(cl, "//"); idx != -1 {
			cl = strings.TrimSpace(cl[:idx])
		}

		if strings.HasPrefix(cl, "func ") {
			if !strings.HasPrefix(cl, "func (") {
				// Top-level function: extract return type
				idx := strings.LastIndex(cl, ")")
				if idx != -1 && idx < len(cl)-1 {
					retType := strings.TrimSpace(cl[idx+1:])
					if !strings.HasPrefix(retType, "{") && retType != "" {
						baseType = retType
					} else {
						baseType = "func"
					}
				} else {
					baseType = "func"
				}
			} else {
				methods = append(methods, cl)
			}
			continue
		}

		if strings.HasPrefix(cl, "type ") {
			// type Name struct or type Name interface
			cl = strings.TrimSuffix(cl, "{")
			parts := strings.Fields(cl)
			if len(parts) >= 3 {
				baseType = parts[2]
			} else if len(parts) >= 2 {
				baseType = parts[1]
			}
			inStruct = true
			continue
		}

		if inStruct {
			if cl == "}" {
				inStruct = false
				continue
			}
			// It's a field
			parts := strings.Fields(cl)
			if len(parts) >= 1 {
				fields = append(fields, parts[0])
			}
			continue
		}

		if strings.HasPrefix(cl, "var ") || strings.HasPrefix(cl, "field ") || strings.HasPrefix(cl, "const ") {
			parts := strings.Fields(cl)
			if len(parts) >= 3 {
				baseType = strings.Join(parts[2:], " ")
			}
		}
	}

	if baseType == "" {
		baseType = "unknown"
	}

	doc := ""
	if len(docLines) > 0 {
		doc = docLines[0]
		doc = strings.ReplaceAll(doc, "\\_", "_")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "- Line %d **%s** (%s)", lineNum, name, baseType)
	if doc != "" {
		fmt.Fprintf(&sb, ": %s", doc)
	}

	if len(fields) > 0 {
		fmt.Fprintf(&sb, "\n  Fields: %s", strings.Join(fields, ", "))
	}

	if len(methods) > 0 {
		cleanMethods := make([]string, 0, len(methods))
		cleanMethods = append(cleanMethods, methods...)
		fmt.Fprintf(&sb, "\n  Methods: %s", strings.Join(cleanMethods, ", "))
	}

	// Filter out low-value entries (like raw functions or unknown types without methods/fields)
	if baseType == "func" && len(methods) == 0 && len(fields) == 0 {
		return ""
	}
	if baseType == "unknown" && len(methods) == 0 && len(fields) == 0 {
		return ""
	}
	// Also skip basic standard library packages
	if strings.HasPrefix(doc, "Package ") {
		return ""
	}

		return sb.String()
}
