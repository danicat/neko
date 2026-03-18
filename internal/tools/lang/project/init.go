// Package project implements the project_init tool.
package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the project_init tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["project_init"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return initHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Path         string   `json:"path" jsonschema:"Directory path for the new project"`
	ModulePath   string   `json:"module_path" jsonschema:"Module/package name (e.g. github.com/user/repo or my-app)"`
	Language     string   `json:"language,omitempty" jsonschema:"Language to use (go, python). Auto-detected if omitted."`
	Dependencies []string `json:"dependencies,omitempty" jsonschema:"Optional: list of dependencies to install"`
}

func initHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	if args.Path == "" {
		return errorResult("path is required"), nil, nil
	}
	if args.ModulePath == "" {
		args.ModulePath = args.Path
	}

	absPath, err := roots.Global.Validate(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	// Determine backend: explicit language or try to detect from existing dir
	var be backend.LanguageBackend
	if args.Language != "" {
		be = reg.Get(args.Language)
		if be == nil {
			return errorResult(fmt.Sprintf("unknown language: %s. Available: %v", args.Language, reg.Available())), nil, nil
		}
	} else {
		be = reg.ForDir(absPath)
		if be == nil {
			// Default to Go if available
			be = reg.Get("go")
			if be == nil {
				be = reg.Get("python")
			}
		}
	}

	if be == nil {
		return errorResult("No language backend available for project initialization."), nil, nil
	}

	err = be.InitProject(ctx, backend.InitOpts{
		Path:         absPath,
		ModulePath:   args.ModulePath,
		Dependencies: args.Dependencies,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("project initialization failed: %v", err)), nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Successfully initialized %s project at `%s`", be.Name(), absPath))

	if len(args.Dependencies) > 0 {
		sb.WriteString("\n\n## Dependencies Documentation\n")
		for _, dep := range args.Dependencies {
			doc, err := be.FetchDocs(ctx, absPath, dep, "")
			if err == nil && doc != "" {
				sb.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", dep, doc))
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
