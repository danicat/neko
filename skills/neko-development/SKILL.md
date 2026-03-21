---
name: neko-development
description: Specialized workflows for language-aware development and refactoring using Neko. Use when performing structural changes, project-wide renames, or intent-based discovery. Provides guidance for semantic refactoring and full project health navigation.
---

# Neko Development Skill

This skill provides precise, language-aware workflows for using **Neko** as a Semantic Operating System.

## Core Development Workflow (v0.2.0)

1.  **Context**: Call `open_project(dir=".")` to initialize the semantic engine.
2.  **Explore**: Use `semantic_search` to find patterns by *meaning* when keywords aren't enough.
3.  **Intel**: Use `read_file` and consult the **Type Info** footer for immediate type signatures and struct fields. Use `describe` for deep contextual analysis.
4.  **Action**: 
    - Use `rename_symbol` for all deterministic renames.
    - Use `multi_edit` for interdependent changes across files.
    - Use `edit_file` for surgical logic updates.
5.  **Navigate**: Use the **Full Disclosure** diagnostic report returned by modification tools as your todo list for the next turn.
6.  **Verify**: Run `build()` for final validation.

---

## 🐹 Go Development (Golang)

Go projects use standard toolchains and the high-fidelity `gopls` LSP.

### Refactoring Nuances
- **Rename**: `rename_symbol` is the authoritative way to refactor Go symbols project-wide.
- **Build**: Auto-fix triggers a `workspace/didChangeWatchedFiles` sync.
- **Test Querying (`query_tests`)**: **Go-Specific**. Query `go test` results using SQL.

---

## 🐍 Python Development

Python projects **must** use `uv` for all operations.

### Refactoring Nuances
- **Rename**: Supported via `rename_symbol` (requires `pylsp` or `pyright`).
- **Lint/Format**: Automated via `ruff` during `edit_file` saves.
- **Modernize**: Uses `ruff` with `pyupgrade` rules.

---

## Pro-Tips for Agents

- **Semantic Momentum**: Use the diagnostic list returned by every edit to steer your next action. Do not wait for a full `build` if the LSP already shows regressions.
- **Type Awareness**: Trust the **Type Info** footer in `read_file`. It provides the compiler's view of variable types, struct fields, and method sets.
- **Atomic Batches**: Submit "Complete Thoughts" using `multi_edit` to keep the diagnostic list actionable.
