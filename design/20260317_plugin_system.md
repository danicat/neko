# Design: Generic Language Plugin System for Neko

## Objective

Consolidate `godoctor` and `pyhd` into `neko` and provide a generic, extensible MCP server for code-aware tools. The system should support native implementations for Go and Python while allowing easy addition of other languages via JSON configuration and LSP integration.

## Architecture

### 1. Language Plugin Definition

A plugin is defined by a JSON file that specifies how to handle various language-specific tasks.

**JSON Schema (`plugin.json`):**

```json
{
  "name": "language-name",
  "extensions": [".ext1", ".ext2"],
  "projectMarkers": ["marker.file", "config.json"],
  "skipDirs": ["dir1", "dir2"],
  "tier": 2,
  "lsp": {
    "command": "lsp-command",
    "args": ["--arg1"]
  },
  "commands": {
    "validate": { "command": "cmd", "args": ["{{filename}}"] },
    "format": { "command": "cmd", "args": ["{{filename}}"] },
    "outline": { "command": "cmd", "args": ["{{filename}}"] },
    "build": { "command": "cmd", "args": ["build", "{{dir}}"] },
    "test": { "command": "cmd", "args": ["test", "{{dir}}"] },
    "fetchDocs": { "command": "cmd", "args": ["docs", "{{package}}", "{{symbol}}"] },
    "addDependency": { "command": "cmd", "args": ["install", "{{packages}}"] },
    "init": { "command": "cmd", "args": ["init", "{{path}}"] },
    "modernize": { "command": "cmd", "args": ["upgrade", "{{dir}}"] },
    "mutationTest": { "command": "cmd", "args": ["mutate", "{{dir}}"] }
  }
}
```

### 2. `PluginBackend` Implementation

A new `PluginBackend` struct will implement the `LanguageBackend` interface by:
- Executing the specified external commands.
- Replacing placeholders like `{{filename}}`, `{{dir}}`, `{{package}}`, `{{symbol}}`, and `{{packages}}`.
- Providing fallback mechanisms (e.g., using LSP for format/validate if specified and external commands are missing).

### 3. Registry Enhancements

The `Registry` will be updated to:
- Load plugins from a standard location (e.g., `~/.neko/plugins/` or a relative `./plugins/` directory).
- Allow discovery of available plugins.
- Prioritize native implementations over generic plugins if both exist for the same language/extension.

### 4. Native Backend Preservation

- `golang` and `python` backends remain as hardcoded Go packages to preserve their specialized logic (e.g., Go's `godoc` integration and Python's AST parsing).
- They will be registered alongside `PluginBackend` instances.

### 5. LSP Integration

Common tasks like `Format` and `Validate` can be delegated to an LSP server if the plugin defines an `lsp` command but doesn't provide specific tool commands for those tasks.

## Implementation Plan

### Phase 1: Core Refactoring
1.  **Define `Plugin` Struct**: Create `internal/backend/plugin/plugin.go` with the JSON-serializable structure.
2.  **Implement `PluginBackend`**: Implement the `LanguageBackend` interface using the `Plugin` struct.
3.  **Command Execution Helper**: Create a utility to execute commands with placeholder replacement and context awareness.

### Phase 2: Registry & Configuration
1.  **Update `Registry`**: Add `LoadPlugins(dir string)` to the `Registry`.
2.  **Plugin Discovery**: Implement logic to scan a directory for `.json` plugin definitions.
3.  **Main Integration**: Update `cmd/neko/main.go` to load plugins on startup.

### Phase 3: Generalization & LSP
1.  **LSP Integration**: Add support for using LSP servers for basic tasks in `PluginBackend`.
2.  **Optional Interface Methods**: Ensure `LanguageBackend` methods return clear "not supported" errors when a plugin doesn't provide the necessary command.

### Phase 4: Validation
1.  **Create Sample Plugins**: Implement sample plugins for languages like JavaScript (using ESLint/Prettier/npm) or Rust (using cargo/rust-analyzer).
2.  **Verify Go/Python Compatibility**: Ensure the new system doesn't break existing Go and Python support.

## Success Criteria

- Neko can support a new language (e.g., Ruby) just by adding a JSON file.
- The Go and Python backends continue to function with their specialized tools.
- LSP servers can be automatically used for formatting and validation when configured.

## Implementation Summary (2026-03-17)

- [x] **Plugin System**: Implemented a JSON-based plugin system in `internal/backend/plugin`.
- [x] **Registry Tiers**: Updated `internal/backend/registry.go` to support tiered prioritization (Native backends at Tier 3, Plugins default to Tier 2).
- [x] **Configuration**: Added `-plugins` flag to specify plugin directories.
- [x] **Integration**: Integrated plugin loading into the MCP server startup and agent instruction generation.
- [x] **LSP Support**: Plugins can specify LSP servers which are automatically used by `symbol_info`, `find_definition`, and `find_references`.
- [x] **Graceful Fallbacks**: Tools like `smart_edit` handle missing `validate` or `format` commands gracefully.
