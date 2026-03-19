# Design: Project Lifecycle & Dynamic Tool Surface

## Problem Statement

Today neko registers all 14 tools at startup, regardless of whether the user has a project open or what languages that project uses. This creates three problems:

1. **Tool overload** — The AI agent sees 14 tools and must guess which apply. For a pure Go project, Python-specific behaviors are noise.
2. **No project context** — Tools operate on "whatever path the user mentions," with no explicit project boundary.
3. **Multi-language ambiguity** — `ForDir()` returns a single backend based on tier priority. But language is a property of the *file*, not the directory. A polyglot repo needs multiple backends active simultaneously.

## Core Principle: Language is a Property of the File

Language is **never** a property of a directory. A directory can contain `.go`, `.ts`, and `.py` files simultaneously. The backend for any operation is determined by the file being operated on, resolved via `Registry.ForFile(path)`. The existing `Registry.ForDir()` should be deprecated in favor of file-based routing and explicit `language` parameters.

## Proposal

Introduce a **project lifecycle** with two phases and dynamic tool registration:

```
[LOBBY]  ──create_project──►  [PROJECT OPEN]  ──close_project──►  [LOBBY]
         ──open_project────►
```

### Phase 1: Lobby (no project open)

Only these tools are available:

| Tool | Purpose |
|------|---------|
| `create_project` | Bootstrap a new project |
| `open_project` | Open an existing project directory |

Navigation tools are deliberately excluded from the Lobby to ensure the AI establishes a valid project context before interacting with the filesystem.


### Phase 2: Project Open

After opening/creating a project, the lobby tools are replaced with project-aware tools. Registration follows a hierarchy of support:

#### 1. Always Enabled (Backend-Agnostic)
These tools work on any file type and provide graceful degradation when no backend is available.

| Tool | Purpose |
|------|---------|
| `close_project` | Return to lobby and clean up sessions |
| `list_files` | Navigate project (respects .gitignore) |
| `read_file` | Read with optional outline/imports |
| `edit_file` | Smart edit with validation and formatting |
| `create_file` | Create file with idiomatic formatting |
| `review_code` | AI-powered idiomatic review |

#### 2. Feature-Dependent (Capability-Based)
These tools are only registered if at least one active backend supports the underlying feature.

| Tool | Required Capability | Example Backends |
|------|---------------------|------------------|
| `build` | `toolchain` | Go, Python, JS |
| `read_docs` | `documentation` | Go, Python, JS |
| `add_dependencies` | `dependencies` | Go, Python, JS |
| `modernize_code` | `modernize` | Go, Python |
| `test_mutations` | `mutation_test` | Go, Python |
| `describe` | `lsp` | Go, Python, JS |
| `find_definition` | `lsp` | Go, Python, JS |
| `find_references` | `lsp` | Go, Python, JS |
| `query_tests` | `test_query` | Go |

When a project is opened, Neko scans for markers and identifies active backends. It then registers only the tools supported by the **union of capabilities** of those backends. For example, in a pure Python project, `query_tests` will not be registered because no active backend advertises SQL Test Querying.

#### Capability Discovery

The `LanguageBackend` interface gains a `Capabilities()` method:

```go
type Capability string

const (
    CapToolchain      Capability = "toolchain"
    CapDocumentation  Capability = "documentation"
    CapDependencies   Capability = "dependencies"
    CapModernize      Capability = "modernize"
    CapMutationTest   Capability = "mutation_test"
    CapTestQuery      Capability = "test_query"
    CapLSP            Capability = "lsp"
)

// Added to LanguageBackend interface
Capabilities() []Capability
```

Native backends (Go, Python) return their full capability set. Plugin backends derive capabilities from their configuration — if a plugin defines the `build` command, it advertises `CapToolchain`; if it defines `lsp`, it advertises `CapLSP`; and so on. This is the mapping that connects the plugin's JSON command definitions to the tool registration system.


## Tool Naming & Arguments

Standardize on **verb_noun** names and universal argument labels. Drop "smart_" prefixes.

### Argument Standards
- `file`: Path to a specific source file (used by file tools and LSP tools).
- `dir`: Path to a directory (defaults to project root). Used by tools that operate on a build target or subdirectory.
- `language`: Explicit backend selector. Optional when only one backend is active; required when ambiguous. If resolution fails, the tool returns an error asking the agent to specify one.
- `packages`: List of packages for dependency tools.
- `dependencies`: List of initial dependencies for project creation.

