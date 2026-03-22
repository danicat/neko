package rename

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
)

type testServer struct {
	be   backend.LanguageBackend
	root string
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.be
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

type mockBackend struct {
	name       string
	lspOk      bool
	lspCommand string
}

func (b *mockBackend) Name() string                          { return b.name }
func (b *mockBackend) LSPCommand() (string, []string, bool)  { return b.lspCommand, nil, b.lspOk }
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
func (b *mockBackend) MutationTest(_ context.Context, _ string) (string, error) { return "", nil }
func (b *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error  { return nil }
func (b *mockBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (b *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }

func TestRenameHandler_NilBackend(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "rename-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	s := &testServer{be: nil, root: tmpDir}
	filePath := filepath.Join(tmpDir, "main.go")

	_, _, err = renameHandler(context.TODO(), Params{File: filePath, Line: 1, Col: 1, NewName: "foo"}, s)
	if err == nil {
		t.Fatal("expected error for nil backend")
	}
	if !strings.Contains(err.Error(), "no language backend") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenameHandler_NoLSP(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rename-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", lspOk: false}
	s := &testServer{be: be, root: tmpDir}
	filePath := filepath.Join(tmpDir, "main.go")

	_, _, err = renameHandler(context.TODO(), Params{File: filePath, Line: 1, Col: 1, NewName: "foo"}, s)
	if err == nil {
		t.Fatal("expected error for no LSP")
	}
	if !strings.Contains(err.Error(), "no LSP server configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenameHandler_EmptyFileDefaultsToProjectRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rename-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	s := &testServer{be: nil, root: tmpDir}

	_, _, err = renameHandler(context.TODO(), Params{File: "", Line: 1, Col: 1, NewName: "foo"}, s)
	if err == nil {
		t.Fatal("expected error for nil backend")
	}
	if !strings.Contains(err.Error(), "no language backend") {
		t.Errorf("unexpected error: %v", err)
	}
}
