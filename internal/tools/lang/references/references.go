// Package references implements the find_references tool via LSP.
package references

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the find_references tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["find_references"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return handler(ctx, args, reg)
	})
}

// Params defines the input parameters for the find_references tool.
type Params struct {
	File        string `json:"file" jsonschema:"The path to the source file"`
	Line        int    `json:"line" jsonschema:"Line number (1-based)"`
	Col         int    `json:"col" jsonschema:"Column number (1-based)"`
	IncludeDecl *bool  `json:"include_declaration,omitempty" jsonschema:"Include the declaration in results (default: true)"`
}

func handler(ctx context.Context, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	absPath, err := roots.Global.Validate(args.File)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	be := reg.ForFile(absPath)
	if be == nil {
		return errorResult(fmt.Sprintf("no language backend for %s", absPath)), nil, nil
	}

	command, cmdArgs, ok := be.LSPCommand()
	if !ok {
		return errorResult(fmt.Sprintf("no LSP server configured for %s", be.Name())), nil, nil
	}

	if _, err := exec.LookPath(command); err != nil {
		return errorResult(fmt.Sprintf("LSP server %q not found in PATH", command)), nil, nil
	}

	workspaceRoot, _ := roots.Global.Validate(".")
	client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, command, cmdArgs, be.InitializationOptions())
	if err != nil {
		return errorResult(fmt.Sprintf("failed to start LSP server: %v", err)), nil, nil
	}

	includeDecl := args.IncludeDecl == nil || *args.IncludeDecl

	locations, err := client.References(ctx, absPath, args.Line, args.Col, includeDecl)
	if err != nil {
		return errorResult(fmt.Sprintf("references lookup failed: %v", err)), nil, nil
	}

	text := fmt.Sprintf("Found %d reference(s):\n%s", len(locations), lsp.FormatLocations(locations))
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
