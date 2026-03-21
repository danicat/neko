package backend

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// mockBackend implements LanguageBackend for testing.
type mockBackend struct {
	name           string
	extensions     []string
	skipDirs       []string
	projectMarkers []string
	tier           int
}

func (m *mockBackend) Capabilities() []Capability                                    { return nil }
func (m *mockBackend) Outline(_ context.Context, _ string) (string, error)           { return "", nil }
func (m *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error)    { return nil, nil }
func (m *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error)    { return nil, nil }
func (m *mockBackend) Validate(_ context.Context, _ string) error                    { return nil }
func (m *mockBackend) Format(_ context.Context, _ string) error                      { return nil }
func (m *mockBackend) BuildPipeline(_ context.Context, _ string, _ BuildOpts) (*BuildReport, error) {
	return nil, nil
}
func (m *mockBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *mockBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (m *mockBackend) InitProject(_ context.Context, _ InitOpts) error              { return nil }
func (m *mockBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) { return "", nil }
func (m *mockBackend) MutationTest(_ context.Context, _ string) (string, error)      { return "", nil }
func (m *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error       { return nil }
func (m *mockBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (m *mockBackend) LSPCommand() (string, []string, bool)   { return "", nil, false }
func (m *mockBackend) InitializationOptions() map[string]any   { return nil }
func (m *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }
func (m *mockBackend) LanguageID() string                      { return m.name }
func (m *mockBackend) Name() string                            { return m.name }
func (m *mockBackend) FileExtensions() []string                { return m.extensions }
func (m *mockBackend) SkipDirs() []string                      { return m.skipDirs }
func (m *mockBackend) ProjectMarkers() []string                { return m.projectMarkers }
func (m *mockBackend) Tier() int                               { return m.tier }
func (m *mockBackend) IsStdLibURI(_ string) bool               { return false }

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(reg.Available()) != 0 {
		t.Errorf("expected empty registry, got %d backends", len(reg.Available()))
	}
}

func TestRegister_And_Available(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}})
	reg.Register(&mockBackend{name: "python", extensions: []string{".py"}})

	names := reg.Available()
	if len(names) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(names))
	}

	// Available returns sorted names.
	expected := []string{"go", "python"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected name %q at index %d, got %q", expected[i], i, name)
		}
	}
}

func TestRegister_OverwritesSameName(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}, tier: 1})
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}, tier: 2})

	names := reg.Available()
	if len(names) != 1 {
		t.Fatalf("expected 1 backend after overwrite, got %d", len(names))
	}

	be := reg.Get("go")
	if be.Tier() != 2 {
		t.Errorf("expected tier 2 after overwrite, got %d", be.Tier())
	}
}

func TestForFile_MatchesExtension(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}})
	reg.Register(&mockBackend{name: "python", extensions: []string{".py"}})

	be := reg.ForFile("main.go")
	if be == nil {
		t.Fatal("expected a backend for main.go, got nil")
	}
	if be.Name() != "go" {
		t.Errorf("expected 'go' backend, got %q", be.Name())
	}

	be = reg.ForFile("script.py")
	if be == nil {
		t.Fatal("expected a backend for script.py, got nil")
	}
	if be.Name() != "python" {
		t.Errorf("expected 'python' backend, got %q", be.Name())
	}
}

func TestForFile_NoMatch(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}})

	be := reg.ForFile("readme.txt")
	if be != nil {
		t.Errorf("expected nil for unmatched extension, got %q", be.Name())
	}
}

func TestForFile_HigherTierWins(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "basic-go", extensions: []string{".go"}, tier: 1})
	reg.Register(&mockBackend{name: "advanced-go", extensions: []string{".go"}, tier: 2})

	be := reg.ForFile("main.go")
	if be == nil {
		t.Fatal("expected a backend, got nil")
	}
	if be.Name() != "advanced-go" {
		t.Errorf("expected higher-tier backend 'advanced-go', got %q", be.Name())
	}
}

func TestForFile_CaseInsensitive(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", extensions: []string{".go"}})

	be := reg.ForFile("Main.GO")
	if be == nil {
		t.Fatal("expected a backend for case-insensitive match, got nil")
	}
}

func TestDetectBackends(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "detect-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	// Create go.mod marker file.
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", projectMarkers: []string{"go.mod"}})
	reg.Register(&mockBackend{name: "python", projectMarkers: []string{"pyproject.toml"}})

	matched := reg.DetectBackends(tmpDir)
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched backend, got %d", len(matched))
	}
	if matched[0].Name() != "go" {
		t.Errorf("expected 'go' backend detected, got %q", matched[0].Name())
	}
}

func TestDetectBackends_NoMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "detect-none-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", projectMarkers: []string{"go.mod"}})

	matched := reg.DetectBackends(tmpDir)
	if len(matched) != 0 {
		t.Errorf("expected 0 matched backends, got %d", len(matched))
	}
}

func TestAllSkipDirs(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go", skipDirs: []string{"vendor", "node_modules"}})
	reg.Register(&mockBackend{name: "python", skipDirs: []string{"__pycache__", "node_modules"}})

	dirs := reg.AllSkipDirs()
	sort.Strings(dirs)

	// node_modules should appear only once (deduped).
	expected := []string{"__pycache__", "node_modules", "vendor"}
	if len(dirs) != len(expected) {
		t.Fatalf("expected %d skip dirs, got %d: %v", len(expected), len(dirs), dirs)
	}
	for i, d := range dirs {
		if d != expected[i] {
			t.Errorf("expected %q at index %d, got %q", expected[i], i, d)
		}
	}
}

func TestAllSkipDirs_Empty(t *testing.T) {
	reg := NewRegistry()
	dirs := reg.AllSkipDirs()
	if len(dirs) != 0 {
		t.Errorf("expected 0 skip dirs for empty registry, got %d", len(dirs))
	}
}

func TestGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockBackend{name: "go"})

	be := reg.Get("go")
	if be == nil {
		t.Fatal("expected backend, got nil")
	}

	be = reg.Get("nonexistent")
	if be != nil {
		t.Error("expected nil for nonexistent backend")
	}
}
