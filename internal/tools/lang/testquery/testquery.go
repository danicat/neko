// Package testquery implements the query_tests tool.
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

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	ResolveBackend(language string) (backend.LanguageBackend, error)
}

// Register registers the query_tests tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["query_tests"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return queryHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	Query    string `json:"query" jsonschema:"SQL query to run against test/coverage data"`
	Dir      string `json:"dir,omitempty" jsonschema:"Project directory (default: current)"`
	Language string `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
	Pkg      string `json:"pkg,omitempty" jsonschema:"Package pattern to analyze (default: ./...)"`
	Rebuild  bool   `json:"rebuild,omitempty" jsonschema:"Force rebuild of test database"`
}

func queryHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
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

	be, err := s.ResolveBackend(args.Language)
	if err != nil {
		return errorResult(err.Error()), nil, nil
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
