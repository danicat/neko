package python

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func pythonModernize(ctx context.Context, dir string, fix bool) (string, error) {
	if !hasUV() {
		return "", fmt.Errorf("uv is required but not found in PATH")
	}

	args := []string{"run", "ruff", "check", "--select", "UP", "."}
	if fix {
		args = []string{"run", "ruff", "check", "--select", "UP", "--fix", "."}
	}

	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil && !fix {
		return fmt.Sprintf("# Modernization Report (pyupgrade rules)\n\n%s\n\nRun with `fix=true` to auto-fix these issues.", output), nil
	}

	if fix {
		if output == "" {
			return "# Modernization Report\n\n✅ No outdated patterns found. Code is already modern.", nil
		}
		return fmt.Sprintf("# Modernization Report\n\n✅ Applied modernization fixes:\n\n%s", output), nil
	}

	if output == "" {
		return "# Modernization Report\n\n✅ No outdated patterns found. Code is already modern.", nil
	}

	return fmt.Sprintf("# Modernization Report (pyupgrade rules)\n\n%s", output), nil
}
