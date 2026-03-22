package get

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
	be         backend.LanguageBackend
	resolveErr error
	root       string
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.be
}

func (ts *testServer) ResolveBackend(_ string) (backend.LanguageBackend, error) {
	if ts.resolveErr != nil {
		return nil, ts.resolveErr
	}
	return ts.be, nil
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

type mockBackend struct {
	name    string
	addOut  string
	addErr  error
	docsOut string
	docsErr error
}

func (b *mockBackend) Name() string                          { return b.name }
func (b *mockBackend) LSPCommand() (string, []string, bool)  { return "", nil, false }
func (b *mockBackend) FileExtensions() []string              { return []string{".go"} }
func (b *mockBackend) SkipDirs() []string                    { return nil }
func (b *mockBackend) ProjectMarkers() []string              { return nil }
func (b *mockBackend) Tier() int                             { return 0 }
func (b *mockBackend) IsStdLibURI(_ string) bool             { return false }
func (b *mockBackend) LanguageID() string                    { return "go" }
func (b *mockBackend) InitializationOptions() map[string]any { return nil }
func (b *mockBackend) Capabilities() []backend.Capability    { return nil }

func (b *mockBackend) Outline(_ context.Context, _ string) (string, error)        { return "", nil }
func (b *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error) { return nil, nil }
func (b *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (b *mockBackend) Validate(_ context.Context, _ string) error                 { return nil }
func (b *mockBackend) Format(_ context.Context, _ string) error                   { return nil }
func (b *mockBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (b *mockBackend) FetchDocs(_ context.Context, _ string, pkg string, _ string) (string, error) {
	if b.docsErr != nil {
		return "", b.docsErr
	}
	return b.docsOut, nil
}
func (b *mockBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return b.addOut, b.addErr
}
func (b *mockBackend) InitProject(_ context.Context, _ backend.InitOpts) error { return nil }
func (b *mockBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}
func (b *mockBackend) MutationTest(_ context.Context, _ string) (string, error) { return "", nil }
func (b *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error  { return nil }
func (b *mockBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (b *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }

func textContent(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestGetHandler_EmptyPackages(t *testing.T) {
	s := &testServer{root: "/tmp/test"}

	_, _, err := getHandler(context.TODO(), nil, Params{Packages: nil}, s)
	if err == nil {
		t.Fatal("expected error for empty packages")
	}
	if !strings.Contains(err.Error(), "at least one package is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetHandler_ResolveBackendError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "get-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	s := &testServer{resolveErr: fmt.Errorf("unsupported language"), root: tmpDir}

	_, _, err = getHandler(context.TODO(), nil, Params{Packages: []string{"foo"}, Dir: tmpDir}, s)
	if err == nil {
		t.Fatal("expected error for resolve failure")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetHandler_AddDependencyError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "get-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", addErr: fmt.Errorf("network error")}
	s := &testServer{be: be, root: tmpDir}

	_, _, err = getHandler(context.TODO(), nil, Params{Packages: []string{"foo"}, Dir: tmpDir}, s)
	if err == nil {
		t.Fatal("expected error for add dependency failure")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetHandler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "get-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{
		name:    "test",
		addOut:  "go get: added github.com/foo/bar v1.0.0",
		docsOut: "Package bar provides utilities.",
	}
	s := &testServer{be: be, root: tmpDir}

	res, _, err := getHandler(context.TODO(), nil, Params{Packages: []string{"github.com/foo/bar"}, Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = res
	text := textContent(res)

	if !strings.Contains(text, "added github.com/foo/bar") {
		t.Errorf("expected add output in result, got: %s", text)
	}
	if !strings.Contains(text, "Documentation") {
		t.Errorf("expected Documentation section in result, got: %s", text)
	}
	if !strings.Contains(text, "Package bar provides utilities") {
		t.Errorf("expected docs in result, got: %s", text)
	}
}

func TestGetHandler_DefaultDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "get-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", addOut: "ok"}
	s := &testServer{be: be, root: tmpDir}

	// Empty dir should use project root
	res, _, err := getHandler(context.TODO(), nil, Params{Packages: []string{"foo"}}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = res
}
