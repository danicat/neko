// Package rename implements the rename_symbol tool.
package rename

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
}

// Register registers the rename_symbol tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["rename_symbol"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return renameHandler(ctx, args, s)
	})
}

// Params defines the input parameters for the rename_symbol tool.
type Params struct {
	File    string `json:"file" jsonschema:"The path to the source file"`
	Line    int    `json:"line" jsonschema:"Line number (1-based)"`
	Col     int    `json:"col" jsonschema:"Column number (1-based)"`
	NewName string `json:"new_name" jsonschema:"The new name for the symbol"`
}

func renameHandler(ctx context.Context, args Params, s Server) (*mcp.CallToolResult, any, error) {
	absPath, err := roots.Global.Validate(args.File)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	be := s.ForFile(ctx, absPath)
	if be == nil {
		return errorResult(fmt.Sprintf("no language backend for %s", absPath)), nil, nil
	}

	cmd, cmdArgs, ok := be.LSPCommand()
	if !ok {
		return errorResult(fmt.Sprintf("no LSP server configured for %s", be.Name())), nil, nil
	}

	workspaceRoot, _ := roots.Global.Validate(".")
	client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
	if err != nil {
		return errorResult(fmt.Sprintf("failed to start LSP server: %v", err)), nil, nil
	}

	edit, err := client.Rename(ctx, absPath, args.Line, args.Col, args.NewName)
	if err != nil {
		return errorResult(fmt.Sprintf("rename failed: %v", err)), nil, nil
	}
	if edit == nil {
		return errorResult("LSP returned no changes for rename"), nil, nil
	}

	// Apply WorkspaceEdit
	var resSb strings.Builder
	resSb.WriteString(fmt.Sprintf("### ✅ Rename Successful: %s -> %s\n", args.File, args.NewName))

	modifiedFiles := make(map[string]bool)

	// In Neko v0.2.0, we prioritize WorkspaceEdit.Changes
	for uri, edits := range edit.Changes {
		path := lsp.URIToPath(uri)
		content, err := os.ReadFile(path)
		if err != nil {
			resSb.WriteString(fmt.Sprintf("\n⚠️ Failed to read %s: %v", path, err))
			continue
		}

		newContent := lsp.ApplyTextEdits(string(content), edits)
		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			resSb.WriteString(fmt.Sprintf("\n⚠️ Failed to write %s: %v", path, err))
			continue
		}

		modifiedFiles[path] = true
		client.DidSave(ctx, path, newContent)
	}

	// Trigger diagnostics for all modified files
	for path := range modifiedFiles {
		client.WaitForDiagnostics(ctx, path)
	}

	// Pull final health
	allDiags := client.GetAllDiagnostics()
	workspaceRoot, _ = roots.Global.Validate(".")
	resSb.WriteString(lsp.FormatDiagnostics(allDiags, workspaceRoot))

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resSb.String()}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
