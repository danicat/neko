// Package rename implements the rename_symbol tool.
package rename

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
	ProjectRoot() string
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
	var absPath string
	if args.File == "" || args.File == "." {
		absPath = s.ProjectRoot()
		if absPath == "" {
			absPath, _ = filepath.Abs(".")
		}
	} else {
		var err error
		absPath, err = filepath.Abs(args.File)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := roots.Global.Validate(absPath); err != nil {
		return nil, nil, err
	}

	be := s.ForFile(ctx, absPath)
	if be == nil {
		return nil, nil, fmt.Errorf("no language backend for %s", absPath)
	}

	cmd, cmdArgs, ok := be.LSPCommand()
	if !ok {
		return nil, nil, fmt.Errorf("no LSP server configured for %s", be.Name())
	}

	workspaceRoot := s.ProjectRoot()
	if workspaceRoot == "" {
		workspaceRoot, _ = filepath.Abs(".")
	}
	client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	edit, err := client.Rename(ctx, absPath, args.Line, args.Col, args.NewName)
	if err != nil {
		return nil, nil, fmt.Errorf("rename failed: %w", err)
	}
	if edit == nil {
		return nil, nil, fmt.Errorf("LSP returned no changes for rename")
	}

	// Apply WorkspaceEdit
	var resSb strings.Builder
	resSb.WriteString(fmt.Sprintf("### ✅ Rename Successful: %s -> %s\n", args.File, args.NewName))

	modifiedFiles := make(map[string]bool)

	// Collect edits: prefer DocumentChanges (gopls default), fall back to Changes
	fileEdits := make(map[string][]lsp.TextEdit)

	if len(edit.DocumentChanges) > 0 {
		for _, raw := range edit.DocumentChanges {
			var tde lsp.TextDocumentEdit
			if err := json.Unmarshal(raw, &tde); err != nil {
				continue
			}
			uri := tde.TextDocument.URI
			fileEdits[uri] = append(fileEdits[uri], tde.Edits...)
		}
	} else {
		maps.Copy(fileEdits, edit.Changes)
	}

	if len(fileEdits) == 0 {
		return nil, nil, fmt.Errorf("rename produced no changes")
	}

	for uri, edits := range fileEdits {
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
		if err := client.DidSave(ctx, path, newContent); err != nil {
			return nil, nil, fmt.Errorf("LSP save failed for %s: %w", path, err)
		}
	}

	// Trigger diagnostics for all modified files
	for path := range modifiedFiles {
		if _, err := client.WaitForDiagnostics(ctx, path); err != nil {
			return nil, nil, fmt.Errorf("LSP diagnostics wait failed for %s: %w", path, err)
		}
	}

	// Pull final health
	allDiags := client.GetAllDiagnostics()
	workspaceRoot = s.ProjectRoot()
	if workspaceRoot == "" {
		workspaceRoot, _ = filepath.Abs(".")
	}
	resSb.WriteString(lsp.FormatDiagnostics(allDiags, workspaceRoot))

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resSb.String()}},
	}, nil, nil
}
