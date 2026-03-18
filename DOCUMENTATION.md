# Neko Documentation

Neko is an intelligent, language-aware development Model Context Protocol (MCP) server. It provides a standardized set of tools for code exploration, editing, building, and analysis across multiple programming languages.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Core Capabilities](#core-capabilities)
4. [Language Support](#language-support)
    - [Go](#go)
    - [Python](#python)
    - [JavaScript/TypeScript](#javascript-typescript)
5. [Extending Neko (Plugin System)](#extending-neko-plugin-system)
6. [Architecture](#architecture)

---

## Prerequisites

To use Neko effectively, ensure the following tools are installed on your system:

### Mandatory
- **Go 1.22+**: Required to build and run the server.
- **uv**: Required for all Python-related operations (initialization, dependencies, linting, testing).

### Recommended (for specific language support)
- **gopls**: For Go language intelligence (LSP).
- **Node.js & npm**: For JavaScript/TypeScript support (via the default plugin).
- **ruff, mypy, pytest**: These are automatically managed by `uv` within Python projects but are essential for the quality gate.

---

## Installation

### 1. Clone the repository
```bash
git clone https://github.com/danicat/neko.git
cd neko
```

### 2. Build the binary
```bash
make build
```
This will create the `neko` executable in the `bin/` directory.

### 3. Configure your MCP client
Add Neko to your MCP configuration (e.g., in Claude Desktop or Gemini CLI):

```json
{
  "mcpServers": {
    "neko": {
      "command": "/path/to/neko/bin/neko",
      "args": ["--plugin-dir", "/path/to/neko/plugins"]
    }
  }
}
```

---

## Core Capabilities

Neko provides a suite of tools designed for a high-quality development lifecycle:

- **`list_files`**: Context-aware file listing (respects `.gitignore`).
- **`smart_read`**: Reads files with optional AST-based outlining to save tokens.
- **`smart_edit`**: Precision editing with fuzzy matching, syntax validation, and auto-formatting.
- **`smart_build`**: A universal "Quality Gate" that runs the language-appropriate Build -> Test -> Lint pipeline.
- **`project_init`**: Bootstraps new projects with idiomatic structures.
- **`add_dependency`**: Installs packages and immediately returns their documentation.
- **LSP Tools**: `find_definition`, `find_references`, and `symbol_info` for deep code intelligence.

---

## Language Support

### Go
- **Backend**: Native (internal).
- **Tools**: Uses `go mod`, `go vet`, and `go test`.
- **LSP**: Invokes `gopls`.
- **Special Features**: `test_query` allows SQL querying of test results and coverage.

### Python
- **Backend**: Native (internal).
- **Requirement**: **Standardized on `uv`**. Neko does not support global/system packages or manual `venv` management.
- **Tools**: Every operation (`ruff`, `mypy`, `pytest`) is executed via `uv run` to ensure project-local consistency.
- **Initialization**: Uses `uv init` to create a modern Python project structure.

### JavaScript / TypeScript
- **Backend**: Plugin-based (via `plugins/javascript.json`).
- **Tools**: Uses `npm` and `node`.
- **Initialization**: Uses `npm init -y`.
- **Intelligence**: Uses `typescript-language-server`.

---

## Extending Neko (Plugin System)

Neko can be extended to support new languages without modifying the Go source code by adding a JSON configuration file to the `plugins/` directory.

### Plugin Schema
```json
{
  "name": "language-name",
  "extensions": [".ext1", ".ext2"],
  "projectMarkers": ["project.file"],
  "skipDirs": ["dir_to_ignore"],
  "tier": 2,
  "lsp": {
    "command": "server-binary",
    "args": ["--stdio"]
  },
  "commands": {
    "init": { "command": "cmd", "args": ["init", "{{dir}}"] },
    "build": { "command": "cmd", "args": ["run", "build"] },
    "test": { "command": "cmd", "args": ["test"] },
    "format": { "command": "formatter", "args": ["{{filename}}"] },
    "addDependency": { "command": "cmd", "args": ["install", "{{packages}}"] }
  }
}
```

### Key Placeholders
- `{{dir}}`: The project root directory.
- `{{filename}}`: The absolute path to the file being processed.
- `{{packages}}`: Space-separated list of packages to install.

---

## Architecture

Neko is built on a modular "Backend" architecture:

1. **Registry**: Discovers backends based on file extensions or project markers (`go.mod`, `pyproject.toml`, `package.json`).
2. **LanguageBackends**: 
    - **Internal**: Compiled-in backends for Go and Python for maximum performance and complex logic.
    - **Plugin**: Dynamic backends loaded from JSON, mapping standard development tasks to shell commands.
3. **LSP Manager**: Manages a pool of language servers, providing one instance per language per workspace root.
4. **Tools**: The MCP-exposed interface that delegates work to the Registry.

---

## Quality Gate

The `smart_build` tool is the heart of Neko's reliability. It doesn't just run tests; it ensures:
1. The code compiles.
2. Coding standards are met (Linting).
3. Types are correct (where applicable).
4. Tests pass with meaningful output.

Validation is the only path to finality in Neko. Never assume a change is correct until the Quality Gate has passed.
