package python

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/danicat/neko/internal/backend"
)

func pythonInit(ctx context.Context, opts backend.InitOpts) error {
	projectDir := opts.Path
	if projectDir == "" {
		return fmt.Errorf("project path is required")
	}

	if !hasUV() {
		return fmt.Errorf("uv is required for Python projects but not found in PATH")
	}

	//nolint:gosec // G301
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Run uv init to set up the project
	if out, err := runCmd(ctx, projectDir, "uv", "init", "--no-workspace"); err != nil {
		return fmt.Errorf("uv init failed: %s", strings.TrimSpace(out))
	}

	// Add essential dev dependencies
	devDeps := []string{"ruff", "mypy", "pytest", "pytest-json-report", "pytest-cov"}
	args := append([]string{"add", "--dev"}, devDeps...)
	if out, err := runCmd(ctx, projectDir, "uv", args...); err != nil {
		return fmt.Errorf("failed to add dev dependencies: %s", strings.TrimSpace(out))
	}

	// Add dependencies if any
	if len(opts.Dependencies) > 0 {

		args := append([]string{"add"}, opts.Dependencies...)
		if out, err := runCmd(ctx, projectDir, "uv", args...); err != nil {
			return fmt.Errorf("uv add failed: %s", strings.TrimSpace(out))
		}
	}

	// Ensure we have a standard structure
	if out, err := runCmd(ctx, projectDir, "uv", "sync"); err != nil {
		return fmt.Errorf("uv sync failed: %s", strings.TrimSpace(out))
	}

	return nil
}
