# PAWS — Protocol for Agent Workspace Services

**Version**: 0.1.0 (Draft)
**Status**: Proposal
**Authors**: Neko Project

---

## Abstract

PAWS (Protocol for Agent Workspace Services) is a protocol for communication between AI coding agents and language-aware development servers. It extends the general-purpose Model Context Protocol (MCP) with primitives purpose-built for software engineering: project lifecycle management, semantic code intelligence, quality gates, and intent-based code discovery.

Where MCP provides a universal "tool call" abstraction, PAWS defines **what those tools should be** for coding — establishing a shared contract that any coding agent can rely on and any language server can implement.

## Motivation

### Why not just MCP?

MCP is a transport and capability-negotiation layer. It says "here is a tool with this schema, call it with JSON." It does not say:

- What tools a coding server should expose
- When those tools become available
- What guarantees an edit provides (is it validated? formatted?)
- How the server manages project state across a session
- How the agent discovers code semantically vs. lexically

Every MCP coding server today reinvents these decisions independently. PAWS standardizes them, so that:

1. **Agents can assume capabilities.** A PAWS server always has a Quality Gate. An agent doesn't need to discover "is there a build tool?" — it knows the contract.
2. **Servers can be swapped.** A PAWS server for Rust and a PAWS server for Go expose the same protocol surface. The agent's workflow doesn't change.
3. **The protocol encodes hard-won lessons.** Fuzzy matching, immediate error feedback, type info injection, session-level deduplication — these aren't nice-to-haves, they're essential for effective AI coding. PAWS makes them first-class.

### Design Principles

1. **Semantic, not textual.** Every file operation is backed by language intelligence. Edits are validated, reads are annotated, renames are AST-driven.
2. **Fail fast, fail loud.** Every mutation returns immediate diagnostic feedback. The agent never has to run a separate build to discover it broke something.
3. **Progressive disclosure.** The server exposes only what the project supports. No phantom tools, no "not implemented" errors.
4. **Session-aware.** The server tracks what the agent has seen and avoids repeating context, optimizing for token efficiency.
5. **Language-agnostic, implementation-specific.** The protocol is the same for every language. The backend wires it to the right toolchain.

---

## 1. Transport Layer

PAWS is transport-agnostic. It can run over:

- **stdio** (JSON-RPC 2.0 over stdin/stdout) — for local CLI agents
- **HTTP** (Streamable HTTP) — for remote or web-based agents
- **Any MCP-compatible transport** — PAWS messages are valid MCP tool calls

A PAWS server MAY be implemented as an MCP server with PAWS-compliant tool schemas. A dedicated PAWS transport MAY extend MCP with additional message types (e.g., `workspace/diagnostics` push notifications).

---

## 2. Lifecycle

### 2.1 Two-Phase State Machine

A PAWS server operates in two phases:

```
┌─────────┐  open_project   ┌──────────┐
│  LOBBY  │ ───────────────→ │  PROJECT │
│         │ ←─────────────── │          │
└─────────┘  close_project   └──────────┘
```

**Lobby Phase**: The server is running but no project context is established. Only lifecycle tools are available:
- `open_project(dir)` — Detect languages, initialize backends, start indexing.
- `create_project(dir, language, dependencies)` — Bootstrap a new project, then transition to Project phase.

**Project Phase**: Full tool suite is available. The server has:
- A `project_root` path
- One or more active language backends
- An optional semantic search index
- Active LSP clients for each detected language

The server MUST dynamically register/unregister tools on phase transitions. The agent SHOULD call `open_project` before attempting any file or language operations.

### 2.2 Capability Discovery

On entering the Project phase, the server advertises its capabilities — the union of features across all active language backends.

```
Capability          Tools Enabled
─────────────────   ──────────────────────────────────
lsp                 describe, find_definition,
                    find_references, rename_symbol
toolchain           build
documentation       read_docs
dependencies        add_dependencies
mutation_testing    test_mutations
test_query          query_tests
semantic_search     semantic_search  (also requires index)
code_review         review_code      (also requires auth)
```

Tools gated by a capability that no active backend supports MUST NOT be registered. The agent MUST NOT see tools it cannot use.

---

## 3. File Operations

### 3.1 `read_file`

Read a file with optional semantic augmentation.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Path to the file |
| `start_line` | int | no | Start of range (1-based) |
| `end_line` | int | no | End of range (1-based) |
| `outline` | bool | no | Return structural skeleton only |

**Semantic Augmentation (PAWS-specific):**

When reading a full file (no line range), the server SHOULD append:

