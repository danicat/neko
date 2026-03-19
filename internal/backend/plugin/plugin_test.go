package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginValidate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  Plugin
		wantErr bool
	}{
		{
			name: "valid plugin",
			plugin: Plugin{
				Name:       "rust",
				Extensions: []string{".rs"},
				Tier:       2,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			plugin: Plugin{
				Name:       "",
				Extensions: []string{".rs"},
				Tier:       2,
			},
			wantErr: true,
		},
		{
			name: "no extensions",
			plugin: Plugin{
				Name:       "rust",
				Extensions: []string{},
				Tier:       2,
			},
			wantErr: true,
		},
		{
			name: "tier too high",
			plugin: Plugin{
				Name:       "rust",
				Extensions: []string{".rs"},
				Tier:       3,
			},
			wantErr: true,
		},
		{
			name: "negative tier",
			plugin: Plugin{
				Name:       "rust",
				Extensions: []string{".rs"},
				Tier:       -1,
			},
			wantErr: true,
		},
		{
			name: "languageId defaults to name",
			plugin: Plugin{
				Name:       "rust",
				Extensions: []string{".rs"},
				Tier:       1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.plugin.LanguageID == "" {
				t.Error("Validate() should set LanguageID when empty")
			}
		})
	}
}

func TestExpandArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		vars     map[string]string
		expected []string
	}{
		{
			name:     "simple substitution",
			args:     []string{"--file", "{{filename}}"},
			vars:     map[string]string{"filename": "/tmp/test.go"},
			expected: []string{"--file", "/tmp/test.go"},
		},
		{
			name:     "variadic expansion",
			args:     []string{"install", "{{packages...}}", "--save"},
			vars:     map[string]string{"packages...": "react express lodash"},
			expected: []string{"install", "react", "express", "lodash", "--save"},
		},
		{
			name:     "no matching placeholder",
			args:     []string{"build", "{{dir}}"},
			vars:     map[string]string{"other": "value"},
			expected: []string{"build", "{{dir}}"},
		},
		{
			name:     "empty variadic",
			args:     []string{"install", "{{packages...}}"},
			vars:     map[string]string{"packages...": ""},
			expected: []string{"install"},
		},
		{
			name:     "multiple substitutions in one arg",
			args:     []string{"{{dir}}/{{filename}}"},
			vars:     map[string]string{"dir": "/home", "filename": "test.go"},
			expected: []string{"/home/test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandArgs(tt.args, tt.vars)
			if len(result) != len(tt.expected) {
				t.Fatalf("expandArgs() got %d args %v, want %d args %v", len(result), result, len(tt.expected), tt.expected)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expandArgs()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestLoadPlugins_ValidDir(t *testing.T) {
	dir := t.TempDir()
	//nolint:gosec // G306
	err := os.WriteFile(filepath.Join(dir, "rust.json"), []byte(`{
		"name": "rust",
		"languageId": "rust",
		"extensions": [".rs"],
		"projectMarkers": ["Cargo.toml"],
		"skipDirs": ["target"],
		"tier": 2,
		"lsp": {"command": "rust-analyzer", "args": []},
		"commands": {}
	}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	backends, err := LoadPlugins(dir)
	if err != nil {
		t.Fatalf("LoadPlugins() error = %v", err)
	}
	if len(backends) != 1 {
		t.Fatalf("LoadPlugins() got %d backends, want 1", len(backends))
	}
	if backends[0].Name() != "rust" {
		t.Errorf("backend name = %q, want %q", backends[0].Name(), "rust")
	}
	if backends[0].LanguageID() != "rust" {
		t.Errorf("backend languageID = %q, want %q", backends[0].LanguageID(), "rust")
	}
}

func TestLoadPlugins_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	//nolint:gosec // G306
	err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{not valid json`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadPlugins(dir)
	if err == nil {
		t.Error("LoadPlugins() should return error for invalid JSON")
	}
}

func TestLoadPlugins_InvalidPlugin(t *testing.T) {
	dir := t.TempDir()
	//nolint:gosec // G306
	err := os.WriteFile(filepath.Join(dir, "empty.json"), []byte(`{
		"name": "",
		"extensions": [".rs"],
		"tier": 2
	}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadPlugins(dir)
	if err == nil {
		t.Error("LoadPlugins() should return error for invalid plugin (empty name)")
	}
}

func TestLoadPlugins_NonExistentDir(t *testing.T) {
	backends, err := LoadPlugins("/nonexistent/dir")
	if err != nil {
		t.Errorf("LoadPlugins() for nonexistent dir should return nil, got error: %v", err)
	}
	if backends != nil {
		t.Errorf("LoadPlugins() for nonexistent dir should return nil backends")
	}
}

func TestLoadPlugins_SkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	//nolint:gosec // G306
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a plugin"), 0644)
	//nolint:gosec // G306
	os.WriteFile(filepath.Join(dir, "rust.json"), []byte(`{
		"name": "rust",
		"extensions": [".rs"],
		"tier": 1,
		"commands": {}
	}`), 0644)

	backends, err := LoadPlugins(dir)
	if err != nil {
		t.Fatalf("LoadPlugins() error = %v", err)
	}
	if len(backends) != 1 {
		t.Fatalf("LoadPlugins() got %d backends, want 1", len(backends))
	}
}

func TestPluginBackend_Metadata(t *testing.T) {
	p := &Plugin{
		Name:           "typescript",
		LanguageID:     "typescript",
		Extensions:     []string{".ts", ".tsx"},
		ProjectMarkers: []string{"tsconfig.json"},
		SkipDirs:       []string{"node_modules"},
		Tier:           2,
		LSP:            &LSPConfig{Command: "ts-server", Args: []string{"--stdio"}},
	}
	be := NewBackend(p)

	if be.Name() != "typescript" {
		t.Errorf("Name() = %q", be.Name())
	}
	if be.LanguageID() != "typescript" {
		t.Errorf("LanguageID() = %q", be.LanguageID())
	}
	if be.Tier() != 2 {
		t.Errorf("Tier() = %d", be.Tier())
	}
	cmd, args, ok := be.LSPCommand()
	if !ok || cmd != "ts-server" || len(args) != 1 || args[0] != "--stdio" {
		t.Errorf("LSPCommand() = %q, %v, %v", cmd, args, ok)
	}
}

func TestPluginBackend_NoLSP(t *testing.T) {
	p := &Plugin{
		Name:       "plain",
		Extensions: []string{".txt"},
		Tier:       0,
	}
	be := NewBackend(p)

	_, _, ok := be.LSPCommand()
	if ok {
		t.Error("LSPCommand() should return false when no LSP configured")
	}
}

func TestPluginBackend_OptionalCommands(t *testing.T) {
	p := &Plugin{
		Name:       "minimal",
		Extensions: []string{".min"},
		Tier:       1,
		Commands:   map[string]CommandConfig{},
	}
	be := NewBackend(p)

	// Validate and Format should return nil when not configured
	if err := be.Validate(nil, "/tmp/test.min"); err != nil {
		t.Errorf("Validate() should return nil when not configured, got: %v", err)
	}
	if err := be.Format(nil, "/tmp/test.min"); err != nil {
		t.Errorf("Format() should return nil when not configured, got: %v", err)
	}
}

func TestPluginBackend_ImportDocs_NoFetchDocs(t *testing.T) {
	p := &Plugin{
		Name:       "minimal",
		Extensions: []string{".min"},
		Tier:       1,
		Commands:   map[string]CommandConfig{},
	}
	be := NewBackend(p)

	docs, err := be.ImportDocs(nil, []string{"pkg1", "pkg2"})
	if err != nil {
		t.Errorf("ImportDocs() error = %v", err)
	}
	if docs != nil {
		t.Errorf("ImportDocs() should return nil when fetchDocs not configured, got: %v", docs)
	}
}

func TestPluginValidate_SetsLanguageID(t *testing.T) {
	p := Plugin{
		Name:       "rust",
		Extensions: []string{".rs"},
		Tier:       1,
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.LanguageID != "rust" {
		t.Errorf("LanguageID should default to name, got %q", p.LanguageID)
	}
}
