// Package golang implements the LanguageBackend interface for Go.
package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang/godoc"
	"golang.org/x/tools/imports"
)

// Backend implements backend.LanguageBackend for Go.
type Backend struct{}

// New creates a new Go backend.
func New() *Backend {
	return &Backend{}
}

func (b *Backend) LSPCommand() (string, []string, bool) {
	return "gopls", nil, true
}

func (b *Backend) InitializationOptions() map[string]interface{} {
	return map[string]interface{}{
		"build.directoryFilters":        []string{"-node_modules", "-vendor"},
		"ui.completion.usePlaceholders": true,
		"ui.diagnostic.staticcheck":     true,
	}
}

func (b *Backend) Name() string             { return "go" }
func (b *Backend) FileExtensions() []string { return []string{".go"} }
func (b *Backend) ProjectMarkers() []string { return []string{"go.mod"} }
func (b *Backend) Tier() int                { return 3 }
func (b *Backend) SkipDirs() []string {
	return []string{"vendor", "testdata"}
}

func (b *Backend) Validate(ctx context.Context, filename string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("syntax error: %v", err)
	}
	return nil
}

func (b *Backend) Format(ctx context.Context, filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	formatted, err := imports.Process(filename, content, nil)
	if err != nil {
		return fmt.Errorf("goimports failed: %v", err)
	}
	if !bytes.Equal(content, formatted) {
		//nolint:gosec // G306
		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			return fmt.Errorf("failed to write formatted file: %w", err)
		}
	}
	return nil
}

func (b *Backend) Outline(ctx context.Context, filename string) (string, error) {
	return goOutline(filename)
}

func (b *Backend) ParseImports(ctx context.Context, filename string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to parse imports: %v", err)
	}
	var imports []string
	for _, imp := range f.Imports {
		if imp.Path != nil {
			pkgPath := strings.Trim(imp.Path.Value, "\"")
			parts := strings.Split(pkgPath, "/")
			// Third-party packages have a dot in the host component (github.com, golang.org, etc.)
			if len(parts) > 0 && strings.Contains(parts[0], ".") {
				imports = append(imports, imp.Path.Value)
			}
		}
	}
	return imports, nil
}

func (b *Backend) ImportDocs(ctx context.Context, importPaths []string) ([]string, error) {
	var docs []string
	for _, imp := range importPaths {
		pkgPath := strings.Trim(imp, "\"")
		// Skip standard library
		parts := strings.Split(pkgPath, "/")
		if len(parts) > 0 && !strings.Contains(parts[0], ".") {
			continue
		}
		d, err := godoc.Load(ctx, pkgPath, "")
		if err != nil {
			continue
		}
		summary := strings.ReplaceAll(d.Description, "\n", " ")
		if len(summary) > 200 {
			summary = summary[:197] + "..."
		}
		docs = append(docs, fmt.Sprintf("**%s**: %s", pkgPath, summary))
	}
	return docs, nil
}

func (b *Backend) BuildPipeline(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	return goBuild(ctx, dir, opts)
}

func (b *Backend) FetchDocs(ctx context.Context, dir string, pkg string, symbol string) (string, error) {
	doc, err := godoc.LoadWithFallback(ctx, pkg, symbol)
	if err != nil {
		return "", err
	}
	return godoc.Render(doc), nil
}

func (b *Backend) FetchDocsJSON(ctx context.Context, pkg string, symbol string) (string, error) {
	doc, err := godoc.LoadWithFallback(ctx, pkg, symbol)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}
	return string(data), nil
}

func (b *Backend) AddDependency(ctx context.Context, dir string, packages []string) (string, error) {
	return goAddDep(ctx, dir, packages)
}

func (b *Backend) InitProject(ctx context.Context, opts backend.InitOpts) error {
	return goInit(ctx, opts)
}

func (b *Backend) Modernize(ctx context.Context, dir string, fix bool) (string, error) {
	return goModernize(ctx, dir, fix)
}

func (b *Backend) MutationTest(ctx context.Context, dir string) (string, error) {
	return goMutationTest(ctx, dir)
}

func (b *Backend) BuildTestDB(ctx context.Context, dir string, pkg string) error {
	return goBuildTestDB(ctx, dir, pkg)
}

func (b *Backend) QueryTestDB(ctx context.Context, dir string, query string) (string, error) {
	return goQueryTestDB(ctx, dir, query)
}
