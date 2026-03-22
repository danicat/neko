// Package create implements the create_file tool.
package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	IngestFile(ctx context.Context, path string, content string, symbols []lsp.DocumentSymbol, imports []string) error
	RAGEnabled() bool
	ProjectRoot() string
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
		return nil, nil, fmt.Errorf("name (file path) cannot be empty")
	}

	absPath, err := filepath.Abs(args.File)
	if err != nil {
		return nil, nil, err
	}
	if err := roots.Global.Validate(absPath); err != nil {
		return nil, nil, err
	}
	args.File = absPath

	finalContent := []byte(args.Content)

	//nolint:gosec // G301
	if err := os.MkdirAll(filepath.Dir(args.File), 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create directory: %w", err)
	}

	be := s.ForFile(ctx, args.File)
	var lspClient *lsp.Client
	if be != nil {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
			lspClient, _ = lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
		}
	}

	if lspClient != nil {
		// Prepare content (format/organize imports if possible before writing)
		if err := lspClient.DidOpen(ctx, args.File, args.Content); err != nil {
			return nil, nil, fmt.Errorf("LSP open failed: %w", err)
		}

		content := args.Content
		if edits, err := lspClient.OrganizeImports(ctx, args.File); err == nil && len(edits) > 0 {
			content = lsp.ApplyTextEdits(content, edits)
			if err := lspClient.DidChange(ctx, args.File, content); err != nil {
				return nil, nil, fmt.Errorf("LSP change failed after import organization: %w", err)
			}
		}
		if edits, err := lspClient.Format(ctx, args.File); err == nil && len(edits) > 0 {
			content = lsp.ApplyTextEdits(content, edits)
			if err := lspClient.DidChange(ctx, args.File, content); err != nil {
				return nil, nil, fmt.Errorf("LSP change failed after formatting: %w", err)
			}
		}
		finalContent = []byte(content)
	}

	// 1. Direct Write
	//nolint:gosec // G306
	if err := os.WriteFile(args.File, finalContent, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write file: %w", err)
	}

	if lspClient != nil {
		// 2. didChangeWatchedFiles (Trigger indexing for new file)
		if err := lspClient.DidChangeWatchedFiles(ctx, args.File, 1); err != nil {
			return nil, nil, fmt.Errorf("LSP file creation notification failed: %w", err)
		}

		// 3. didSave
		if err := lspClient.DidSave(ctx, args.File, string(finalContent)); err != nil {
			return nil, nil, fmt.Errorf("LSP save failed: %w", err)
		}

		// 4. WaitForDiagnostics
		if _, err := lspClient.WaitForDiagnostics(ctx, args.File); err != nil {
			return nil, nil, fmt.Errorf("LSP diagnostics wait failed: %w", err)
		}
	} else if be != nil {

		// Manual validation fallback for non-LSP backends or when LSP is down
		if err := be.Validate(ctx, args.File); err != nil {
			return nil, nil, err
		}
	}

	// Synchronous RAG Re-indexing
	if s.RAGEnabled() {
		var symbols []lsp.DocumentSymbol
		if lspClient != nil {
			symbols, _ = lspClient.DocumentSymbol(ctx, args.File)
		}
		var imports []string
		if be != nil {
			imports, _ = be.ParseImports(ctx, args.File)
		}
		if err := s.IngestFile(ctx, args.File, string(finalContent), symbols, imports); err != nil {
			return nil, nil, fmt.Errorf("RAG indexing failed: %w", err)
		}
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
		workspaceRoot := s.ProjectRoot()
		if workspaceRoot == "" {
			workspaceRoot, _ = filepath.Abs(".")
		}
		resSb.WriteString(lsp.FormatDiagnostics(allDiags, workspaceRoot))
	} else {
		resSb.WriteString("\n*Note: LSP unavailable. Global semantic verification skipped.*")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resSb.String()}},
	}, nil, nil
}