1. **Type Info Footer**: For each non-trivial identifier in the file, the server resolves its type signature via LSP hover. Standard library types are excluded. Types already shown in the current session are deduplicated via a session-level cache (`seenTypeInfo`).

2. **Imported Packages Section**: Brief documentation summaries for third-party imports, deduplicated per session.

This eliminates multi-tool round-trips where the agent would otherwise need to `find_definition` → `read_file` for every unfamiliar type.

### 3.2 `edit_file`

Apply a targeted edit to a file with fuzzy matching and immediate validation.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Path to the file |
| `old_content` | string | yes | Content to find and replace |
| `new_content` | string | yes | Replacement content |

**Behavior:**

1. **Fuzzy Match**: The server normalizes whitespace and uses seed-based candidate detection with Levenshtein scoring to locate `old_content`, tolerating minor formatting drift. If multiple ambiguous matches are found, the server rejects the edit and returns the candidates.

2. **LSP Validation Pipeline**: After splicing the new content:
   - `didOpen` → `didChange` → `OrganizeImports` → `Format` → write to disk → `didSave`
   - The server returns diagnostics (errors, warnings) from the language server **in the same response**.

3. **RAG Re-Indexing**: The modified file is synchronously re-ingested into the semantic search index.

This is the **Quality Gate** — the defining feature of PAWS. Every edit is validated before the agent sees "success."

### 3.3 `multi_edit`

Atomic batch edit across multiple files.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `edits` | array | yes | Array of `{file, old_content, new_content}` |

Each edit follows the same fuzzy match and validation pipeline. A unified diagnostic report covers all modified files. This prevents transient LSP errors from intermediate states during cross-file refactors.

### 3.4 `create_file`

Create a new file with language-aware initialization.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Path to create |
| `content` | string | yes | Initial content |

The server runs the full LSP lifecycle (open → organize imports → format → write → diagnostics → close) and re-indexes the file for semantic search.

### 3.5 `list_files`

List project files with intelligent filtering.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | no | Directory to list (default: project root) |
| `depth` | int | no | Maximum depth (default: 5) |

The server uses git-aware listing when available (respecting `.gitignore`), falling back to a directory walk with backend-aware skip directories.

---

## 4. Language Intelligence

### 4.1 `describe`

Enhanced hover — return type information for a symbol.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Path to the file |
| `line` | int | yes | Line number (1-based) |
| `col` | int | yes | Column number (1-based) |

Returns the LSP hover result augmented with resolved type definitions. Unlike `read_file`'s type info footer, `describe` always returns full detail (no deduplication) — it is an explicit request for deep context.

### 4.2 `find_definition`

Jump to a symbol's definition.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Source file |
| `line` | int | yes | Line number |
| `col` | int | yes | Column number |

Returns the file path, line, and column of the definition.

### 4.3 `find_references`

Find all usages of a symbol.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | Source file |
| `line` | int | yes | Line number |
| `col` | int | yes | Column number |
| `include_declaration` | bool | no | Include the declaration itself (default: true) |

Returns locations grouped into `[SOURCE]` and `[TESTS]` sections, with relative paths and containing symbol context. This categorization helps the agent assess the impact of a change.

### 4.4 `rename_symbol`

AST-driven project-wide rename.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | yes | File containing the symbol |
| `line` | int | yes | Line number |
| `col` | int | yes | Column number |
| `new_name` | string | yes | New identifier name |

The server sends an LSP `textDocument/rename` request, applies the resulting workspace edit atomically across all files, and validates the result. This is **never** a regex find-and-replace — it is type-safe and scope-aware.

---

## 5. Semantic Search

### 5.1 `semantic_search`

Intent-based code discovery via RAG.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | yes | Natural language query |
| `limit` | int | no | Maximum results (default: 5, max: 10) |

The server embeds the query, performs cosine similarity search against the project's vector index, and returns matching code snippets with file paths, line numbers, and similarity scores.

This is fundamentally different from `grep` — it finds code by **meaning**, not by text pattern.

---

## 6. Quality Gates

### 6.1 `build`

Run the full quality pipeline: compile → test → lint.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `packages` | string | no | Package pattern (default: all) |
| `auto_fix` | bool | no | Run formatters/fixers before quality checks |
| `run_tests` | bool | no | Include test suite (default: true) |
| `run_lint` | bool | no | Include linter (default: true) |

Returns a structured Markdown report with sections for build status, test results (with coverage), and lint output.

### 6.2 `test_mutations`

Measure test suite quality via mutation testing.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `dir` | string | no | Directory to test (default: project root) |

