package docs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	root    string
	backend backend.LanguageBackend
	err     error
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.backend
}

func (ts *testServer) ResolveBackend(language string) (backend.LanguageBackend, error) {
	if ts.err != nil {
		return nil, ts.err
	}
	if ts.backend == nil {
		return nil, fmt.Errorf("no backend available")
	}
	return ts.backend, nil
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

// mockBackend implements backend.LanguageBackend for testing.
type mockBackend struct {
	name    string
	docsOut string
	docsErr error
}

func (b *mockBackend) Name() string                 { return b.name }
func (b *mockBackend) LanguageID() string            { return b.name }
func (b *mockBackend) FileExtensions() []string      { return []string{".go"} }
func (b *mockBackend) ProjectMarkers() []string      { return nil }
func (b *mockBackend) SkipDirs() []string            { return nil }
func (b *mockBackend) Tier() int                     { return 1 }
func (b *mockBackend) IsStdLibURI(string) bool       { return false }
func (b *mockBackend) Capabilities() []backend.Capability { return nil }

func (b *mockBackend) Outline(_ context.Context, _ string) (string, error)       { return "", nil }
func (b *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error) { return nil, nil }
func (b *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (b *mockBackend) Validate(_ context.Context, _ string) error                { return nil }
func (b *mockBackend) Format(_ context.Context, _ string) error                  { return nil }
func (b *mockBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (b *mockBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return b.docsOut, b.docsErr
}
func (b *mockBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (b *mockBackend) InitProject(_ context.Context, _ backend.InitOpts) error { return nil }
func (b *mockBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}
func (b *mockBackend) MutationTest(_ context.Context, _ string) (string, error)      { return "", nil }
func (b *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error       { return nil }
func (b *mockBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (b *mockBackend) LSPCommand() (string, []string, bool) { return "", nil, false }
func (b *mockBackend) InitializationOptions() map[string]any { return nil }
func (b *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }

func resultText(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestDocsHandler_EmptyImportPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := &testServer{root: tmpDir, backend: &mockBackend{name: "go"}}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for empty import_path")
	}
	if !strings.Contains(resultText(res), "import_path is required") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_InvalidFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := &testServer{root: tmpDir, backend: &mockBackend{name: "go"}}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "net/http",
		Format:     "xml",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(resultText(res), "invalid format") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_NoBackend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := &testServer{root: tmpDir, backend: nil, err: fmt.Errorf("no backend for language")}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "net/http",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for nil backend")
	}
	if !strings.Contains(resultText(res), "no backend") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_FetchDocsFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsErr: fmt.Errorf("package not found"),
		},
	}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "nonexistent/package",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error when FetchDocs fails")
	}
	if !strings.Contains(resultText(res), "documentation lookup failed") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsOut: "# net/http\n\nHTTP client and server implementations.",
		},
	}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "net/http",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "net/http") {
		t.Errorf("expected docs content, got: %s", resultText(res))
	}
}

func TestDocsHandler_MarkdownFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsOut: "docs output",
		},
	}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "net/http",
		Format:     "markdown",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_DefaultDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsOut: "docs",
		},
	}
	// Empty dir should fall back to ProjectRoot
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "fmt",
		Dir:        "",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_DotDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsOut: "docs",
		},
	}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "fmt",
		Dir:        ".",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}

func TestDocsHandler_WithSymbol(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "docs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:    "go",
			docsOut: "func Println(a ...any)",
		},
	}
	res, _, err := docsHandler(context.Background(), nil, Params{
		ImportPath: "fmt",
		Symbol:     "Println",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "Println") {
		t.Errorf("expected symbol docs, got: %s", resultText(res))
	}
}
