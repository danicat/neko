# Neko

Neko is an intelligent, language-aware Model Context Protocol (MCP) server designed to empower AI agents with professional-grade software engineering tools.

It standardizes code exploration, precision editing, and verification across multiple languages, ensuring that every code change is validated through a rigorous Quality Gate.

## Quick Start

### Prerequisites
- **Go 1.22+** (to build the server)
- **uv** (required for all Python operations)
- **Node.js & npm** (for JavaScript/TypeScript support via plugin)

### Installation
1. **Build the binary:**
   ```bash
   make build
   ```
2. **Add to your MCP configuration:**
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

## Core Philosophy

1.  **Project First**: Establish a project context before working. Neko operates in two phases — a Lobby for opening/creating projects, and a Project phase with the full toolset.
2.  **Explore First**: Systematically map the codebase before acting.
3.  **Precise Editing**: Targeted, surgical updates with syntax verification.
4.  **The Quality Gate**: Build -> Test -> Lint must pass for every change.
5.  **Language is a Property of the File**: Backends are resolved per-file, not per-directory, enabling seamless polyglot support.

## Project Lifecycle

Neko uses a two-phase lifecycle to ensure the AI agent always has a valid project context:

```
[LOBBY]  ──open_project──►  [PROJECT OPEN]  ──close_project──►  [LOBBY]
         ──create_project──►
```

- **Lobby**: Only `open_project` and `create_project` are available.
- **Project Open**: The full toolset is dynamically registered based on the languages detected in the project.

## Tool Highlights

| Tool | Purpose |
|------|---------|
| `read_file` | AST-aware reading with outline mode to maximize context efficiency |
| `edit_file` | Robust fuzzy matching for safe editing with syntax validation |
| `build` | The universal quality gate for all supported languages |
| `create_project` | Idiomatic project bootstrapping (e.g., `uv init`, `npm init`) |
| `describe` | LSP-powered type info and documentation via `gopls`, `pylsp`, etc. |
| `review_code` | AI-powered idiomatic code review |

## Documentation

For setup details, architecture, and how to extend Neko with new languages using the plugin system, see [DOCUMENTATION.md](./DOCUMENTATION.md).

## Extension System

Neko is highly extensible. Add support for new languages by dropping a JSON configuration file into the `plugins/` directory. No Go code changes required.
