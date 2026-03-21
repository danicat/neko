package mutation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	be       backend.LanguageBackend
	resolveErr error
	root     string
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
	name          string
	mutationOut   string
	mutationErr   error
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

func (b *mockBackend) Outline(_ context.Context, _ string) (string, error)           { return "", nil }
func (b *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error)    { return nil, nil }
func (b *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error)    { return nil, nil }
func (b *mockBackend) Validate(_ context.Context, _ string) error                    { return nil }
func (b *mockBackend) Format(_ context.Context, _ string) error                      { return nil }
func (b *mockBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
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
func (b *mockBackend) MutationTest(_ context.Context, _ string) (string, error) {
	return b.mutationOut, b.mutationErr
}
func (b *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error         { return nil }
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

func TestMutationHandler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mutation-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", mutationOut: "Mutation score: 85%"}
	s := &testServer{be: be, root: tmpDir}

	res, _, err := mutationHandler(context.TODO(), nil, Params{Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "Mutation score: 85%") {
		t.Errorf("unexpected output: %s", textContent(res))
	}
}

func TestMutationHandler_ResolveBackendError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mutation-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	s := &testServer{resolveErr: fmt.Errorf("no backend found"), root: tmpDir}

	res, _, err := mutationHandler(context.TODO(), nil, Params{Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error for resolve failure")
	}
	if !strings.Contains(textContent(res), "no backend found") {
		t.Errorf("unexpected error: %s", textContent(res))
	}
}

func TestMutationHandler_MutationTestError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mutation-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", mutationErr: fmt.Errorf("gremlins crashed")}
	s := &testServer{be: be, root: tmpDir}

	res, _, err := mutationHandler(context.TODO(), nil, Params{Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error for mutation test failure")
	}
	if !strings.Contains(textContent(res), "gremlins crashed") {
		t.Errorf("unexpected error: %s", textContent(res))
	}
}

func TestMutationHandler_DefaultDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mutation-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", mutationOut: "ok"}
	s := &testServer{be: be, root: tmpDir}

	// Empty dir should use project root
	res, _, err := mutationHandler(context.TODO(), nil, Params{}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(res))
	}
}

func TestMutationHandler_ExplicitDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mutation-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", mutationOut: "ok"}
	s := &testServer{be: be, root: tmpDir}

	res, _, err := mutationHandler(context.TODO(), nil, Params{Dir: subDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", textContent(res))
	}
}
