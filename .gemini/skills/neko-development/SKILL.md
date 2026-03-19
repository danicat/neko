---
name: neko-development
description: Specialized workflows for language-aware development using Neko. Use when building, refactoring, or testing Go, Python, or JS/TS projects. Provides language-specific guidance for tools like build, query_tests, and modernize.
---

# Neko Development Skill

This skill provides precise, language-aware workflows for using **Neko**. Neko's power lies in its deep integration with language toolchains (Go, Python, JS/TS).

## Core Development Workflow (All Languages)

1.  **Context**: Always call `open_project(dir=".")` first to initialize LSPs and backends.
2.  **Explore**: Use `read_file(file="...", outline=true)` for AST-based structural summaries.
3.  **Intel**: Use `describe`, `find_definition`, and `read_docs` for compiler-level symbol info.
4.  **Edit**: Use `edit_file` with `start_line` and `end_line`. Neko validates syntax and applies formatting (gofmt/ruff) before saving.
5.  **Verify**: Run `build()` to trigger the unified quality gate (Compile -> Modernize -> Test -> Lint).

---

## 🐹 Go Development (Golang)

Go projects use standard toolchains (`go` command) and advanced SQL-based test analysis.

### Toolchain Nuances
- **Build**: Runs `go build`, `go mod tidy`, and `gofmt`.
- **Testing**: Uses `go test -v -cover`. Neko parses coverage at the package level.
- **Modernize**: Analyzes code for outdated patterns (e.g., pre-Go 1.18 generics or old `io/ioutil` usage).

### Advanced Go Analysis
- **Test Querying (`query_tests`)**: **Go-Specific**. Allows querying `go test` results and coverage data using SQL.
  - *Example*: `SELECT package, coverage FROM all_coverage WHERE coverage < 50`
- **Mutation Testing**: Uses `go-mutesting` to find "survivors" (code changes that tests don't catch).

---

## 🐍 Python Development

Python projects **must** use `uv` for all operations. Neko enforces this to ensure deterministic environments.

### Toolchain Nuances
- **Environment**: All commands run via `uv run`. If `uv` is missing, the backend will fail.
- **Lint/Format**: Uses `ruff` for extremely fast linting and formatting.
- **Type Checking**: Runs `mypy` automatically during `build()`.
- **Testing**: Uses `pytest`. Neko captures `passed`/`failed` summaries from the pytest output.
- **Modernize**: Uses `ruff` with `pyupgrade` rules (`UP` category) to refactor code for newer Python versions.

### ⚠️ Python Limitations
- **Test Querying**: SQL querying via `query_tests` is **not supported** for Python. Test data is collected via pytest JSON reports but is not searchable via SQL.

---

## 📜 JavaScript / TypeScript Development

Neko supports JS/TS primarily through a plugin system.

### Toolchain Nuances
- **Package Manager**: Detects `npm`, `yarn`, or `pnpm` automatically based on lockfiles.
- **Modernize**: If configured, uses `eslint` or `prettier` for code standards.
- **Testing**: Typically uses `jest` or `vitest`.

---

## Pro-Tips for Agents

- **Token Efficiency**: Always prefer `outline: true` in `read_file` for initial exploration.
- **Safety Gate**: If `edit_file` reports a syntax error, do not attempt to force the write. Fix the syntax in the `new_content` block and retry.
- **On-Touch Activation**: Neko backends activate "on-touch." Reading a `.go` file in a generic project will automatically spin up the Go backend and LSP.
