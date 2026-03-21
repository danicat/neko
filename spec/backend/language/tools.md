# External Tool Management

## Strategy

Neko relies on external CLI tools for advanced features like mutation testing, code modernization, and test coverage analysis. Each language ecosystem has its own native mechanism for managing tool dependencies. Neko follows a single principle:

> **Use the language's native tool management, pinned in the project's dependency file.**

This means:
- **Go**: Tools are declared as `tool` directives in the project's `go.mod` (Go 1.24+) and invoked via `go tool <name>`.
- **Python**: Tools are installed into the project's virtual environment and invoked via `uv run <name>`.
- **Plugins**: Tools are assumed to be pre-installed and available on PATH.

## Lifecycle

The `EnsureTools(ctx, dir) error` method on each `LanguageBackend` is called once during `open_project`, after backend detection. It installs any missing tools into the project's dependency file.

If `EnsureTools` fails (e.g., no network), the error is logged as a warning but does not block project opening. Tools that depend on the missing binary will fail individually when invoked, providing a clear error message at that point.

## Go Tools

The Go backend requires three external tools:

| Tool | Module Path | Purpose |
|------|-------------|---------|
| `golangci-lint` | `github.com/golangci/golangci-lint/cmd/golangci-lint` | Linting |
| `selene` | `github.com/danicat/selene/cmd/selene` | Mutation testing |
| `testquery` | `github.com/danicat/testquery` | Test coverage SQL database |
| `modernize` | `golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize` | Code modernization analysis |

`EnsureTools` runs `go get -tool <module>` for each tool, which:
1. Adds a `tool` directive to the project's `go.mod` (if not already present).
2. Downloads and caches the binary via the standard Go module cache.
3. Makes the tool available via `go tool <name>`.

All Go external tools are then invoked as `go tool <name> [args...]` — never `go run <module>@latest`.

### Why not `go run`?

`go run <module>@latest` has three problems:
1. **Non-deterministic**: `@latest` resolves to whatever version is newest at invocation time.
2. **Slow**: Downloads the module on every invocation (unless already cached by accident).
3. **No project-level pinning**: The version is not recorded anywhere, so different machines may run different versions.

`go tool` solves all three: the version is pinned in `go.mod`, cached by the Go toolchain, and consistent across all machines that share the same `go.mod`.

## Python Tools

Python tools (e.g., `ruff`, `mutmut`) are managed by `uv` and installed into the project's virtual environment. `uv run <tool>` resolves the tool from the venv, similar to how `go tool` resolves from `go.mod`. The Python backend's `EnsureTools` is a no-op because `uv run` handles installation transparently.

## Plugin Tools

Plugin backends assume all required tools are pre-installed on PATH. Their `EnsureTools` is a no-op.