The backend introduces small logical mutations and reports which ones survived (tests didn't catch them). This is an objective quality metric.

### 6.3 `query_tests`

SQL interface to test and coverage data.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | yes | SQL query |
| `rebuild` | bool | no | Force rebuild of test database |

The server maintains a SQLite database with four tables:
- `all_tests` — test results (package, test, action, elapsed, output)
- `all_coverage` — coverage data (file, function, start_line, end_line, count)
- `test_coverage` — per-test coverage mapping
- `all_code` — searchable source code (file, line_number, content)

---

## 7. Documentation & Dependencies

### 7.1 `read_docs`

Retrieve documentation for a package or symbol.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `import_path` | string | yes | Package to look up |
| `symbol` | string | no | Specific symbol within the package |
| `format` | string | no | `markdown` (default) or `json` |

### 7.2 `add_dependencies`

Install packages with the **"Get and Learn"** pattern.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `packages` | array | yes | List of packages to install |

After installation, the server immediately fetches and returns the API documentation for each installed package. The agent can begin using the package in the next turn without a separate `read_docs` call.

### 7.3 `review_code`

AI-assisted code review.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file` | string | no | Path to review |
| `file_content` | string | no | Raw source to review |
| `hint` | string | no | Focus area for the review |

Returns structured suggestions with line numbers, severity levels, and explanations.

---

## 8. Language Backend Contract

A PAWS server is language-agnostic at the protocol level. Language-specific behavior is provided by **backends** that implement a standard interface:

```
Backend Interface
├── Identity: name, languageId, extensions, markers, tier
├── LSP: command, args, initOptions
├── Setup: ensureTools(dir)
├── File Ops: validate, format, outline, parseImports, isStdLib
├── Quality: buildPipeline, modernize, mutationTest
├── Testing: buildTestDB, queryTestDB
├── Docs: fetchDocs, importDocs, addDependency
└── Project: initProject
```

A PAWS server MUST support adding backends via:
1. **Built-in implementations** (e.g., Go, Python)
2. **Plugin system** (JSON-configured backends for arbitrary languages)

### 8.1 External Tool Management

Backends manage their own tool dependencies using the language's native mechanism:

| Language | Mechanism | Pinning |
|----------|-----------|---------|
| Go | `go tool` (tool directives in `go.mod`) | Version pinned in `go.mod` |
| Python | `uv run` (project virtual environment) | Version in `pyproject.toml` |
| Plugin | Assumed on PATH | N/A |

Tools are installed lazily on `open_project` via `ensureTools()`. Failures are warnings, not errors.

---

## 9. Session State

A PAWS server maintains session-level state to optimize for token efficiency:

| State | Purpose | Reset On |
|-------|---------|----------|
| `seenTypeInfo` | Dedup type signatures across reads | `close_project` |
| `seenDocs` | Dedup import documentation across reads | `close_project` |
| `ragIndex` | Semantic search vector database | `close_project` |
| `activeBackends` | Detected language backends | `close_project` |

This session awareness is a key differentiator from stateless tool servers. A PAWS server **remembers what it has told the agent** and avoids wasting tokens on repetition.

---

## 10. Security Model

### Root Validation

All file operations are validated against a set of allowed roots. The roots are **client-declared** — the agent (or its host) specifies which directories the server may access. The server MUST NOT access files outside the declared roots.

### Credential Isolation

Tools that require external credentials (e.g., `review_code` requiring GenAI API keys) MUST perform conditional registration: if credentials are absent, the tool is not exposed. The agent never sees a tool it cannot use.

---

## Appendix A: PAWS vs. MCP Comparison

| Aspect | MCP | PAWS |
|--------|-----|------|
| Scope | General-purpose tool protocol | Coding-specific protocol |
| Lifecycle | Stateless | Two-phase (Lobby → Project) |
| Tool discovery | Static list | Capability-based, dynamic |
| File edits | Raw text replacement | Fuzzy match + LSP validation |
| Error feedback | Tool-specific | Guaranteed (Quality Gate) |
| Code navigation | Not specified | LSP-backed (definition, references, rename) |
| Search | Not specified | RAG-backed semantic search |
| Session state | None | Type info dedup, doc dedup, search index |
| Language support | N/A | Backend interface + plugin system |

## Appendix B: Naming

**PAWS** — Protocol for Agent Workspace Services.

The name is a nod to the Neko project (neko = cat in Japanese). A cat's paws are precise, gentle, and effective instruments — qualities we want in a coding protocol.

Alternative names considered:
- **AIDE** (Agent IDE Extensions) — emphasizes the IDE analogy
- **SAGE** (Semantic Agent Gateway Extensions) — emphasizes semantic intelligence
- **CASE** (Code Agent Semantic Extensions) — emphasizes the coding focus
