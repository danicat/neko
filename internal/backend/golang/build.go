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
	fmt.Fprintf(&sb, "# Smart Build Report (`%s`)\n\n", pkgs)

	// 1. Auto-Fix
	if opts.AutoFix {
		if err := runGoCmd(ctx, dir, "go", "mod", "tidy"); err != nil {
			fmt.Fprintf(&sb, "### ⚠️ Auto-Fix: `go mod tidy` Failed\n> %v\n\n", err)
		}
		if err := runGoCmd(ctx, dir, "gofmt", "-w", "."); err != nil {
			fmt.Fprintf(&sb, "### ⚠️ Auto-Fix: `gofmt` Failed\n> %v\n\n", err)
		}
	}

	// 2. Build
	sb.WriteString("### 🛠️ Build: ")

	buildArgs := []string{"build"}
	if opts.Output != "" {
		buildArgs = append(buildArgs, "-o", opts.Output)
	}
	buildArgs = append(buildArgs, pkgs)

	buildOut, buildErr := runGoCmdOutput(ctx, dir, "go", buildArgs...)
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
			sb.WriteString("❌ FAILED\n\n")
			sb.WriteString(goFormatOutput(modOut))
		} else if modOut == "" {
			sb.WriteString("✅ PASS (No issues found)\n\n")
		} else {
			sb.WriteString("📝 ISSUES FOUND\n\n")
			sb.WriteString(goFormatOutput(modOut))
			if opts.AutoFix {
				sb.WriteString("\n✅ Auto-fixed all modernization issues.\n\n")
			} else {
				sb.WriteString("\n⚠️ Fixes not applied (auto_fix disabled).\n\n")
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

		var coveredPkgs []string
		var zeroPkgs []string
		pkgMap := make(map[string]string)

		if funcErr == nil {
			lines := strings.SplitSeq(strings.TrimSpace(funcOut), "\n")
			for line := range lines {
				if strings.HasPrefix(line, "total:") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						sb.WriteString(fmt.Sprintf("- **Total Project Coverage**: %s\n", parts[len(parts)-1]))
					}
					continue
				}
				// go tool cover -func format:
				// path/to/file.go:line:	function	coverage%
				// We want to group by package
				parts := strings.Fields(line)
				if len(parts) < 3 {
					continue
				}
				filePart := parts[0] // path/to/file.go:line
				pkgPath := ""
				if found := strings.Contains(filePart, "/"); found {
					if lastSlash := strings.LastIndex(filePart, "/"); lastSlash != -1 {
						pkgPath = filePart[:lastSlash]
					}
				}
				if pkgPath == "" {
					pkgPath = "."
				}
				// This is a rough estimation since -func doesn't give per-package summary directly
				// But we'll use the go test summary for the actual package list if possible.
			}
		}

		// 3.1 Get full list of packages to ensure we report those without tests
		allPkgsOut, _ := runGoCmdOutput(ctx, dir, "go", "list", pkgs)
		allPkgs := strings.Split(strings.TrimSpace(allPkgsOut), "\n")

		// Use go test -v output to get precise per-package coverage and [no test files]
		lines := strings.SplitSeq(testOut, "\n")
		for line := range lines {
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
					pkgMap[pkg] = covStr
				}
			} else if strings.Contains(line, "[no test files]") {
				pkg := parts[1]
				pkgMap[pkg] = "0.0%"
			}
		}

		// Add missing packages from allPkgs to zeroPkgs
		for _, pkg := range allPkgs {
			if pkg == "" {
				continue
			}
			if _, seen := pkgMap[pkg]; !seen {
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
			// Deduplicate and sort zeroPkgs
			uniqueZero := make(map[string]bool)
			var sortedZero []string
			for _, p := range zeroPkgs {
				if !uniqueZero[p] {
					uniqueZero[p] = true
					sortedZero = append(sortedZero, p)
				}
			}
			// (Optional: sort sortedZero here if desired)

			sb.WriteString("- **Zero Coverage / No Tests**: ")
			for i, pkg := range sortedZero {
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
		lintOut, lintErr := runGoCmdOutput(ctx, dir, "go", "tool", "golangci-lint", "run", pkgs)
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
