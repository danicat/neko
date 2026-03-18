package python

import (
	"context"
	"fmt"
	"os/exec"
)

func pythonBuildTestDB(ctx context.Context, dir string, pkg string) error {
	if !hasUV() {
		return fmt.Errorf("uv is required but not found in PATH")
	}

	args := []string{
		"run", "pytest",
		"--json-report", "--json-report-file=.test-report.json",
		"--cov", "--cov-report=json:.coverage.json",
		"-v",
	}
	if pkg != "" {
		args = append(args, pkg)
	}

	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pytest failed: %s", string(out))
	}

	return nil
}
