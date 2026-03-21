# Go Implementation

## Overview
The Go language backend (`internal/backend/golang`) leverages the standard Go toolchain and integrates with `gopls` for LSP support.

## Implementation Details

1. **Detection**:
   - Matches if `go.mod` exists in the workspace.

2. **LSP Binding**:
   - `LSPCommand()` returns `gopls` (no extra arguments). LSP initialization options (semantic tokens, hover, staticcheck diagnostics) are provided separately via `InitializationOptions()`.

3. **Tool Setup** (`EnsureTools`):
   - On `open_project`, installs required tools into the project's `go.mod` via `go get -tool`:
     - `golangci-lint` — linting
     - `selene` — mutation testing
     - `testquery` — test coverage SQL database
     - `modernize` — code modernization analysis
   - See [External Tool Management](tools.md) for the full strategy.

4. **BuildPipeline**:
   - Executes in order: `go mod tidy` → `gofmt` → `go build` → `go test -cover` → lint.
   - Lint uses `go tool golangci-lint` (installed via `EnsureTools`).
   - Results are formatted as a structured Markdown report.

5. **Modernization & Formatting**:
   - `Format()` uses `goimports` (via `golang.org/x/tools/imports`) for formatting and import organization.
   - `Modernize()` invokes `go tool modernize` to detect and optionally auto-fix legacy Go patterns.

6. **Mutation Testing**:
   - Delegates to `go tool selene` for AST-level mutation testing.

7. **Test Coverage Database**:
   - `BuildTestDB` invokes `go tool testquery build` to run tests with coverage and ingest results into a SQLite database.
   - `QueryTestDB` invokes `go tool testquery query` for SQL-based coverage analysis.
