// Package quality implements the build tool.
package quality

import (
	"context"
	"fmt"
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
	ResolveBackend(language string) (backend.LanguageBackend, error)
	ProjectRoot() string
}

// Register registers the build tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["build"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return buildHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters.
type Params struct {
	Dir          string `json:"dir,omitempty" jsonschema:"Directory to build in (default: current)"`
	Language     string `json:"language,omitempty" jsonschema:"Explicit language backend to use"`
	Packages     string `json:"packages,omitempty" jsonschema:"Packages to check (default: . or ./...)"`
	RunTests     *bool  `json:"run_tests,omitempty" jsonschema:"Run unit tests (default: true)"`
	RunLint      *bool  `json:"run_lint,omitempty" jsonschema:"Run linter (default: true)"`
	AutoFix      *bool  `json:"auto_fix,omitempty" jsonschema:"Auto-fix format and lint issues (default: true)"`
	RunModernize *bool  `json:"run_modernize,omitempty" jsonschema:"Run modernization check (default: true)"`
}

func buildHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
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
			return result(err.Error(), true), nil, nil
		}
	}

	if err := roots.Global.Validate(absDir); err != nil {
		return result(err.Error(), true), nil, nil
	}

	runTests := true
	if args.RunTests != nil {
		runTests = *args.RunTests
	}
	runLint := true
	if args.RunLint != nil {
		runLint = *args.RunLint
	}
	autoFix := true
	if args.AutoFix != nil {
		autoFix = *args.AutoFix
	}
	runModernize := true
	if args.RunModernize != nil {
		runModernize = *args.RunModernize
	}

	be, err := s.ResolveBackend(args.Language)
	if err != nil {
		return result(err.Error(), true), nil, nil
	}

	report, err := be.BuildPipeline(ctx, absDir, backend.BuildOpts{
		Packages:     args.Packages,
		RunTests:     runTests,
		RunLint:      runLint,
		AutoFix:      autoFix,
		RunModernize: runModernize,
	})
	if err != nil {
		return result(fmt.Sprintf("build pipeline error: %v", err), true), nil, nil
	}

	// LSP Sync if auto-fix was used
	if autoFix {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
			if lspClient, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions()); err == nil {
				// We don't know exactly which files changed, so we trigger a generic workspace update
				lspClient.DidChangeWatchedFiles(ctx, ".", 2) // 2: Changed

				// Pull diagnostics to include in the report
				lspClient.PullDiagnostics(ctx)
				workspaceRoot := s.ProjectRoot()
				if workspaceRoot == "" {
					workspaceRoot, _ = filepath.Abs(".")
				}
				report.Output += "\n---\n" + lsp.FormatDiagnostics(lspClient.GetAllDiagnostics(), workspaceRoot)
			}
		}
	}

	return result(report.Output, report.IsError), nil, nil
}

func result(content string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: isError,
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}
}
