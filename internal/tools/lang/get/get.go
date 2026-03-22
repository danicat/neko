// Package get implements the add_dependencies tool.
package get

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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

// Register registers the add_dependencies tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["add_dependencies"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return getHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	Packages []string `json:"packages" jsonschema:"List of packages to install"`
	Dir      string   `json:"dir,omitempty" jsonschema:"Directory containing the project (default: current)"`
	Language string   `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
}

func getHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	if len(args.Packages) == 0 {
		return nil, nil, fmt.Errorf("at least one package is required")
	}

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

	output, err := be.AddDependency(ctx, absDir, args.Packages)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add dependency: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(output)
	sb.WriteString("\n\n## Documentation\n")

	for _, pkg := range args.Packages {
		doc, err := be.FetchDocs(ctx, absDir, pkg, "")
		if err == nil && doc != "" {
			sb.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", pkg, doc))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}
