package golang

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func goMutationTest(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "tool", "selene", "./...")
	cmd.Dir = dir
	out, runErr := cmd.CombinedOutput()

	output := filterNoise(string(out))

	if runErr != nil && output == "" {
		return "", fmt.Errorf("mutation testing failed to run: %v", runErr)
	}

	if output == "" {
		return "✅ All mutations were caught by tests.", nil
	}

	if runErr != nil {
		return fmt.Sprintf("🧬 Mutation testing results:\n\n%s", output), nil
	}

	return fmt.Sprintf("✅ Mutation testing results:\n\n%s", output), nil
}

func filterNoise(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, "exit status") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
