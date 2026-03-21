package golang

import (
	"context"
	"fmt"
	"os/exec"
)

func goModernize(ctx context.Context, dir string, fix bool) (string, error) {
	checkCmd := exec.CommandContext(ctx, "go", "tool", "modernize", "./...")
	checkCmd.Dir = dir
	checkOut, checkErr := checkCmd.CombinedOutput()
	diagnostics := string(checkOut)

	if checkErr != nil && diagnostics == "" {
		return "", fmt.Errorf("modernize check failed to run: %v", checkErr)
	}

	diagnostics = filterNoise(diagnostics)

	if diagnostics == "" {
		return "", nil
	}

	if fix {
		fixCmd := exec.CommandContext(ctx, "go", "tool", "modernize", "-fix", "./...")
		fixCmd.Dir = dir
		fixOut, fixErr := fixCmd.CombinedOutput()
		if fixErr != nil {
			return string(fixOut), fmt.Errorf("modernization fix failed")
		}
		// Return the original diagnostics so the user knows what was fixed
		return diagnostics, nil
	}

	return diagnostics, nil
}
