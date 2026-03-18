// Package testquery implements the test_query tool.
package testquery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the test_query tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["test_query"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return queryHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Query   string `json:"query" jsonschema:"SQL query to run against test/coverage data"`
	Dir     string `json:"dir,omitempty" jsonschema:"Project directory (default: current)"`
	Pkg     string `json:"pkg,omitempty" jsonschema:"Package pattern to analyze (default: ./...)"`
	Rebuild bool   `json:"rebuild,omitempty" jsonschema:"Force rebuild of test database"`
}

func queryHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	if args.Query == "" {
		return errorResult("query is required"), nil, nil
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

	pkg := args.Pkg
	if pkg == "" {
		pkg = "./..."
	}

	// Build the DB if it doesn't exist or if rebuild is requested
	dbPath := filepath.Join(absDir, "testquery.db")
	if args.Rebuild || !fileExists(dbPath) {
		if err := be.BuildTestDB(ctx, absDir, pkg); err != nil {
			// Build may fail if tests fail, but the DB might still be usable
			if !fileExists(dbPath) {
				return errorResult(fmt.Sprintf("failed to build test database: %v", err)), nil, nil
			}
		}
	}

	// Execute the query
	output, err := be.QueryTestDB(ctx, absDir, args.Query)
	if err != nil {
		return errorResult(fmt.Sprintf("query failed: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
