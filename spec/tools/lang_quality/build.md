# Build Pipeline Logic

## Overview
The `build` tool (`internal/tools/lang/quality/build.go`) forces the LLM to verify its work against the compiler, linter, and test suite.

## Implementation Details

1. **Delegation**:
   - Routes the request to the active `LanguageBackend.BuildPipeline`.

2. **Auto-Fixing**:
   - Supports an `auto_fix` parameter. If true, the backend runs formatting or modernization routines before executing quality checks.
   - For Go, the auto-fix pipeline runs in order: `go mod tidy` → `gofmt -w .` → build → test → lint.
   - If `gofmt` fails, the error is logged as a warning in the output rather than silently ignored.

3. **Markdown Reporting**:
   - The backend constructs a Markdown-formatted string directly (not a struct), organizing results into sections:
     - **Build status**: Compiler errors with file/line references.
     - **Test results**: Pass/fail summary with coverage percentage when available.
     - **Lint output**: Warnings and suggestions from `golangci-lint` (via `go tool`).
   - This succinct Markdown report highlights exactly what failed so the LLM can fix it in the next turn, without flooding the context with thousands of lines of raw terminal output.
