// Package modernize implements the modernize_code tool.
package modernize

import (
	"context"
	"fmt"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	Registry() *backend.Registry
}

// Register registers the modernize_code tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["modernize_code"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return modernizeHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	Dir      string `json:"dir,omitempty" jsonschema:"Directory to modernize (default: current)"`
	Language string `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
	Fix      bool   `json:"fix,omitempty" jsonschema:"If true, apply fixes automatically"`
}

func modernizeHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	dir := args.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := roots.Global.Validate(dir)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	var be backend.LanguageBackend
	if args.Language != "" {
		be = s.Registry().Get(args.Language)
		if be == nil {
			return errorResult(fmt.Sprintf("unknown language backend: %s", args.Language)), nil, nil
		}
	} else {
		be = s.Registry().ForDir(absDir)
	}

	if be == nil {
		return errorResult("No language backend detected for this directory."), nil, nil
	}

	output, err := be.Modernize(ctx, absDir, args.Fix)
	if err != nil {
		return errorResult(fmt.Sprintf("modernize failed: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
