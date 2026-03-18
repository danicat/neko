package golang

import (
	"context"
	"fmt"
	"os/exec"
)

func goAddDep(ctx context.Context, dir string, packages []string) (string, error) {
	cmdArgs := []string{"get"}
	cmdArgs = append(cmdArgs, packages...)

	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf("go get failed: %v", err)
	}
	return string(output), nil
}
