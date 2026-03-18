package python

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func pythonAddDep(ctx context.Context, dir string, packages []string) (string, error) {
	var sb strings.Builder

	if !hasUV() {
		return "", fmt.Errorf("uv is required but not found in PATH")
	}

	args := append([]string{"add"}, packages...)
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("uv add failed: %s", string(out))
	}
	sb.WriteString(fmt.Sprintf("Successfully added packages via `uv add`: %s\n\n", strings.Join(packages, ", ")))
	sb.WriteString(string(out))

	sb.WriteString("\n## Package Documentation\n\n")
	for _, pkg := range packages {

		name := pkg
		for _, sep := range []string{">=", "<=", "==", "~=", "!=", ">", "<", "[", "@"} {
			if idx := strings.Index(name, sep); idx > 0 {
				name = name[:idx]
			}
		}
		cmd := exec.CommandContext(ctx, "python3", "-m", "pydoc", name)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			sb.WriteString(fmt.Sprintf("### %s\n*Documentation not available: %v*\n\n", pkg, err))
			continue
		}
		docStr := strings.TrimSpace(string(out))
		if docStr != "" && len(docStr) > 2000 {
			docStr = docStr[:1997] + "..."
		}
		if docStr != "" {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", pkg, docStr))
		}
	}

	return sb.String(), nil
}
