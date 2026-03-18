// Package read implements the smart_read tool.
package read

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/core/shared"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the smart_read tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["smart_read"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return readHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters for the smart_read tool.
type Params struct {
	Filename  string `json:"filename" jsonschema:"The path to the file to read"`
	Outline   bool   `json:"outline,omitempty" jsonschema:"Optional: if true, returns the structure (AST) only"`
	StartLine int    `json:"start_line,omitempty" jsonschema:"Optional: start reading from this line number"`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"Optional: stop reading at this line number"`
}

func readHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	absPath, err := roots.Global.Validate(args.Filename)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}
	args.Filename = absPath

	be := reg.ForFile(absPath)

	// Outline Mode
	if args.Outline && args.StartLine == 0 && be != nil {
		out, err := be.Outline(ctx, absPath)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to generate outline: %v", err)), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# File: %s (Outline)\n\n", absPath))
		sb.WriteString("```")
		sb.WriteString(langTag(absPath))
		sb.WriteString("\n")
		sb.WriteString(out)
		sb.WriteString("\n```\n")

		// Show third-party imports if available
		// Both backends' ParseImports return only third-party imports
		if imports, err := be.ParseImports(ctx, absPath); err == nil && len(imports) > 0 {
			sb.WriteString("\n## Third-Party Imports\n")
			for _, imp := range imports {
				sb.WriteString(fmt.Sprintf("- %s\n", imp))
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
		contentWithLines.WriteString(fmt.Sprintf("%4d | %s\n", startLine+i, line))
	}

	isPartial := args.StartLine > 1 || args.EndLine > 0

	var sb strings.Builder
	rangeInfo := ""
	if isPartial {
		rangeInfo = fmt.Sprintf(" (Lines %d-%d)", startLine, startLine+len(lines)-1)
	}
	sb.WriteString(fmt.Sprintf("# File: %s%s\n\n", args.Filename, rangeInfo))

	sb.WriteString("```")
	sb.WriteString(langTag(absPath))
	sb.WriteString("\n")
	sb.WriteString(contentWithLines.String())
	sb.WriteString("```\n\n")

	if isPartial {
		sb.WriteString("*Note: Partial read - analysis skipped.*\n\n")
	}

	// Import analysis for full reads
	if !isPartial && be != nil {
		imports, parseErr := be.ParseImports(ctx, absPath)
		if parseErr == nil && len(imports) > 0 {
			if importDocs, err := be.ImportDocs(ctx, imports); err == nil && len(importDocs) > 0 {
				sb.WriteString("## Imported Packages\n")
				for _, pd := range importDocs {
					sb.WriteString(pd + "\n")
				}
				sb.WriteString("\n")
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
