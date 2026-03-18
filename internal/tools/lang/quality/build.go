// Package quality implements the smart_build tool.
package quality

import (
	"context"
	"fmt"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["smart_build"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return buildHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Dir      string `json:"dir,omitempty" jsonschema:"Directory to build in (default: current)"`
	Packages string `json:"packages,omitempty" jsonschema:"Packages to check (default: . or ./...)"`
	RunTests *bool  `json:"run_tests,omitempty" jsonschema:"Run unit tests (default: true)"`
	RunLint  *bool  `json:"run_lint,omitempty" jsonschema:"Run linter (default: true)"`
	AutoFix  *bool  `json:"auto_fix,omitempty" jsonschema:"Auto-fix format and lint issues (default: true)"`
}

func buildHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	dir := args.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := roots.Global.Validate(dir)
	if err != nil {
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

	be := reg.ForDir(absDir)
	if be == nil {
		return result("No language backend detected for this directory. Ensure the project has a recognizable project file (go.mod, pyproject.toml, etc.).", true), nil, nil
	}

	report, err := be.BuildPipeline(ctx, absDir, backend.BuildOpts{
		Packages: args.Packages,
		RunTests: runTests,
		RunLint:  runLint,
		AutoFix:  autoFix,
	})
	if err != nil {
		return result(fmt.Sprintf("build pipeline error: %v", err), true), nil, nil
	}

	return result(report.Output, report.IsError), nil, nil
}

func result(content string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: isError,
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}
}
