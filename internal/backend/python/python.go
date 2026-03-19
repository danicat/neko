// Package python implements the LanguageBackend interface for Python.
package python

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
)

// Backend implements backend.LanguageBackend for Python.
type Backend struct{}

// New creates a new Python backend.
func New() *Backend {
	return &Backend{}
}

func (b *Backend) LSPCommand() (string, []string, bool) {
	if !hasUV() {
		return "", nil, false
	}
	return "uv", []string{"run", "pylsp"}, true
}

func (b *Backend) InitializationOptions() map[string]any {
	return nil
}

func (b *Backend) LanguageID() string { return "python" }
func (b *Backend) Name() string       { return "python" }

func (b *Backend) Capabilities() []backend.Capability {
	return []backend.Capability{
		backend.CapToolchain,
		backend.CapDocumentation,
		backend.CapDependencies,
		backend.CapModernize,
		backend.CapMutationTest,
		backend.CapLSP,
	}
}

func (b *Backend) FileExtensions() []string { return []string{".py", ".pyi"} }

func (b *Backend) ProjectMarkers() []string {
	return []string{"pyproject.toml", "setup.py", "requirements.txt", "setup.cfg", "Pipfile"}
}
func (b *Backend) Tier() int { return 3 }
func (b *Backend) SkipDirs() []string {
	return []string{"__pycache__", ".venv", "venv", ".mypy_cache", ".pytest_cache", ".ruff_cache", "*.egg-info", ".tox", ".nox", "dist", "build", ".eggs"}
}

func (b *Backend) Validate(ctx context.Context, filename string) error {
	cmd := exec.CommandContext(ctx, "python3", "-c",
		fmt.Sprintf("import ast; ast.parse(open(%q).read())", filename))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(out))
	}
	return nil
}

func (b *Backend) Format(ctx context.Context, filename string) error {
	if !hasUV() {
		return nil
	}
	dir := filepath.Dir(filename)
	cmd := exec.CommandContext(ctx, "uv", "run", "ruff", "format", filename)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ruff format failed: %s", string(out))
	}
	return nil
}

func (b *Backend) Outline(ctx context.Context, filename string) (string, error) {
	return pythonOutline(ctx, filename)
}

func (b *Backend) ParseImports(ctx context.Context, filename string) ([]string, error) {
	return pythonParseImports(ctx, filename)
}

func (b *Backend) ImportDocs(ctx context.Context, imports []string) ([]string, error) {
	var docs []string
	for i, imp := range imports {
		if i >= 10 {
			docs = append(docs, "... (more imports)")
			break
		}
		doc, err := b.FetchDocs(ctx, ".", imp, "")

		if err != nil {
			continue
		}
		summary := extractPydocSummary(doc)
		if summary != "" {
			docs = append(docs, fmt.Sprintf("**%s**: %s", imp, summary))
		}
	}
	return docs, nil
}

// extractPydocSummary extracts a concise description from pydoc output.
// pydoc format typically starts with:
//
//	NAME
//	    module_name - Short description here...
func extractPydocSummary(pydocOutput string) string {
	lines := strings.Split(pydocOutput, "\n")
	inName := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "NAME" {
			inName = true
			continue
		}
		if inName && trimmed != "" {
			// The NAME section line is like: "    flask - A microframework..."
			if _, after, ok := strings.Cut(trimmed, " - "); ok {
				desc := after
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				return desc
			}
			// No " - " separator means the module has no one-line description
			// Try DESCRIPTION section instead
			break
		}
		if inName && trimmed == "" {
			break
		}
	}

	// Fallback: look for DESCRIPTION section
	inDesc := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "DESCRIPTION" {
			inDesc = true
			continue
		}
		if inDesc && trimmed != "" {
			if len(trimmed) > 200 {
				trimmed = trimmed[:197] + "..."
			}
			return trimmed
		}
		if inDesc && trimmed == "" && strings.TrimSpace(line) == "" {
			break
		}
	}

	// Last resort: first non-empty content line
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "NAME" && trimmed != "DESCRIPTION" &&
			trimmed != "PACKAGE CONTENTS" && trimmed != "CLASSES" &&
			trimmed != "FUNCTIONS" && trimmed != "DATA" {
			if len(trimmed) > 200 {
				trimmed = trimmed[:197] + "..."
			}
			return trimmed
		}
	}
	return ""
}

func (b *Backend) BuildPipeline(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	return pythonBuild(ctx, dir, opts)
}

func (b *Backend) FetchDocs(ctx context.Context, dir string, pkg string, symbol string) (string, error) {
	if !hasUV() {
		return "", fmt.Errorf("uv is required but not found in PATH")
	}
	target := pkg
	if symbol != "" {
		target = pkg + "." + symbol
	}
	cmd := exec.CommandContext(ctx, "uv", "run", "python3", "-m", "pydoc", target)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pydoc failed: %s", string(out))
	}
	return string(out), nil
}

func (b *Backend) AddDependency(ctx context.Context, dir string, packages []string) (string, error) {
	return pythonAddDep(ctx, dir, packages)
}

func (b *Backend) InitProject(ctx context.Context, opts backend.InitOpts) error {
	return pythonInit(ctx, opts)
}

func (b *Backend) Modernize(ctx context.Context, dir string, fix bool) (string, error) {
	return pythonModernize(ctx, dir, fix)
}

func (b *Backend) MutationTest(ctx context.Context, dir string) (string, error) {
	return pythonMutationTest(ctx, dir)
}

func (b *Backend) BuildTestDB(ctx context.Context, dir string, pkg string) error {
	return pythonBuildTestDB(ctx, dir, pkg)
}

func (b *Backend) QueryTestDB(ctx context.Context, dir string, query string) (string, error) {
	return "", fmt.Errorf("SQL test querying is not yet supported for Python projects. Test data is collected via pytest JSON reports")
}
