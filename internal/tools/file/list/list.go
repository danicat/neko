// Package list implements the list_files tool.
package list

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["list_files"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return listHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters.
type Params struct {
	Path  string `json:"path" jsonschema:"The root path to list (default: .)"`
	Depth int    `json:"depth,omitempty" jsonschema:"Maximum recursion depth (0 for default of 5, 1 for non-recursive)"`
}

func listHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	absRoot, err := roots.Global.Validate(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	maxDepth := args.Depth
	if maxDepth == 0 {
		maxDepth = 5
	}

	if result, ok := tryGitLsFiles(ctx, absRoot, maxDepth); ok {
		return result, nil, nil
	}

	skipDirs := defaultSkipDirs()
	skipDirs = append(skipDirs, reg.AllSkipDirs()...)
	return walkDir(absRoot, maxDepth, skipDirs)
}

func tryGitLsFiles(ctx context.Context, absRoot string, maxDepth int) (*mcp.CallToolResult, bool) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = absRoot
	if _, err := cmd.Output(); err != nil {
		return nil, false
	}

	cmd = exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = absRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Listing files in %s (Depth: %d, git-aware)\n\n", absRoot, maxDepth))

	fileCount := 0
	dirsSeen := make(map[string]bool)
	const maxFiles = 1000

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		depth := strings.Count(line, "/") + 1
		if depth > maxDepth {
			continue
		}
		if fileCount >= maxFiles {
			sb.WriteString(fmt.Sprintf("\n(Limit of %d files reached, output truncated)\n", maxFiles))
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
			}, true
		}

		dir := filepath.Dir(line)
		if dir != "." {
			parts := strings.Split(dir, "/")
			for i := range parts {
				d := strings.Join(parts[:i+1], "/")
				if !dirsSeen[d] {
					dirsSeen[d] = true
					sb.WriteString(fmt.Sprintf("%s/\n", d))
				}
			}
		}
		sb.WriteString(fmt.Sprintf("%s\n", line))
		fileCount++
	}

	sb.WriteString(fmt.Sprintf("\nFound %d files, %d directories.\n", fileCount, len(dirsSeen)))
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, true
}

func walkDir(absRoot string, maxDepth int, skipDirs []string) (*mcp.CallToolResult, any, error) {
	skipSet := make(map[string]bool)
	for _, d := range skipDirs {
		skipSet[d] = true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Listing files in %s (Depth: %d)\n\n", absRoot, maxDepth))

	fileCount := 0
	dirCount := 0
	limitReached := false
	const maxFiles = 1000

	err := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			sb.WriteString(fmt.Sprintf("Warning: skipping %s: %v\n", path, err))
			return nil
		}

		relPath, _ := filepath.Rel(absRoot, path)
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(os.PathSeparator)) + 1
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() && (skipSet[d.Name()] || d.Name() == ".git" || d.Name() == ".idea" || d.Name() == ".vscode" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}

		if fileCount >= maxFiles {
			limitReached = true
			return filepath.SkipAll
		}

		if d.IsDir() {
			sb.WriteString(fmt.Sprintf("%s/\n", relPath))
			dirCount++
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", relPath))
			fileCount++
		}

		return nil
	})

	if err != nil {
		sb.WriteString(fmt.Sprintf("\nError walking: %v\n", err))
	}

	if limitReached {
		sb.WriteString(fmt.Sprintf("\n(Limit of %d files reached, output truncated)\n", maxFiles))
	} else {
		sb.WriteString(fmt.Sprintf("\nFound %d files, %d directories.\n", fileCount, dirCount))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func defaultSkipDirs() []string {
	return []string{"__pycache__", ".venv", "venv", ".mypy_cache", ".pytest_cache", ".ruff_cache", ".tox", ".nox", "dist", "build", ".eggs"}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
