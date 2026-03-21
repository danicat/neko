# Build Pipeline Logic

## Overview
The `build` tool (`internal/tools/lang/quality/build.go`) forces the LLM to verify its work against the compiler, linter, and test suite.

## Implementation Details

1. **Delegation**:
   - Routes the request to the active `LanguageBackend.BuildPipeline`.

2. **Auto-Fixing**:
   - Supports an `auto_fix` parameter. If true, the backend runs formatting or modernization routines (like `go fmt` or `ruff format`) before executing the quality checks.

3. **Standardized Reporting**:
   - The tool does not return raw terminal output (which can be thousands of lines).
   - The backend parses the output into a standardized `BuildReport` struct containing structured warnings and errors.
   - The tool formats this into a succinct Markdown checklist, highlighting exactly what failed so the LLM can fix it in the next turn.
