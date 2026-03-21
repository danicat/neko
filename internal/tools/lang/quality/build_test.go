package quality

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
	name        string
	buildReport *backend.BuildReport
	buildErr    error
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
func (b *mockBackend) BuildPipeline(_ context.Context, _ string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	return b.buildReport, b.buildErr
}
func (b *mockBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
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

func TestBuildHandler_OutsideRoots(t *testing.T) {
	s := &testServer{root: "/some/root", backend: &mockBackend{name: "go"}}
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir: "/outside/roots/path",
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for path outside roots")
	}
}

func TestBuildHandler_NoBackend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{root: tmpDir, backend: nil, err: fmt.Errorf("no backend available")}
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir: tmpDir,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for nil backend")
	}
	if !strings.Contains(resultText(res), "no backend available") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestBuildHandler_BuildPipelineError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name:     "go",
			buildErr: fmt.Errorf("compilation failed"),
		},
	}
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir: tmpDir,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for build pipeline failure")
	}
	if !strings.Contains(resultText(res), "build pipeline error") {
		t.Errorf("unexpected error: %s", resultText(res))
	}
}

func TestBuildHandler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name: "go",
			buildReport: &backend.BuildReport{
				Output:  "all tests passed",
				IsError: false,
			},
		},
	}
	f := false
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir:     tmpDir,
		AutoFix: &f, // disable auto-fix to avoid LSP sync
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "all tests passed") {
		t.Errorf("expected success output, got: %s", resultText(res))
	}
}

func TestBuildHandler_BuildReportsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name: "go",
			buildReport: &backend.BuildReport{
				Output:  "2 tests failed",
				IsError: true,
			},
		},
	}
	f := false
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir:     tmpDir,
		AutoFix: &f,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected IsError when build report has errors")
	}
	if !strings.Contains(resultText(res), "2 tests failed") {
		t.Errorf("expected failure output, got: %s", resultText(res))
	}
}

func TestBuildHandler_DefaultDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name: "go",
			buildReport: &backend.BuildReport{
				Output:  "ok",
				IsError: false,
			},
		},
	}
	f := false
	// Empty dir should fall back to ProjectRoot
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir:     "",
		AutoFix: &f,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}

func TestBuildHandler_BoolDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "build-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{
		root: tmpDir,
		backend: &mockBackend{
			name: "go",
			buildReport: &backend.BuildReport{
				Output:  "ok",
				IsError: false,
			},
		},
	}
	// All bool pointers nil = defaults to true; auto_fix=true will try LSP sync
	// but since our mockBackend has no LSP, the LSP sync is skipped gracefully.
	res, _, err := buildHandler(context.Background(), nil, Params{
		Dir: tmpDir,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}
