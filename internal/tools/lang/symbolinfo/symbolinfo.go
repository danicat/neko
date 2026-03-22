// Package describe implements the describe tool for hover/type information via LSP.
package describe

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	HasSeenTypeInfo(name string) bool
	ProjectRoot() string
}

// Register registers the describe tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["describe"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return handler(ctx, args, s)
	})
}

// Params defines the input parameters for the describe tool.
type Params struct {
	File string `json:"file" jsonschema:"The path to the source file"`
	Line int    `json:"line" jsonschema:"Line number (1-based)"`
	Col  int    `json:"col" jsonschema:"Column number (1-based)"`
}

func handler(ctx context.Context, args Params, s Server) (*mcp.CallToolResult, any, error) {
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

	command, cmdArgs, ok := be.LSPCommand()
	if !ok {
		return nil, nil, fmt.Errorf("no LSP server configured for %s", be.Name())
	}

	if _, err := exec.LookPath(command); err != nil {
		return nil, nil, fmt.Errorf("LSP server %q not found in PATH", command)
	}

	workspaceRoot := s.ProjectRoot()
	if workspaceRoot == "" {
		workspaceRoot, _ = filepath.Abs(".")
	}
	client, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, command, cmdArgs, be.LanguageID(), be.InitializationOptions())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	hoverText, err := client.EnhancedHover(ctx, absPath, args.Line, args.Col)
	if err != nil {
		return nil, nil, fmt.Errorf("describe failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: hoverText}},
	}, nil, nil
}
