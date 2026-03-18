package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang/godoc"
)

func goInit(ctx context.Context, opts backend.InitOpts) error {
	absPath := opts.Path

	//nolint:gosec // G301
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(absPath, "go.mod")); err == nil {
		return fmt.Errorf("project already initialized (go.mod exists)")
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "init", opts.ModulePath)
	cmd.Dir = absPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to init module: %v\nOutput: %s", err, string(out))
	}

	if len(opts.Dependencies) > 0 {
		var depErrors []string
		for _, dep := range opts.Dependencies {
			cmd := exec.CommandContext(ctx, "go", "get", dep)
			cmd.Dir = absPath
			if out, err := cmd.CombinedOutput(); err != nil {
				depErrors = append(depErrors, fmt.Sprintf("go get %s: %s", dep, strings.TrimSpace(string(out))))
			}
		}

		tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
		tidyCmd.Dir = absPath
		if out, err := tidyCmd.CombinedOutput(); err != nil {
			depErrors = append(depErrors, fmt.Sprintf("go mod tidy: %s", strings.TrimSpace(string(out))))
		}

		if len(depErrors) > 0 {
			return fmt.Errorf("project initialized but dependency errors occurred:\n%s", strings.Join(depErrors, "\n"))
		}

		// Pre-fetch docs for dependencies so they're cached locally
		for _, dep := range opts.Dependencies {
			pkgPath := strings.Split(dep, "@")[0]
			godoc.GetDocumentationWithFallback(ctx, pkgPath)
		}
	}

	return nil
}