### Full Toolset

| Tool | Key Arguments | LSP Mapping |
|------|---------------|-------------|
| `create_project` | `dir`, `language`, `dependencies` | - |
| `open_project` | `dir` | - |
| `close_project` | - | - |
| `list_files` | `dir`, `depth` | - |
| `read_file` | `file`, `outline`, `start_line`, `end_line` | - |
| `create_file` | `file`, `content` | - |
| `edit_file` | `file`, `old_content`, `new_content` | - |
| `build` | `dir`, `language` | - |
| `read_docs` | `import_path`, `symbol`, `language` | - |
| `add_dependencies` | `packages`, `language` | - |
| `modernize_code` | `dir`, `language`, `fix` | - |
| `test_mutations` | `dir`, `language` | - |
| `query_tests` | `query`, `language` | - |
| `describe` | `file`, `line`, `col` | `textDocument/hover` |
| `find_definition` | `file`, `line`, `col` | `textDocument/definition` |
| `find_references` | `file`, `line`, `col` | `textDocument/references` |
| `review_code` | `file` | - |

## Language Detection

Neko uses **automatic detection** to determine active languages.

1. **On Open**: Scans project root for markers (`go.mod`, `pyproject.toml`, `package.json`). Activates **ALL** matching backends.
2. **On File Touch**: When a tool operates on a new file extension, the matching backend is activated dynamically.
3. **LSP Eagerness**: LSP servers are started **immediately** upon backend activation (at open or on-touch) to ensure warm sessions and instant error reporting.

## Implementation Phases

### Phase 1: Naming & Argument Standardization
- Rename all tools in the registry.
- Standardize all schemas to use `file`, `dir`, `language`, `packages`.
- Update instructions and documentation.

### Phase 2: Project Lifecycle Logic
- Implement `open_project`, `create_project`, and `close_project` handlers.
- Implement exclusive state management (Lobby vs Project).
- Implement eager LSP initialization.

### Phase 3: File-Based Routing
- Deprecate `ForDir()` in favor of `ForFile(file)` and `Get(language)`.
- Wire `language` parameter to backend selection logic.

### Phase 4: Progressive Discovery
- Trigger tool list notifications on file-based backend activation.

## Backend-Agnostic Tools (Nil Backend Support)

To ensure Neko remains useful for all file types (markdown, text, shell scripts, etc.), the core file tools are designed to work even when `Registry.ForFile(file)` returns `nil`. This is known as **Graceful Degradation**.

### Registration Logic
- **Always Enabled**: Once a project is open, backend-agnostic tools are registered immediately.
- **Backend-Required**: Tools that require a language backend (e.g., `build`, `hover`) are only registered if at least one language backend is active for the project. If a project is opened but no languages are detected, these tools remain hidden.

| Tool | Behavior with Nil Backend |
|------|---------------------------|
| `list_files` | Operates normally. Uses standard filesystem traversal and respects project-level `.gitignore`. |
| `read_file` | Operates normally for content reading (`start_line`, `end_line`). If `outline=true` is requested, it returns the full content with a note: "Outline not available for this file type." |
| `create_file` | Writes the file to disk. Skips the "Auto-format" step. |
| `edit_file` | Performs the fuzzy match and patch operation. Skips the "Syntax Validation" and "Auto-format" steps. |
| `close_project` | Operates normally. Clears the project context and LSPs regardless of backends. |
| `review_code` | Performs a general text-based AI review without language-specific hints or AST context. |

### Implementation Note for Developers
Handlers for these tools must check if the backend is `nil` and skip language-specific steps (validation, formatting, outlining) instead of returning an error.

---

## Reviewer Notes

### Protocol-Level Enforcement
Neko utilizes the MCP `notifications/tools/list_changed` capability to enforce the project lifecycle. The agent simply cannot "see" or call engineering tools until the handshake (`open_project`) is complete.

### Eager LSP Initialization
The `open_project` handshake is the ideal moment to initialize LSP servers. Eagerly starting LSPs for all detected languages ensures a "warm start" and allows Neko to surface initialization errors immediately.

### Context Reset
`close_project` is a mandatory **Context Reset**. It shuts down all LSP sessions and clears the agent's tool surface, ensuring a clean slate for the next project.
