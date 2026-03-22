// Package docs implements the read_docs tool.
package docs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	ResolveBackend(language string) (backend.LanguageBackend, error)
	ProjectRoot() string
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
		return nil, nil, fmt.Errorf("import_path is required")
	}

	// Validate format
	format := strings.ToLower(args.Format)
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "json" {
		return nil, nil, fmt.Errorf("invalid format: must be 'markdown' or 'json'")
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

	// For documentation, we don't strictly enforce boundary check if it's external,
	// but we resolve the path for language detection.
	be, err := s.ResolveBackend(args.Language)
	if err != nil {
		return nil, nil, err
	}

	// For JSON format, use the Go backend's structured output if available
	if format == "json" {
		if goBe, ok := be.(*golang.Backend); ok {
			doc, err := goBe.FetchDocsJSON(ctx, args.ImportPath, args.Symbol)
			if err != nil {
				return nil, nil, fmt.Errorf("documentation lookup failed: %w", err)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: doc}},
			}, nil, nil
		}
	}

	doc, err := be.FetchDocs(ctx, absDir, args.ImportPath, args.Symbol)
	if err != nil {
		return nil, nil, fmt.Errorf("documentation lookup failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: doc}},
	}, nil, nil
}
