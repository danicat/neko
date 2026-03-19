package python

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/danicat/neko/internal/backend"
)

func pythonBuild(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	var sb strings.Builder
	sb.WriteString("# Smart Build Report (Python)\n\n")
	isError := false

	if !hasUV() {
		return &backend.BuildReport{
			Output:  "uv is required but not found in PATH",
			IsError: true,
		}, nil
	}

	if opts.AutoFix {
		out, err := runCmd(ctx, dir, "uv", "sync")
		if err != nil {
			sb.WriteString(fmt.Sprintf("### ⚠️ Auto-Fix: `uv sync` Failed\n%s\n\n", formatOutput(out)))
		}
	}

	sb.WriteString("### 🎨 Format: ")
	out, err := runCmd(ctx, dir, "uv", "run", "ruff", "format", "--check", ".")
	if err != nil {
		sb.WriteString("❌ FAILED\n\n")
		sb.WriteString(formatOutput(out))
		if opts.AutoFix {
			fixOut, fixErr := runCmd(ctx, dir, "uv", "run", "ruff", "format", ".")
			if fixErr == nil {
				sb.WriteString("\n✅ Auto-fixed formatting issues.\n\n")
			} else {
				sb.WriteString(fmt.Sprintf("\n⚠️ Auto-fix failed: %s\n\n", formatOutput(fixOut)))
				isError = true
			}
		} else {
			isError = true
		}
	} else {
		sb.WriteString("✅ PASS\n\n")
	}

	if opts.RunLint {
		sb.WriteString("### 🧹 Lint: ")
		out, err := runCmd(ctx, dir, "uv", "run", "ruff", "check", ".")
		if err != nil {
			sb.WriteString("❌ ISSUES FOUND\n\n")
			sb.WriteString(formatOutput(out))
			if opts.AutoFix {
				fixOut, fixErr := runCmd(ctx, dir, "uv", "run", "ruff", "check", "--fix", ".")
				if fixErr == nil {
					sb.WriteString("\n✅ Auto-fixed lint issues.\n\n")
				} else {
					sb.WriteString(fmt.Sprintf("\n⚠️ Some issues remain: %s\n\n", formatOutput(fixOut)))
				}
			}
			isError = true
		} else {
			sb.WriteString("✅ PASS\n\n")
		}
	}

	sb.WriteString("### 🔍 Type Check: ")
	out, err = runCmd(ctx, dir, "uv", "run", "mypy", ".")
	if err != nil {
		sb.WriteString("⚠️ ISSUES FOUND\n\n")
		sb.WriteString(formatOutput(out))
	} else {
		sb.WriteString("✅ PASS\n\n")
	}

	if opts.RunModernize {
		sb.WriteString("### 🚀 Modernize: ")
		modOut, modErr := pythonModernize(ctx, dir, opts.AutoFix)
		if modErr != nil {
			sb.WriteString("⚠️ FAILED\n\n")
			sb.WriteString(formatOutput(modErr.Error()))
		} else if strings.Contains(modOut, "No outdated patterns found") || modOut == "" {
			sb.WriteString("✅ PASS\n\n")
		} else {
			sb.WriteString("📝 ISSUES FOUND\n\n")
			sb.WriteString(formatOutput(modOut))
			if opts.AutoFix {
				sb.WriteString("\n✅ Auto-fixed modernization issues.\n\n")
			}
		}
	}

	if opts.RunTests {
		sb.WriteString("### 🧪 Tests: ")
		// First try to collect to see if there are any tests
		out, collectErr := runCmd(ctx, dir, "uv", "run", "pytest", "-v", "--tb=short", "--co", "-q")
		if collectErr != nil || strings.Contains(out, "no tests ran") || strings.Contains(out, "collected 0 items") {
			sb.WriteString("⏭️ NO TESTS FOUND\n\n")
		} else {
			testOut, testErr := runCmd(ctx, dir, "uv", "run", "pytest", "-v", "--tb=short")
			if testErr != nil {
				sb.WriteString("❌ FAILED\n\n")
				sb.WriteString(formatOutput(testOut))
				isError = true
			} else {
				sb.WriteString("✅ PASS\n\n")
				lines := strings.Split(testOut, "\n")
				for _, line := range lines {
					if strings.Contains(line, "passed") || strings.Contains(line, "failed") {
						sb.WriteString(fmt.Sprintf("Summary: %s\n", strings.TrimSpace(line)))
						break
					}
				}
			}
		}
	}

	return &backend.BuildReport{
		Output:  sb.String(),
		IsError: isError,
	}, nil
}

func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func formatOutput(out string) string {
	if out == "" {
		return ""
	}
	return "```text\n" + strings.TrimSpace(out) + "\n```\n"
}

func hasUV() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}
