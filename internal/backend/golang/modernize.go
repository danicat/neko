package golang

import (
	"context"
	"fmt"
	"os/exec"
)

func goModernize(ctx context.Context, dir string, fix bool) (string, error) {
	toolPath := "golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest"

	checkCmd := exec.CommandContext(ctx, "go", "run", toolPath, "./...")
	checkCmd.Dir = dir
	checkOut, checkErr := checkCmd.CombinedOutput()
	diagnostics := string(checkOut)

	if checkErr != nil && diagnostics == "" {
		return "", fmt.Errorf("modernize check failed to run: %v", checkErr)
	}

	diagnostics = filterNoise(diagnostics)

	if diagnostics == "" {
		return "✅ No modernization issues found.", nil
	}

	if fix {
		fixCmd := exec.CommandContext(ctx, "go", "run", toolPath, "-fix", "./...")
		fixCmd.Dir = dir
		fixOut, fixErr := fixCmd.CombinedOutput()
		if fixErr != nil {
			return fmt.Sprintf("⚠️ Modernization fix attempted but encountered errors:\n\n%s", string(fixOut)), nil
		}
		return fmt.Sprintf("⚠️ Found modernization opportunities:\n\n%s\n\n✅ Automatically applied modernization fixes.", diagnostics), nil
	}

	return fmt.Sprintf("⚠️ Found modernization opportunities:\n\n%s", diagnostics), nil
}
