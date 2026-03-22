# Neko

Neko is a **Semantic Operating System** for AI agents, built on the Model Context Protocol (MCP). It moves beyond simple file manipulation by providing a deterministic, LSP-first refactoring engine and intent-based discovery via local RAG.

It standardizes code exploration, precision editing, and verification across multiple languages, ensuring that every code change is validated through a rigorous **Full Disclosure** diagnostic loop.

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
         "command": "/path/to/neko/neko",
         "args": ["-plugins", "/path/to/neko/plugins"]
       }
     }
   }
   ```

## Core Philosophy

1.  **Semantic Integrity**: Every edit or creation triggers a synchronous re-evaluation of the entire project health via the Language Server Protocol (LSP).
2.  **Full Reality Disclosure**: Neko never prunes or hides errors. The agent is always confronted with the *Total Reality* of the project's semantic state.
3.  **Intent-Based Discovery**: Local Retrieval-Augmented Generation (RAG) allows agents to find patterns and logic by *meaning* rather than just keywords.
4.  **Turn-Level Certainty**: Neko tools block until the semantic outcome of an action is known, ensuring the agent's next thought is based on accurate reality.

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
| `edit_file` | LSP-aware editing with automated formatting, import cleanup, and synchronous diagnostics. |
| `multi_edit` | Transactional batch editing across multiple files in a single turn. |
| `rename_symbol` | Deterministic, project-wide symbol renaming via LSP. |
| `semantic_search` | Intent-based discovery using a local vector database (`chromem-go`). |
| `read_file` | Enhanced reading with virtual semantic annotations (inline type signatures). |
| `build` | The universal quality gate with automatic LSP synchronization. |
| `describe` | Contextual type info and documentation via LSP `hover`. |

## Documentation

For setup details, architecture, and how to extend Neko with new languages using the plugin system, see [DOCUMENTATION.md](./DOCUMENTATION.md).

## Extension System

Neko is highly extensible. Add support for new languages by dropping a JSON configuration file into the `plugins/` directory. No Go code changes required.
