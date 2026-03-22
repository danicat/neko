package definition

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
)

type testServer struct {
	root    string
	backend backend.LanguageBackend
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.backend
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

// noLSPBackend is a minimal backend with no LSP support.
type noLSPBackend struct {
	name string
}

func (b *noLSPBackend) Name() string                       { return b.name }
func (b *noLSPBackend) LanguageID() string                 { return b.name }
func (b *noLSPBackend) FileExtensions() []string           { return []string{".go"} }
func (b *noLSPBackend) ProjectMarkers() []string           { return nil }
func (b *noLSPBackend) SkipDirs() []string                 { return nil }
func (b *noLSPBackend) Tier() int                          { return 1 }
func (b *noLSPBackend) IsStdLibURI(string) bool            { return false }
func (b *noLSPBackend) Capabilities() []backend.Capability { return nil }

func (b *noLSPBackend) Outline(_ context.Context, _ string) (string, error)        { return "", nil }
func (b *noLSPBackend) ImportDocs(_ context.Context, _ []string) ([]string, error) { return nil, nil }
func (b *noLSPBackend) ParseImports(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (b *noLSPBackend) Validate(_ context.Context, _ string) error                 { return nil }
func (b *noLSPBackend) Format(_ context.Context, _ string) error                   { return nil }
func (b *noLSPBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (b *noLSPBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (b *noLSPBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (b *noLSPBackend) InitProject(_ context.Context, _ backend.InitOpts) error { return nil }
func (b *noLSPBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}
func (b *noLSPBackend) MutationTest(_ context.Context, _ string) (string, error) { return "", nil }
func (b *noLSPBackend) BuildTestDB(_ context.Context, _ string, _ string) error  { return nil }
func (b *noLSPBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (b *noLSPBackend) LSPCommand() (string, []string, bool)          { return "", nil, false }
func (b *noLSPBackend) InitializationOptions() map[string]any         { return nil }
func (b *noLSPBackend) EnsureTools(_ context.Context, _ string) error { return nil }

// lspBackend has an LSP command but one that won't be found in PATH.
type lspBackend struct {
	noLSPBackend
}

func (b *lspBackend) LSPCommand() (string, []string, bool) {
	return "nonexistent-lsp-server-binary-xyz", nil, true
}

func TestHandler_NoBackend(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "def-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{root: tmpDir, backend: nil}
	_, _, err = handler(context.Background(), Params{
		File: tmpDir + "/main.go",
		Line: 1,
		Col:  1,
	}, s)
	if err == nil {
		t.Fatal("expected error for nil backend")
	}
	if !strings.Contains(err.Error(), "no language backend") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_NoLSPConfigured(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "def-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{root: tmpDir, backend: &noLSPBackend{name: "test"}}
	_, _, err = handler(context.Background(), Params{
		File: tmpDir + "/main.go",
		Line: 1,
		Col:  1,
	}, s)
	if err == nil {
		t.Fatal("expected error for no LSP configured")
	}
	if !strings.Contains(err.Error(), "no LSP server configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_LSPNotInPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "def-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{root: tmpDir, backend: &lspBackend{noLSPBackend{name: "test"}}}
	_, _, err = handler(context.Background(), Params{
		File: tmpDir + "/main.go",
		Line: 1,
		Col:  1,
	}, s)
	if err == nil {
		t.Fatal("expected error for LSP not in PATH")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_OutsideRoots(t *testing.T) {
	s := &testServer{root: "/some/root", backend: &noLSPBackend{name: "test"}}
	_, _, err := handler(context.Background(), Params{
		File: "/outside/roots/main.go",

		Line: 1,
		Col:  1,
	}, s)
	if err == nil {
		t.Fatal("expected error for path outside roots")
	}
}

func TestHandler_EmptyFile(t *testing.T) {
	// When file is empty, it falls back to ProjectRoot
	tmpDir, err := os.MkdirTemp("", "def-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	roots.Global.Add(tmpDir)

	s := &testServer{root: tmpDir, backend: nil}
	_, _, err = handler(context.Background(), Params{

		File: "",
		Line: 1,
		Col:  1,
	}, s)
	if err == nil {
		t.Fatal("expected error for nil backend with empty file")
	}
	// With nil backend, should get "no language backend" error
	if !strings.Contains(err.Error(), "no language backend") {
		t.Errorf("unexpected error: %v", err)
	}
}
