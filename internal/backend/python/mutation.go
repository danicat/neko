package python

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func pythonMutationTest(ctx context.Context, dir string) (string, error) {
	if !hasUV() {
		return "", fmt.Errorf("uv is required but not found in PATH")
	}

	// Try uv run mutmut
	_, err := runCmd(ctx, dir, "uv", "run", "mutmut", "--version")
	if err != nil {
		return "", fmt.Errorf("mutmut is required for mutation testing but not found in project (uv run mutmut failed).\nInstall it with: uv add mutmut --dev")
	}

	cmd := exec.CommandContext(ctx, "uv", "run", "mutmut", "run", "--no-progress")
	cmd.Dir = dir
	out, runErr := cmd.CombinedOutput()

	resultsCmd := exec.CommandContext(ctx, "uv", "run", "mutmut", "results")
	resultsCmd.Dir = dir
	results, resultsErr := resultsCmd.CombinedOutput()

	var sb strings.Builder

	sb.WriteString("# Mutation Testing Report\n\n")
	sb.WriteString("## Run Output\n")
	if runErr != nil {
		sb.WriteString(fmt.Sprintf("(mutmut exited with error: %v)\n", runErr))
	}
	sb.WriteString(fmt.Sprintf("```text\n%s\n```\n\n", strings.TrimSpace(string(out))))
	sb.WriteString("## Results\n")
	if resultsErr != nil {
		sb.WriteString(fmt.Sprintf("(mutmut results exited with error: %v)\n", resultsErr))
	}
	sb.WriteString(fmt.Sprintf("```text\n%s\n```\n", strings.TrimSpace(string(results))))

	return sb.String(), nil
}
