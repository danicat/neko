package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/danicat/neko/internal/backend"
)

func goBuild(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	pkgs := opts.Packages
	if pkgs == "" {
		pkgs = "./..."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Smart Build Report (`%s`)\n\n", pkgs))

	// 1. Auto-Fix
	if opts.AutoFix {
		if err := runGoCmd(ctx, dir, "go", "mod", "tidy"); err != nil {
			sb.WriteString(fmt.Sprintf("### ⚠️ Auto-Fix: `go mod tidy` Failed\n> %v\n\n", err))
		}
		_ = runGoCmd(ctx, dir, "gofmt", "-w", ".")
	}

	// 2. Build
	sb.WriteString("### 🛠️ Build: ")
	buildOut, buildErr := runGoCmdOutput(ctx, dir, "go", "build", pkgs)
	if buildErr != nil {
		sb.WriteString("❌ FAILED\n\n")
		sb.WriteString(goFormatOutput(buildOut))
		sb.WriteString(GetDocHintFromOutput(buildOut))
		return &backend.BuildReport{Output: sb.String(), IsError: true}, nil
	}
	sb.WriteString("✅ PASS\n\n")

	// 2.1 Modernize
	if opts.RunModernize {
		sb.WriteString("### 🚀 Modernize: ")
		modOut, modErr := goModernize(ctx, dir, opts.AutoFix)
		if modErr != nil {
			sb.WriteString("⚠️ FAILED\n\n")
			sb.WriteString(goFormatOutput(modOut))
		} else if strings.Contains(modOut, "No issues found") || modOut == "" {
			sb.WriteString("✅ PASS\n\n")
		} else {
			sb.WriteString("📝 ISSUES FOUND\n\n")
			sb.WriteString(goFormatOutput(modOut))
			if opts.AutoFix {
				sb.WriteString("\n✅ Auto-fixed modernization issues.\n\n")
			}
		}
	}

	// 3. Tests
	if opts.RunTests {
		sb.WriteString("### 🧪 Tests: ")
		covFile := "coverage.out"
		defer func() { _ = os.Remove(covFile) }()

		testArgs := []string{"test", "-v", "-coverprofile=" + covFile, pkgs}
		testOut, testErr := runGoCmdOutput(ctx, dir, "go", testArgs...)
		if testErr != nil {
			sb.WriteString("❌ FAILED\n\n")
			sb.WriteString(goFormatOutput(testOut))
			return &backend.BuildReport{Output: sb.String(), IsError: true}, nil
		}
		sb.WriteString("✅ PASS\n\n")

		sb.WriteString("#### Coverage\n")
		funcOut, funcErr := runGoCmdOutput(ctx, dir, "go", "tool", "cover", "-func="+covFile)
		if funcErr == nil {
			lines := strings.Split(strings.TrimSpace(funcOut), "\n")
			if len(lines) > 0 {
				lastLine := lines[len(lines)-1]
				if strings.HasPrefix(lastLine, "total:") {
					parts := strings.Fields(lastLine)
					if len(parts) >= 3 {
						sb.WriteString(fmt.Sprintf("- **Total Project Coverage**: %s\n", parts[len(parts)-1]))
					}
				}
			}
		}

		lines := strings.Split(testOut, "\n")
		var coveredPkgs []string
		var zeroPkgs []string

		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			if strings.Contains(line, "\tcoverage: ") {
				covIdx := -1
				for i, p := range parts {
					if p == "coverage:" {
						covIdx = i
						break
					}
				}

				if covIdx > 1 {
					pkg := parts[1]
					covStr := ""
					if covIdx+1 < len(parts) {
						covStr = parts[covIdx+1]
					}

					if covStr == "0.0%" {
						zeroPkgs = append(zeroPkgs, pkg)
					} else if covStr != "" {
						coveredPkgs = append(coveredPkgs, fmt.Sprintf("  - `%s`: %s", pkg, covStr))
					}
				}
			} else if strings.Contains(line, "[no test files]") {
				pkg := parts[1]
				zeroPkgs = append(zeroPkgs, pkg)
			}
		}

		if len(coveredPkgs) > 0 {
			sb.WriteString("- **Packages**:\n")
			for _, p := range coveredPkgs {
				sb.WriteString(p + "\n")
			}
		}

		if len(zeroPkgs) > 0 {
			sb.WriteString("- **Zero Coverage / No Tests**: ")
			for i, pkg := range zeroPkgs {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("`%s`", pkg))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// 4. Lint
	if opts.RunLint {
		sb.WriteString("### 🧹 Lint: ")
		lintCmd := "golangci-lint"
		lintArgs := []string{"run", pkgs}
		if _, err := exec.LookPath("golangci-lint"); err != nil {
			lintCmd = "go"
			lintArgs = []string{"vet", pkgs}
			sb.WriteString("(using `go vet`) ")
		}
		lintOut, lintErr := runGoCmdOutput(ctx, dir, lintCmd, lintArgs...)
		if lintErr != nil {
			sb.WriteString("⚠️ ISSUES FOUND\n\n")
			sb.WriteString(goFormatOutput(lintOut))
			return &backend.BuildReport{Output: sb.String(), IsError: true}, nil
		}
		sb.WriteString("✅ PASS\n")
	}

	return &backend.BuildReport{Output: sb.String(), IsError: false}, nil
}

func runGoCmd(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

func runGoCmdOutput(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func goFormatOutput(out string) string {
	if out == "" {
		return ""
	}
	return "```text\n" + strings.TrimSpace(out) + "\n```\n"
}
