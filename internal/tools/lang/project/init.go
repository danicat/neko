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

// Register registers the create_project tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["create_project"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return InitHandler(ctx, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Dir          string   `json:"dir" jsonschema:"Directory path for the new project"`
	ModulePath   string   `json:"module_path" jsonschema:"Module/package name (e.g. github.com/user/repo or my-app)"`
	Language     string   `json:"language,omitempty" jsonschema:"Language to use (go, python). Auto-detected if omitted."`
	Dependencies []string `json:"dependencies,omitempty" jsonschema:"Optional: list of dependencies to install"`
}

func InitHandler(ctx context.Context, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	if args.Dir == "" {

		return errorResult("dir is required"), nil, nil
	}
	if args.ModulePath == "" {
		args.ModulePath = args.Dir
	}

	absPath, err := roots.Global.Validate(args.Dir)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	// Determine backend: explicit language or detect from project markers
	var be backend.LanguageBackend
	if args.Language != "" {
		be = reg.Get(args.Language)
		if be == nil {
			return errorResult(fmt.Sprintf("unknown language: %q. Supported: %v", args.Language, reg.Available())), nil, nil
		}
	} else {
		// Try to detect from existing markers in the directory
		detected := reg.DetectBackends(absPath)
		if len(detected) == 1 {
			be = detected[0]
		} else if len(detected) == 0 {
			// Default to Go if available, then Python
			be = reg.Get("go")
			if be == nil {
				be = reg.Get("python")
			}
		} else {
			var names []string
			for _, d := range detected {
				names = append(names, d.Name())
			}
			return errorResult(fmt.Sprintf("multiple languages detected (%s), please specify the 'language' parameter", strings.Join(names, ", "))), nil, nil
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
