// Package create implements the create_file tool.
package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/rag"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	RAG() *rag.Engine
}

// Register registers the create_file tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["create_file"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return createHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	File    string `json:"file" jsonschema:"The path to the file to create"`
	Content string `json:"content" jsonschema:"The content to write"`
}

func createHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	if args.File == "" {
		return errorResult("name (file path) cannot be empty"), nil, nil
	}

	absPath, err := roots.Global.Validate(args.File)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}
	args.File = absPath

	finalContent := []byte(args.Content)

	//nolint:gosec // G301
	if err := os.MkdirAll(filepath.Dir(args.File), 0755); err != nil {
		return errorResult(fmt.Sprintf("failed to create directory: %v", err)), nil, nil
	}

	be := s.ForFile(ctx, args.File)
	var lspClient *lsp.Client
	if be != nil {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot, _ := roots.Global.Validate(".")
			lspClient, _ = lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
		}
	}

	if lspClient != nil {
		// Prepare content (format/organize imports if possible before writing)
		lspClient.DidOpen(ctx, args.File, args.Content)

		content := args.Content
		if edits, err := lspClient.OrganizeImports(ctx, args.File); err == nil && len(edits) > 0 {
			content = lsp.ApplyTextEdits(content, edits)
			lspClient.DidChange(ctx, args.File, content)
		}
		if edits, err := lspClient.Format(ctx, args.File); err == nil && len(edits) > 0 {
			content = lsp.ApplyTextEdits(content, edits)
			lspClient.DidChange(ctx, args.File, content)
		}
		finalContent = []byte(content)
	}

	// 1. Direct Write
	//nolint:gosec // G306
	if err := os.WriteFile(args.File, finalContent, 0644); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), nil, nil
	}

	var warning string
	if lspClient != nil {
		// 2. didChangeWatchedFiles (Trigger indexing for new file)
		lspClient.DidChangeWatchedFiles(ctx, args.File, 1) // 1: Created

		// 3. didSave
		lspClient.DidSave(ctx, args.File, string(finalContent))

		// 4. WaitForDiagnostics
		lspClient.WaitForDiagnostics(ctx, args.File)
	} else if be != nil {
		// Manual validation fallback for non-LSP backends or when LSP is down
		if err := be.Validate(ctx, args.File); err != nil {
			warning = fmt.Sprintf("\n\n**WARNING:** Post-write syntax check failed: %v", err)
		}
	}

	// Synchronous RAG Re-indexing
	if engine := s.RAG(); engine != nil {
		var symbols []lsp.DocumentSymbol
		if lspClient != nil {
			symbols, _ = lspClient.DocumentSymbol(ctx, args.File)
		}
		var imports []string
		if be != nil {
			imports, _ = be.ParseImports(ctx, args.File)
		}
		engine.IngestFile(ctx, args.File, string(finalContent), symbols, imports)
	}

	if lspClient != nil {
		// 5. didClose
		lspClient.DidClose(ctx, args.File)
	}

	// Standardized Markdown Response
	var resSb strings.Builder
	resSb.WriteString(fmt.Sprintf("### ✅ File Created: %s\n", args.File))

	if lspClient != nil {
		allDiags := lspClient.GetAllDiagnostics()
		workspaceRoot, _ := roots.Global.Validate(".")
		resSb.WriteString(lsp.FormatDiagnostics(allDiags, workspaceRoot))
	} else {
		resSb.WriteString("\n*Note: LSP unavailable. Global semantic verification skipped.*")
		if warning != "" {
			resSb.WriteString(warning)
		}
	}

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
