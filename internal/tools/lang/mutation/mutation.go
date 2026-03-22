// Package mutation implements the test_mutations tool.
package mutation

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	ResolveBackend(language string) (backend.LanguageBackend, error)
	ProjectRoot() string
}

// Register registers the test_mutations tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["test_mutations"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return mutationHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	Dir      string `json:"dir,omitempty" jsonschema:"Directory to run mutation testing in (default: current)"`
	Language string `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
}

func mutationHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	var absDir string
	if args.Dir == "" || args.Dir == "." {
		absDir = s.ProjectRoot()
		if absDir == "" {
			absDir, _ = filepath.Abs(".")
		}
	} else {
		var err error
		absDir, err = filepath.Abs(args.Dir)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := roots.Global.Validate(absDir); err != nil {
		return nil, nil, err
	}

	be, err := s.ResolveBackend(args.Language)
	if err != nil {
		return nil, nil, err
	}

	output, err := be.MutationTest(ctx, absDir)
	if err != nil {
		return nil, nil, fmt.Errorf("mutation testing failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}
