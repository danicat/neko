// Package get implements the add_dependency tool.
package get

import (
	"context"
	"fmt"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the add_dependency tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["add_dependency"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return getHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Packages []string `json:"packages" jsonschema:"List of packages to install"`
	Dir      string   `json:"dir,omitempty" jsonschema:"Directory containing the project (default: current)"`
}

func getHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	if len(args.Packages) == 0 {
		return errorResult("at least one package is required"), nil, nil
	}

	dir := args.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := roots.Global.Validate(dir)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	be := reg.ForDir(absDir)
	if be == nil {
		return errorResult("No language backend detected for this directory."), nil, nil
	}

	output, err := be.AddDependency(ctx, absDir, args.Packages)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to add dependency: %v", err)), nil, nil
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

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
