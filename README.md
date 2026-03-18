# Neko 🐈

Neko is an intelligent, language-aware Model Context Protocol (MCP) server designed to empower AI agents with professional-grade software engineering tools.

It standardizes code exploration, precision editing, and verification across multiple languages, ensuring that every code change is validated through a rigorous "Quality Gate".

## 🚀 Quick Start

### Prerequisites
- **Go 1.22+** (to build the server)
- **uv** (required for all Python operations)
- **Node.js & npm** (for JavaScript/TypeScript support)

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

## 🧠 Core Philosophy

1.  **Explore First**: Systematically map the codebase before acting.
2.  **Precise Editing**: Targeted, surgical updates with syntax verification.
3.  **The Quality Gate**: Build -> Test -> Lint must pass for every change.
4.  **Idiomatic Excellence**: Leveraging modern patterns (e.g., `uv` for Python, `go mod` for Go).

## 🛠️ Tool Highlights

- **`smart_read`**: AST-aware reading to maximize context efficiency.
- **`smart_edit`**: Robust fuzzy matching for safe, multi-file editing.
- **`smart_build`**: The universal quality gate for all supported languages.
- **`project_init`**: Idiomatic project bootstrapping (e.g., `uv init`, `npm init`).
- **LSP Integration**: Deep intelligence via `gopls` and `typescript-language-server`.

## 📚 Documentation

For extensive details on setup, internal architecture, and how to extend Neko with new languages using the plugin system, see [DOCUMENTATION.md](./DOCUMENTATION.md).

## 🧩 Extension System

Neko is highly extensible. Add support for new languages by simply dropping a JSON configuration into the `plugins/` directory. No Go code changes required.

---
*Built with ❤️ by engineers for engineers.*
