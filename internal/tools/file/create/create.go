// Package create implements the file_create tool.
package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the file_create tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["file_create"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return createHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Filename string `json:"filename" jsonschema:"The path to the file to create"`
	Content  string `json:"content" jsonschema:"The content to write"`
}

func createHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	if args.Filename == "" {
		return errorResult("name (file path) cannot be empty"), nil, nil
	}

	finalContent := []byte(args.Content)

	//nolint:gosec // G301
	if err := os.MkdirAll(filepath.Dir(args.Filename), 0755); err != nil {
		return errorResult(fmt.Sprintf("failed to create directory: %v", err)), nil, nil
	}

	//nolint:gosec // G306
	if err := os.WriteFile(args.Filename, finalContent, 0644); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), nil, nil
	}

	be := reg.ForFile(args.Filename)
	var warning string

	if be != nil {
		if fmtErr := be.Format(ctx, args.Filename); fmtErr != nil {
			warning = fmt.Sprintf("\n\n**WARNING:** formatting failed: %v", fmtErr)
		}
		if err := be.Validate(ctx, args.Filename); err != nil {
			warning += fmt.Sprintf("\n\n**WARNING:** Post-write syntax check failed: %v", err)
		}
		if formatted, err := os.ReadFile(args.Filename); err == nil {
			finalContent = formatted
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Successfully wrote `%s` (%d bytes)", args.Filename, len(finalContent)))
	if be != nil {
		sb.WriteString(fmt.Sprintf("\n- ✅ %s format (auto-format)", be.Name()))
		sb.WriteString("\n- ✅ syntax verification")
	}
	if warning != "" {
		sb.WriteString(warning)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
