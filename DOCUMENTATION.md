# Neko Documentation

Neko is an intelligent, language-aware development Model Context Protocol (MCP) server. It provides a standardized set of tools for code exploration, editing, building, and analysis across multiple programming languages.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Project Lifecycle](#project-lifecycle)
4. [Tool Reference](#tool-reference)
5. [Language Support](#language-support)
6. [Extending Neko (Plugin System)](#extending-neko-plugin-system)
7. [Architecture](#architecture)

---

## Prerequisites

### Mandatory
- **Go 1.22+**: Required to build and run the server.
- **uv**: Required for all Python-related operations (initialization, dependencies, linting, testing).

### Recommended (for specific language support)
- **gopls**: For Go language intelligence (LSP).
- **Node.js & npm**: For JavaScript/TypeScript support (via the default plugin).
- **ruff, mypy, pytest**: Automatically managed by `uv` within Python projects but essential for the quality gate.

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
This creates the `neko` executable in the project root.

### 3. Configure your MCP client
Add Neko to your MCP configuration (e.g., in Claude Desktop or Gemini CLI):

```json
{
  "mcpServers": {
    "neko": {
      "command": "/path/to/neko/neko",
      "args": ["-plugins", "/path/to/neko/plugins"]
    }
  }
}
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `--version` | Print version and exit |
| `--listen ADDR` | Start HTTP server (e.g., `127.0.0.1:8080`) |
| `--model MODEL` | Default Gemini model for code review |
| `--plugins DIR` | Plugin directory path |

---

## Project Lifecycle

Neko enforces a two-phase lifecycle using MCP's `notifications/tools/list_changed` capability. The AI agent cannot see or call engineering tools until it has established a project context.

```
[LOBBY]  ──open_project──►  [PROJECT OPEN]  ──close_project──►  [LOBBY]
         ──create_project──►
```

### Lobby Phase

Only two tools are available:

| Tool | Purpose |
|------|---------|
| `open_project` | Open an existing project directory, detect languages, start LSP servers |
| `create_project` | Bootstrap a new project with idiomatic structure and dependencies |

### Project Open Phase

After opening a project, Neko scans for language markers (`go.mod`, `pyproject.toml`, `package.json`) and activates the corresponding backends. Tools are registered based on the **union of capabilities** of all active backends.

#### Always Enabled (Backend-Agnostic)

These tools work on any file type and provide graceful degradation when no backend is available.

| Tool | Purpose |
|------|---------|
| `close_project` | Return to lobby, shut down LSP servers |
| `list_files` | Navigate project (respects .gitignore) |
| `read_file` | Read with optional outline/imports |
| `edit_file` | Smart edit with validation and formatting |
| `create_file` | Create file with idiomatic formatting |
| `review_code` | AI-powered idiomatic review |

#### Capability-Dependent

These tools are only registered if at least one active backend supports the underlying feature.

| Tool | Required Capability | Description |
|------|---------------------|-------------|
| `build` | `toolchain` | Universal quality gate: build, test, lint, modernize |
| `read_docs` | `documentation` | Fetch API docs for any package or symbol |
| `add_dependencies` | `dependencies` | Install packages and return their docs |
| `test_mutations` | `mutation_test` | Mutation testing to measure test quality |
| `query_tests` | `test_query` | SQL queries over test results and coverage |
| `describe` | `lsp` | Type info and docs for a symbol at a position |
| `find_definition` | `lsp` | Jump to a symbol's definition |
| `find_references` | `lsp` | Find all references to a symbol |

### Dynamic Backend Activation

When a tool operates on a file whose extension matches a non-active backend, that backend is activated dynamically ("on-touch"). This ensures polyglot repos work seamlessly without requiring all languages to be detected up front.

---

## Tool Reference

### Argument Standards

All tools use consistent argument names:

| Argument | Description |
|----------|-------------|
| `file` | Path to a specific source file |
| `dir` | Path to a directory (defaults to project root) |
| `language` | Explicit backend selector. Optional when only one backend is active; required when ambiguous |
| `packages` | List of packages for dependency tools |
| `dependencies` | List of initial dependencies for project creation |

### Navigation

- **`list_files(dir, depth)`**: Recursively list source files while filtering build artifacts. Respects `.gitignore`.
- **`read_file(file, start_line, end_line)`**: Enhanced file reader.
  - **Semantic Annotations**: Automatically injects type-signature metadata using `<NEKO>` tags.
  - **Snippet mode** (`start_line`, `end_line`): Targeted line range reading.
  - **Full read**: Returns content with line numbers and imported package documentation.
- **`semantic_search(query, limit)`**: Intent-based discovery using a local vector database. Returns ranked code chunks by meaning.

### Editing

- **`edit_file(file, old_content, new_content)`**: Intelligent editor with safety guarantees.
  - **Fuzzy matching**: Locates target blocks despite minor whitespace differences. Returns ranked suggestions on failure.
  - **LSP Lifecycle**: Synchronizes with LSP and returns full project diagnostics.
  - **Automated Actions**: Automatically organizes imports and formats code on save.
- **`multi_edit(edits)`**: Transactional batch editing across multiple files in a single turn.
- **`create_file(file, content)`**: Creates a file with automatic parent directory creation, LSP synchronization, and diagnostics.

### Toolchain

- **`build(dir, language, auto_fix, run_modernize)`**: The quality gate. Runs the language-appropriate Build -> Modernize -> Test -> Lint pipeline with automatic LSP synchronization.
- **`read_docs(import_path, symbol, language)`**: Fetch documentation for any package or symbol.
- **`add_dependencies(packages, language)`**: Install packages and return their API documentation.
- **`create_project(dir, language, dependencies)`**: Bootstrap a new project with idiomatic structure.

### Code Intelligence (LSP)

- **`describe(file, line, col)`**: Returns type information and documentation for a symbol. Maps to `textDocument/hover`.
- **`find_definition(file, line, col)`**: Jumps to a symbol's definition. Maps to `textDocument/definition`.
- **`find_references(file, line, col)`**: Finds all references across the codebase. Categorizes results into [SOURCE] and [TESTS] with symbol context.
- **`rename_symbol(file, line, col, new_name)`**: Performs a deterministic, project-wide rename via LSP.

### Testing

- **`test_mutations(dir, language)`**: Mutation testing to objectively measure test suite quality.
- **`query_tests(query, language)`**: SQL queries over test results and coverage data.
- **`review_code(file)`**: AI-powered architectural and idiomatic review.

---

## Language Support

### Go
- **Backend**: Native (internal).
- **Capabilities**: All (toolchain, documentation, dependencies, modernize, mutation_test, test_query, lsp).
- **LSP**: `gopls`.
- **Special Features**: `query_tests` enables SQL querying of test results and coverage data.

### Python
- **Backend**: Native (internal).
- **Capabilities**: toolchain, documentation, dependencies, modernize, mutation_test, lsp.
- **Requirement**: Standardized on `uv`. Every operation (`ruff`, `mypy`, `pytest`) runs via `uv run`. `uv sync` is used for environment management.
- **LSP**: `uv run pylsp`.
- **Initialization**: Uses `uv init` for modern Python project structure.

### JavaScript / TypeScript
- **Backend**: Plugin-based (via `plugins/javascript.json`).
- **Tools**: Uses `npm` and `node`.
- **LSP**: `typescript-language-server`.
- **Modernize**: Uses `eslint --fix` via the plugin.
- **Initialization**: Uses `npm init -y`.

---

## Extending Neko (Plugin System)

Add support for new languages by placing a JSON configuration file in the `plugins/` directory. No Go code changes required.

### Plugin Schema
```json
{
  "name": "rust",
  "languageId": "rust",
  "extensions": [".rs"],
  "projectMarkers": ["Cargo.toml"],
  "skipDirs": ["target"],
  "tier": 2,
  "lsp": {
    "command": "rust-analyzer",
    "args": [],
    "initializationOptions": {}
  },
  "commands": {
    "build": { "command": "cargo", "args": ["build"] },
    "test": { "command": "cargo", "args": ["test"] },
    "validate": { "command": "cargo", "args": ["check", "{{filename}}"] },
    "format": { "command": "rustfmt", "args": ["{{filename}}"] },
    "fetchDocs": { "command": "rust-doc-fetch", "args": ["{{package}}", "{{symbol}}"] },
    "addDependency": { "command": "cargo", "args": ["add", "{{packages...}}"] }
  }
}
```

### Placeholders
| Placeholder | Description |
|-------------|-------------|
| `{{dir}}` | The project root directory |
| `{{filename}}` | Absolute path to the file being processed |
| `{{package}}` | Package name for documentation lookup |
| `{{symbol}}` | Symbol name for documentation lookup |
| `{{packages}}` | Space-separated list of packages |
| `{{packages...}}` | Variadic: each package becomes a separate argument |

### Capability Mapping

Plugin capabilities are derived from the commands defined in the JSON:

| Command | Capability |
|---------|------------|
| `build` or `buildSteps` | `toolchain` |
| `fetchDocs` | `documentation` |
| `addDependency` | `dependencies` |
| `modernize` | `modernize` |
| `mutationTest` | `mutation_test` |
| `queryTestDB` | `test_query` |
| `lsp` section defined | `lsp` |

### Tier System

Backends are prioritized by tier when multiple match a file extension:
- **Tier 3**: Native backends (Go, Python)
- **Tier 2**: Standard plugins
- **Tier 0-1**: Low-priority or experimental plugins

---

## Architecture

Neko is built on a modular backend architecture with a two-phase lifecycle:

```
┌─────────────────────────────────────────────────┐
│                   MCP Server                     │
│  ┌───────────┐  ┌────────────┐  ┌────────────┐  │
│  │   Lobby   │  │  Project   │  │ Capability  │  │
│  │  Phase    │──│  State     │──│  Scanner    │  │
│  └───────────┘  └────────────┘  └────────────┘  │
├─────────────────────────────────────────────────┤
│              ResolveBackend(language)             │
│  ┌──────────────────────────────────────────┐    │
│  │            Backend Registry               │    │
│  │  ┌────────┐  ┌────────┐  ┌────────────┐  │    │
│  │  │   Go   │  │ Python │  │  Plugins   │  │    │
│  │  │ (native)│  │(native)│  │  (JSON)    │  │    │
│  │  └────────┘  └────────┘  └────────────┘  │    │
│  └──────────────────────────────────────────┘    │
├─────────────────────────────────────────────────┤
│               LSP Manager                        │
│  One client per language per workspace root      │
└─────────────────────────────────────────────────┘
```

1. **MCP Server**: Manages the two-phase lifecycle (Lobby vs Project Open) and dynamic tool registration.
2. **ResolveBackend**: Routes tool calls to the correct backend. Uses active project backends when a single language is active; requires an explicit `language` parameter when ambiguous.
3. **Backend Registry**: Discovers and manages backends via `ForFile()` (file extension), `DetectBackends()` (project markers), and `Get()` (by name).
4. **LSP Manager**: Manages a pool of language servers, one instance per language per workspace root. Eagerly initialized when a project is opened.

### Quality Gate

The `build` tool is the heart of Neko's reliability. It enforces:
1. The code compiles.
2. Coding standards are met (linting).
3. Types are correct (where applicable).
4. Tests pass with meaningful output.

Validation is the only path to finality. Never assume a change is correct until the Quality Gate has passed.
