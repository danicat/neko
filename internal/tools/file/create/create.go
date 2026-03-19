// Package create implements the create_file tool.
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

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
}

// Register registers the create_file tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["create_file"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return createHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	File    string `json:"file" jsonschema:"The path to the file to create"`
	Content string `json:"content" jsonschema:"The content to write"`
}

func createHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	if args.File == "" {
		return errorResult("name (file path) cannot be empty"), nil, nil
	}

	finalContent := []byte(args.Content)

	//nolint:gosec // G301
	if err := os.MkdirAll(filepath.Dir(args.File), 0755); err != nil {
		return errorResult(fmt.Sprintf("failed to create directory: %v", err)), nil, nil
	}

	//nolint:gosec // G306
	if err := os.WriteFile(args.File, finalContent, 0644); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), nil, nil
	}

	be := s.ForFile(ctx, args.File)
	var warning string

	if be != nil {

		if fmtErr := be.Format(ctx, args.File); fmtErr != nil {
			warning = fmt.Sprintf("\n\n**WARNING:** formatting failed: %v", fmtErr)
		}
		if err := be.Validate(ctx, args.File); err != nil {
			warning += fmt.Sprintf("\n\n**WARNING:** Post-write syntax check failed: %v", err)
		}
		if formatted, err := os.ReadFile(args.File); err == nil {
			finalContent = formatted
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Successfully wrote `%s` (%d bytes)", args.File, len(finalContent)))
	if be != nil {
		sb.WriteString(fmt.Sprintf("\n- ✅ %s format (auto-format)", be.Name()))
		sb.WriteString("\n- ✅ syntax verification")
	} else {
		sb.WriteString("\n- Note: Syntax validation and formatting skipped for this file type.")
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
