package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danicat/neko/internal/backend"
)

// CommandConfig defines an external command to execute.
type CommandConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// LSPConfig defines how to start an LSP server.
type LSPConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Plugin defines a language plugin from a JSON configuration.
type Plugin struct {
	Name           string                   `json:"name"`
	Extensions     []string                 `json:"extensions"`
	ProjectMarkers []string                 `json:"projectMarkers"`
	SkipDirs       []string                 `json:"skipDirs"`
	Tier           int                      `json:"tier"`
	LSP            *LSPConfig               `json:"lsp,omitempty"`
	Commands       map[string]CommandConfig `json:"commands"`
}

// PluginBackend implements backend.LanguageBackend using a Plugin configuration.
type PluginBackend struct {
	plugin *Plugin
}

// NewBackend creates a new Backend from a Plugin.
func NewBackend(p *Plugin) *PluginBackend {
	return &PluginBackend{plugin: p}
}

// Name implements backend.LanguageBackend.
func (b *PluginBackend) Name() string {
	return b.plugin.Name
}

// FileExtensions implements backend.LanguageBackend.
func (b *PluginBackend) FileExtensions() []string {
	return b.plugin.Extensions
}

// ProjectMarkers implements backend.LanguageBackend.
func (b *PluginBackend) ProjectMarkers() []string {
	return b.plugin.ProjectMarkers
}

// SkipDirs implements backend.LanguageBackend.
func (b *PluginBackend) SkipDirs() []string {
	return b.plugin.SkipDirs
}

// Tier implements backend.LanguageBackend.
func (b *PluginBackend) Tier() int {
	return b.plugin.Tier
}

// LSPCommand implements backend.LanguageBackend.
func (b *PluginBackend) LSPCommand() (string, []string, bool) {
	if b.plugin.LSP == nil {
		return "", nil, false
	}
	return b.plugin.LSP.Command, b.plugin.LSP.Args, true
}

func (b *PluginBackend) InitializationOptions() map[string]interface{} {
	return nil // Could be added to Plugin struct later if needed
}

func (b *PluginBackend) runCommand(ctx context.Context, cmdName string, vars map[string]string) (string, error) {
	cmdCfg, ok := b.plugin.Commands[cmdName]
	if !ok {
		return "", fmt.Errorf("command %s not configured for plugin %s", cmdName, b.plugin.Name)
	}

	args := make([]string, len(cmdCfg.Args))
	for i, arg := range cmdCfg.Args {
		newArg := arg
		for k, v := range vars {
			newArg = strings.ReplaceAll(newArg, "{{"+k+"}}", v)
		}
		args[i] = newArg
	}

	cmd := exec.CommandContext(ctx, cmdCfg.Command, args...)
	if dir, ok := vars["dir"]; ok && dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), fmt.Errorf("command %s failed: %w", cmdName, err)
	}
	return string(out), nil
}

// Validate implements backend.LanguageBackend.
func (b *PluginBackend) Validate(ctx context.Context, filename string) error {
	if _, ok := b.plugin.Commands["validate"]; !ok {
		return nil // Optional
	}
	_, err := b.runCommand(ctx, "validate", map[string]string{"filename": filename})
	return err
}

// Format implements backend.LanguageBackend.
func (b *PluginBackend) Format(ctx context.Context, filename string) error {
	if _, ok := b.plugin.Commands["format"]; !ok {
		return nil // Optional
	}
	_, err := b.runCommand(ctx, "format", map[string]string{"filename": filename})
	return err
}

// Outline implements backend.LanguageBackend.
func (b *PluginBackend) Outline(ctx context.Context, filename string) (string, error) {
	return b.runCommand(ctx, "outline", map[string]string{"filename": filename})
}

// ParseImports implements backend.LanguageBackend.
func (b *PluginBackend) ParseImports(ctx context.Context, filename string) ([]string, error) {
	out, err := b.runCommand(ctx, "parseImports", map[string]string{"filename": filename})
	if err != nil {
		return nil, err
	}
	var imports []string
	if err := json.Unmarshal([]byte(out), &imports); err != nil {
		// Try line by line if not JSON
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				imports = append(imports, trimmed)
			}
		}
	}
	return imports, nil
}

// ImportDocs implements backend.LanguageBackend.
func (b *PluginBackend) ImportDocs(ctx context.Context, imports []string) ([]string, error) {
	// Not easily generalized, returning empty docs for now
	return nil, nil
}

// BuildPipeline implements backend.LanguageBackend.
func (b *PluginBackend) BuildPipeline(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	out, err := b.runCommand(ctx, "build", map[string]string{"dir": dir})
	report := &backend.BuildReport{Output: out}
	if err != nil {
		report.IsError = true
	}
	return report, nil
}

// FetchDocs implements backend.LanguageBackend.
func (b *PluginBackend) FetchDocs(ctx context.Context, dir string, pkg string, symbol string) (string, error) {
	return b.runCommand(ctx, "fetchDocs", map[string]string{"package": pkg, "symbol": symbol, "dir": dir})
}

// AddDependency implements backend.LanguageBackend.
func (b *PluginBackend) AddDependency(ctx context.Context, dir string, packages []string) (string, error) {
	return b.runCommand(ctx, "addDependency", map[string]string{"dir": dir, "packages": strings.Join(packages, " ")})
}

// InitProject implements backend.LanguageBackend.
func (b *PluginBackend) InitProject(ctx context.Context, opts backend.InitOpts) error {
	if opts.Path != "" {
		if err := os.MkdirAll(opts.Path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	_, err := b.runCommand(ctx, "init", map[string]string{"path": opts.Path, "modulePath": opts.ModulePath, "dir": opts.Path})
	return err
}

// Modernize implements backend.LanguageBackend.
func (b *PluginBackend) Modernize(ctx context.Context, dir string, fix bool) (string, error) {
	fixStr := "false"
	if fix {
		fixStr = "true"
	}
	return b.runCommand(ctx, "modernize", map[string]string{"dir": dir, "fix": fixStr})
}

// MutationTest implements backend.LanguageBackend.
func (b *PluginBackend) MutationTest(ctx context.Context, dir string) (string, error) {
	return b.runCommand(ctx, "mutationTest", map[string]string{"dir": dir})
}

// BuildTestDB implements backend.LanguageBackend.
func (b *PluginBackend) BuildTestDB(ctx context.Context, dir string, pkg string) error {
	_, err := b.runCommand(ctx, "buildTestDB", map[string]string{"dir": dir, "package": pkg})
	return err
}

// QueryTestDB implements backend.LanguageBackend.
func (b *PluginBackend) QueryTestDB(ctx context.Context, dir string, query string) (string, error) {
	return b.runCommand(ctx, "queryTestDB", map[string]string{"dir": dir, "query": query})
}

// LoadPlugins loads all plugins from a directory.
func LoadPlugins(pluginDir string) ([]*PluginBackend, error) {
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, err
	}

	var backends []*PluginBackend
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			p, err := loadPlugin(filepath.Join(pluginDir, f.Name()))
			if err != nil {
				continue // log error?
			}
			backends = append(backends, NewBackend(p))
		}
	}
	return backends, nil
}

func loadPlugin(path string) (*Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plugin
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
