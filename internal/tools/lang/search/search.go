// Package search implements the semantic_search tool.
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/danicat/neko/internal/core/rag"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	RAG() *rag.Engine
}

// Register registers the semantic_search tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["semantic_search"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return searchHandler(ctx, args, s)
	})
}

// Params defines the input parameters for the semantic_search tool.
type Params struct {
	Query string `json:"query" jsonschema:"The natural language query (e.g., 'handling of lsp client lifecycle')"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return (default 5, max 10)"`
}

func searchHandler(ctx context.Context, args Params, s Server) (*mcp.CallToolResult, any, error) {
	if args.Query == "" {
		return errorResult("query is required"), nil, nil
	}

	engine := s.RAG()
	if engine == nil {
		return errorResult("RAG engine not initialized for this project"), nil, nil
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	results, err := engine.Search(ctx, args.Query, limit)
	if err != nil {
		return errorResult(fmt.Sprintf("search failed: %v", err)), nil, nil
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "No relevant code snippets found for your query."}},
		}, nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 🔍 Semantic Search Results for: '%s'\n\n", args.Query))

	for i, res := range results {
		path := res.Metadata["path"]
		line := res.Metadata["line"]
		name := res.Metadata["name"]
		
		symbolInfo := ""
		if name != "" {
			symbolInfo = fmt.Sprintf(" (Symbol: %s)", name)
		}

		sb.WriteString(fmt.Sprintf("#### %d. %s:%s%s (Score: %.2f)\n", i+1, path, line, symbolInfo, res.Similarity))
		sb.WriteString("```\n")
		sb.WriteString(res.Content)
		sb.WriteString("\n```\n\n")
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
