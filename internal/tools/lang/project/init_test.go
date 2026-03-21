package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockBackend implements backend.LanguageBackend for testing.
type mockBackend struct {
	name       string
	initErr    error
	extensions []string
	markers    []string
}

func (m *mockBackend) Name() string                 { return m.name }
func (m *mockBackend) LanguageID() string            { return m.name }
func (m *mockBackend) FileExtensions() []string      { return m.extensions }
func (m *mockBackend) ProjectMarkers() []string      { return m.markers }
func (m *mockBackend) SkipDirs() []string            { return nil }
func (m *mockBackend) Tier() int                     { return 1 }
func (m *mockBackend) IsStdLibURI(string) bool       { return false }
func (m *mockBackend) Capabilities() []backend.Capability { return nil }

func (m *mockBackend) Outline(_ context.Context, _ string) (string, error)       { return "", nil }
func (m *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error) { return nil, nil }
func (m *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (m *mockBackend) Validate(_ context.Context, _ string) error                { return nil }
func (m *mockBackend) Format(_ context.Context, _ string) error                  { return nil }
func (m *mockBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (m *mockBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *mockBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (m *mockBackend) InitProject(_ context.Context, _ backend.InitOpts) error {
	return m.initErr
}
func (m *mockBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) { return "", nil }
func (m *mockBackend) MutationTest(_ context.Context, _ string) (string, error)      { return "", nil }
func (m *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error       { return nil }
func (m *mockBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (m *mockBackend) LSPCommand() (string, []string, bool) { return "", nil, false }
func (m *mockBackend) InitializationOptions() map[string]any { return nil }
func (m *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }

func resultText(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestInitHandler_EmptyDir(t *testing.T) {
	reg := backend.NewRegistry()
	res, _, err := InitHandler(context.Background(), Params{Dir: ""}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for empty dir")
	}
	if !strings.Contains(resultText(res), "dir is required") {
		t.Errorf("unexpected error message: %s", resultText(res))
	}
}

func TestInitHandler_OutsideRoots(t *testing.T) {
	reg := backend.NewRegistry()
	// Use a path that is not in any registered root
	res, _, err := InitHandler(context.Background(), Params{Dir: "/nonexistent/path"}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for path outside roots")
	}
}

func TestInitHandler_UnknownLanguage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	res, _, err := InitHandler(context.Background(), Params{
		Dir:      tmpDir,
		Language: "brainfuck",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(resultText(res), "unknown language") {
		t.Errorf("unexpected error message: %s", resultText(res))
	}
}

func TestInitHandler_NoBackendAvailable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	res, _, err := InitHandler(context.Background(), Params{
		Dir:        tmpDir,
		ModulePath: "test-module",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for no backend available")
	}
	if !strings.Contains(resultText(res), "No language backend available") {
		t.Errorf("unexpected error message: %s", resultText(res))
	}
}

func TestInitHandler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}, markers: []string{"go.mod"}})

	res, _, err := InitHandler(context.Background(), Params{
		Dir:        tmpDir,
		ModulePath: "github.com/test/project",
		Language:   "go",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "Successfully initialized") {
		t.Errorf("expected success message, got: %s", resultText(res))
	}
}

func TestInitHandler_InitFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(&mockBackend{
		name:       "go",
		extensions: []string{".go"},
		markers:    []string{"go.mod"},
		initErr:    os.ErrPermission,
	})

	res, _, err := InitHandler(context.Background(), Params{
		Dir:        tmpDir,
		ModulePath: "github.com/test/project",
		Language:   "go",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error when init fails")
	}
	if !strings.Contains(resultText(res), "project initialization failed") {
		t.Errorf("unexpected error message: %s", resultText(res))
	}
}

func TestInitHandler_DefaultModulePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}, markers: []string{"go.mod"}})

	// When ModulePath is empty, it defaults to Dir
	res, _, err := InitHandler(context.Background(), Params{
		Dir:      filepath.Join(tmpDir, "myproject"),
		Language: "go",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(res))
	}
}

func TestInitHandler_MultipleBackendsDetected(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	// Create markers for both backends
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("flask\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}, markers: []string{"go.mod"}})
	reg.Register(&mockBackend{name: "python", extensions: []string{".py"}, markers: []string{"requirements.txt"}})

	res, _, err := InitHandler(context.Background(), Params{
		Dir:        tmpDir,
		ModulePath: "test",
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error for multiple backends detected")
	}
	if !strings.Contains(resultText(res), "multiple languages detected") {
		t.Errorf("unexpected error message: %s", resultText(res))
	}
}
