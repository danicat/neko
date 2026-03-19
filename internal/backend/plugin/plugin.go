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
	Command               string                 `json:"command"`
	Args                  []string               `json:"args"`
	InitializationOptions map[string]interface{} `json:"initializationOptions,omitempty"`
}

// Plugin defines a language plugin from a JSON configuration.
type Plugin struct {
	Name           string                   `json:"name"`
	LanguageID     string                   `json:"languageId"`
	Extensions     []string                 `json:"extensions"`
	ProjectMarkers []string                 `json:"projectMarkers"`
	SkipDirs       []string                 `json:"skipDirs"`
	Tier           int                      `json:"tier"`
	LSP            *LSPConfig               `json:"lsp,omitempty"`
	Commands       map[string]CommandConfig `json:"commands"`
	BuildSteps     []CommandConfig          `json:"buildSteps,omitempty"`
}

// Validate checks that the plugin configuration is well-formed.
func (p *Plugin) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if len(p.Extensions) == 0 {
		return fmt.Errorf("plugin %q must define at least one file extension", p.Name)
	}
	if p.Tier < 0 || p.Tier > 2 {
		return fmt.Errorf("plugin %q tier must be 0-2 (got %d); tiers >= 3 are reserved for native backends", p.Name, p.Tier)
	}
	if p.LanguageID == "" {
		// Derive from name as fallback
		p.LanguageID = p.Name
	}
	return nil
}

// PluginBackend implements backend.LanguageBackend using a Plugin configuration.
type PluginBackend struct {
	plugin *Plugin
}

// NewBackend creates a new Backend from a Plugin.
func NewBackend(p *Plugin) *PluginBackend {
	return &PluginBackend{plugin: p}
}

func (b *PluginBackend) Name() string             { return b.plugin.Name }
func (b *PluginBackend) FileExtensions() []string { return b.plugin.Extensions }
func (b *PluginBackend) ProjectMarkers() []string { return b.plugin.ProjectMarkers }
func (b *PluginBackend) SkipDirs() []string       { return b.plugin.SkipDirs }
func (b *PluginBackend) Tier() int                { return b.plugin.Tier }

func (b *PluginBackend) Capabilities() []backend.Capability {
	var caps []backend.Capability
	if b.plugin.LSP != nil {
		caps = append(caps, backend.CapLSP)
	}
	if _, ok := b.plugin.Commands["build"]; ok || len(b.plugin.BuildSteps) > 0 {
		caps = append(caps, backend.CapToolchain)
	}
	if _, ok := b.plugin.Commands["fetchDocs"]; ok {
		caps = append(caps, backend.CapDocumentation)
	}
	if _, ok := b.plugin.Commands["addDependency"]; ok {
		caps = append(caps, backend.CapDependencies)
	}
	if _, ok := b.plugin.Commands["modernize"]; ok {
		caps = append(caps, backend.CapModernize)
	}
	if _, ok := b.plugin.Commands["mutationTest"]; ok {
		caps = append(caps, backend.CapMutationTest)
	}
	if _, ok := b.plugin.Commands["queryTestDB"]; ok {
		caps = append(caps, backend.CapTestQuery)
	}
	return caps
}

// LanguageID returns the LSP language identifier for this plugin.

func (b *PluginBackend) LanguageID() string { return b.plugin.LanguageID }

func (b *PluginBackend) LSPCommand() (string, []string, bool) {
	if b.plugin.LSP == nil {
		return "", nil, false
	}
	return b.plugin.LSP.Command, b.plugin.LSP.Args, true
}

func (b *PluginBackend) InitializationOptions() map[string]interface{} {
	if b.plugin.LSP != nil {
		return b.plugin.LSP.InitializationOptions
	}
	return nil
}

func (b *PluginBackend) runCommand(ctx context.Context, cmdName string, vars map[string]string) (string, error) {
	cmdCfg, ok := b.plugin.Commands[cmdName]
	if !ok {
		return "", fmt.Errorf("command %s not configured for plugin %s", cmdName, b.plugin.Name)
	}

	args := expandArgs(cmdCfg.Args, vars)

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

// expandArgs substitutes placeholders in command arguments.
// {{key...}} is a variadic placeholder: its value is split on spaces
// and each token becomes a separate argument.
// All other {{key}} placeholders are replaced inline.
func expandArgs(templateArgs []string, vars map[string]string) []string {
	var result []string
	for _, arg := range templateArgs {
		// Check for variadic placeholder {{key...}}
		expanded := false
		for k, v := range vars {
			if !strings.HasSuffix(k, "...") {
				continue
			}
			placeholder := "{{" + k + "}}"
			if arg == placeholder {
				for _, part := range strings.Fields(v) {
					result = append(result, part)
				}
				expanded = true
				break
			}
		}
		if expanded {
			continue
		}

		// Standard placeholder substitution
		newArg := arg
		for k, v := range vars {
			if strings.HasSuffix(k, "...") {
				continue // variadic keys only match as standalone args
			}
			newArg = strings.ReplaceAll(newArg, "{{"+k+"}}", v)
		}
		result = append(result, newArg)
	}
	return result
}

func (b *PluginBackend) Validate(ctx context.Context, filename string) error {
	if _, ok := b.plugin.Commands["validate"]; !ok {
		return nil // Optional: no validator configured
	}
	_, err := b.runCommand(ctx, "validate", map[string]string{"filename": filename})
	return err
}

func (b *PluginBackend) Format(ctx context.Context, filename string) error {
	if _, ok := b.plugin.Commands["format"]; !ok {
		return nil // Optional: no formatter configured
	}
	_, err := b.runCommand(ctx, "format", map[string]string{"filename": filename})
	return err
}

func (b *PluginBackend) Outline(ctx context.Context, filename string) (string, error) {
	return b.runCommand(ctx, "outline", map[string]string{"filename": filename})
}

func (b *PluginBackend) ParseImports(ctx context.Context, filename string) ([]string, error) {
	out, err := b.runCommand(ctx, "parseImports", map[string]string{"filename": filename})
	if err != nil {
		return nil, err
	}
	var imports []string
	if err := json.Unmarshal([]byte(out), &imports); err != nil {
		// Fallback: line-delimited output
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				imports = append(imports, trimmed)
			}
		}
	}
	return imports, nil
}

