package instructions

import (
	"context"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/config"
)

// stubBackend is a minimal LanguageBackend for testing instructions generation.
type stubBackend struct {
	name string
}

func (s *stubBackend) Capabilities() []backend.Capability                                    { return nil }
func (s *stubBackend) Outline(_ context.Context, _ string) (string, error)                   { return "", nil }
func (s *stubBackend) ImportDocs(_ context.Context, _ []string) ([]string, error)             { return nil, nil }
func (s *stubBackend) ParseImports(_ context.Context, _ string) ([]string, error)             { return nil, nil }
func (s *stubBackend) Validate(_ context.Context, _ string) error                             { return nil }
func (s *stubBackend) Format(_ context.Context, _ string) error                               { return nil }
func (s *stubBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (s *stubBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (s *stubBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (s *stubBackend) InitProject(_ context.Context, _ backend.InitOpts) error                { return nil }
func (s *stubBackend) Modernize(_ context.Context, _ string, _ bool) (string, error)          { return "", nil }
func (s *stubBackend) MutationTest(_ context.Context, _ string) (string, error)               { return "", nil }
func (s *stubBackend) BuildTestDB(_ context.Context, _ string, _ string) error                { return nil }
func (s *stubBackend) QueryTestDB(_ context.Context, _ string, _ string) (string, error)      { return "", nil }
func (s *stubBackend) LSPCommand() (string, []string, bool)                                   { return "", nil, false }
func (s *stubBackend) InitializationOptions() map[string]any                                  { return nil }
func (s *stubBackend) EnsureTools(_ context.Context, _ string) error                          { return nil }
func (s *stubBackend) LanguageID() string                                                     { return s.name }
func (s *stubBackend) Name() string                                                           { return s.name }
func (s *stubBackend) FileExtensions() []string                                               { return nil }
func (s *stubBackend) SkipDirs() []string                                                     { return nil }
func (s *stubBackend) ProjectMarkers() []string                                               { return nil }
func (s *stubBackend) Tier() int                                                              { return 0 }
func (s *stubBackend) IsStdLibURI(_ string) bool                                              { return false }

func TestGet_HeaderWithBackends(t *testing.T) {
	reg := backend.NewRegistry()
	reg.Register(&stubBackend{name: "Go"})
	reg.Register(&stubBackend{name: "Python"})

	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	if !strings.Contains(result, "Go") || !strings.Contains(result, "Python") {
		t.Errorf("expected header to contain backend names, got:\n%s", result[:200])
	}
	if !strings.Contains(result, "# Neko Project Guide (") {
		t.Error("expected header with language list in parentheses")
	}
}

func TestGet_HeaderWithoutBackends(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	if !strings.Contains(result, "# Neko Project Guide\n") {
		t.Error("expected plain header without languages when no backends registered")
	}
	if strings.Contains(result, "# Neko Project Guide (") {
		t.Error("should not contain parenthesized language list when no backends are registered")
	}
}

func TestGet_AllToolsEnabled(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	// With no allowed/disabled filter, all tools should appear.
	for _, toolName := range []string{"read_file", "list_files", "edit_file", "create_file", "build", "describe", "find_definition", "find_references"} {
		if !strings.Contains(result, toolName) {
			t.Errorf("expected instruction to mention %q when all tools enabled", toolName)
		}
	}
}

func TestGet_DisabledToolOmitted(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{"read_file": true},
	}

	result := Get(cfg, reg)

	// The navigation section header should still exist.
	if !strings.Contains(result, "Navigation") {
		t.Error("expected Navigation section header")
	}

	// read_file instruction should not appear, but list_files should.
	if strings.Contains(result, "smart_read") && strings.Contains(result, "read_file") {
		// The instruction for read_file contains "read_file" -- check more carefully.
		// The read_file instruction block starts with "**`read_file`**".
		if strings.Contains(result, "**`read_file`**") {
			t.Error("expected read_file instruction to be omitted when disabled")
		}
	}
	if !strings.Contains(result, "list_files") {
		t.Error("expected list_files instruction to still be present")
	}
}

func TestGet_AllowListFiltersTools(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{"read_file": true},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	// Only read_file should be included from the conditional tools.
	if !strings.Contains(result, "**`read_file`**") {
		t.Error("expected read_file instruction when in allow list")
	}
	if strings.Contains(result, "**`edit_file`**") {
		t.Error("expected edit_file instruction to be omitted when not in allow list")
	}
}

func TestGet_ContainsProjectLifecycleSection(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	// Project lifecycle tools are always included (not gated by isEnabled).
	if !strings.Contains(result, "open_project") {
		t.Error("expected open_project instruction")
	}
	if !strings.Contains(result, "create_project") {
		t.Error("expected create_project instruction")
	}
	if !strings.Contains(result, "close_project") {
		t.Error("expected close_project instruction")
	}
}

func TestGet_ContainsAllSections(t *testing.T) {
	reg := backend.NewRegistry()
	cfg := &config.Config{
		AllowedTools:  map[string]bool{},
		DisabledTools: map[string]bool{},
	}

	result := Get(cfg, reg)

	sections := []string{
		"Project Lifecycle",
		"Navigation",
		"Editing",
		"Utilities",
		"Code Intelligence",
		"Testing",
	}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("expected section %q in output", section)
		}
	}
}
