// Package docs implements the read_docs tool.
package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	Registry() *backend.Registry
}

// Register registers the read_docs tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["read_docs"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return docsHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	ImportPath string `json:"import_path" jsonschema:"The module or package to look up (e.g. net/http, pathlib)"`
	Symbol     string `json:"symbol,omitempty" jsonschema:"Optional: specific symbol within the module"`
	Dir        string `json:"dir,omitempty" jsonschema:"Optional: project directory to detect language (default: current)"`
	Language   string `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
	Format     string `json:"format,omitempty" jsonschema:"Output format: 'markdown' (default) or 'json'"`
}

func docsHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	if args.ImportPath == "" {
		return errorResult("import_path is required"), nil, nil
	}

	// Validate format
	format := strings.ToLower(args.Format)
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "json" {
		return errorResult("invalid format: must be 'markdown' or 'json'"), nil, nil
	}

	dir := args.Dir
	if dir == "" {
		dir = "."
	}
	absDir, _ := roots.Global.Validate(dir)

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
		return errorResult("No language backend detected. Ensure you're in a project directory or specify 'language'."), nil, nil
	}

	// For JSON format, use the Go backend's structured output if available
	if format == "json" {
		if goBe, ok := be.(*golang.Backend); ok {
			doc, err := goBe.FetchDocsJSON(ctx, args.ImportPath, args.Symbol)
			if err != nil {
				return errorResult(fmt.Sprintf("documentation lookup failed: %v", err)), nil, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: doc}},
			}, nil, nil
		}
	}

	doc, err := be.FetchDocs(ctx, absDir, args.ImportPath, args.Symbol)
	if err != nil {
		return errorResult(fmt.Sprintf("documentation lookup failed: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: doc}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