func (b *PluginBackend) ImportDocs(ctx context.Context, imports []string) ([]string, error) {
	if _, ok := b.plugin.Commands["fetchDocs"]; !ok {
		return nil, nil
	}
	var docs []string
	for i, imp := range imports {
		if i >= 10 {
			docs = append(docs, "... (more imports)")
			break
		}
		out, err := b.FetchDocs(ctx, ".", imp, "")
		if err != nil {
			continue
		}
		summary := strings.TrimSpace(out)
		if len(summary) > 200 {
			summary = summary[:197] + "..."
		}
		if summary != "" {
			docs = append(docs, fmt.Sprintf("**%s**: %s", imp, summary))
		}
	}
	return docs, nil
}

func (b *PluginBackend) BuildPipeline(ctx context.Context, dir string, opts backend.BuildOpts) (*backend.BuildReport, error) {
	vars := map[string]string{"dir": dir, "packages": opts.Packages}

	// Multi-step pipeline if buildSteps is configured
	if len(b.plugin.BuildSteps) > 0 {
		var combined strings.Builder
		for _, step := range b.plugin.BuildSteps {
			args := expandArgs(step.Args, vars)
			cmd := exec.CommandContext(ctx, step.Command, args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			combined.WriteString(string(out))
			if err != nil {
				return &backend.BuildReport{Output: combined.String(), IsError: true}, nil
			}
		}
		return &backend.BuildReport{Output: combined.String()}, nil
	}

	// Single-command fallback
	out, err := b.runCommand(ctx, "build", vars)
	report := &backend.BuildReport{Output: out}
	if err != nil {
		report.IsError = true
	}
	return report, nil
}

func (b *PluginBackend) FetchDocs(ctx context.Context, dir string, pkg string, symbol string) (string, error) {
	return b.runCommand(ctx, "fetchDocs", map[string]string{"package": pkg, "symbol": symbol, "dir": dir})
}

func (b *PluginBackend) AddDependency(ctx context.Context, dir string, packages []string) (string, error) {
	return b.runCommand(ctx, "addDependency", map[string]string{"dir": dir, "packages...": strings.Join(packages, " ")})
}

func (b *PluginBackend) InitProject(ctx context.Context, opts backend.InitOpts) error {
	if opts.Path != "" {
		//nolint:gosec // G301
		if err := os.MkdirAll(opts.Path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	_, err := b.runCommand(ctx, "init", map[string]string{"path": opts.Path, "modulePath": opts.ModulePath, "dir": opts.Path})
	return err
}

func (b *PluginBackend) Modernize(ctx context.Context, dir string, fix bool) (string, error) {
	fixStr := "false"
	if fix {
		fixStr = "true"
	}
	return b.runCommand(ctx, "modernize", map[string]string{"dir": dir, "fix": fixStr})
}

func (b *PluginBackend) MutationTest(ctx context.Context, dir string) (string, error) {
	return b.runCommand(ctx, "mutationTest", map[string]string{"dir": dir})
}

func (b *PluginBackend) BuildTestDB(ctx context.Context, dir string, pkg string) error {
	_, err := b.runCommand(ctx, "buildTestDB", map[string]string{"dir": dir, "package": pkg})
	return err
}

func (b *PluginBackend) QueryTestDB(ctx context.Context, dir string, query string) (string, error) {
	return b.runCommand(ctx, "queryTestDB", map[string]string{"dir": dir, "query": query})
}

// LoadPlugins loads all plugins from a directory. Returns an error if the
// directory exists but any plugin file is malformed or invalid.
func LoadPlugins(pluginDir string) ([]*PluginBackend, error) {
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory %s: %w", pluginDir, err)
	}

	var backends []*PluginBackend
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		path := filepath.Join(pluginDir, f.Name())
		p, err := loadPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", path, err)
		}
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("invalid plugin %s: %w", path, err)
		}
		backends = append(backends, NewBackend(p))
	}
	return backends, nil
}

func loadPlugin(path string) (*Plugin, error) {
	//nolint:gosec // G304
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plugin
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &p, nil
}
