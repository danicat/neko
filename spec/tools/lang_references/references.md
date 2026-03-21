# Find References Logic

## Overview
The `find_references` tool (`internal/tools/lang/references/references.go`) locates all usages of a symbol across the codebase via LSP. It post-processes results to categorize them by context, helping the LLM decide whether a change affects business logic, tests, or both.

## Implementation Details

1. **LSP Request**:
   - Sends a `textDocument/references` request using the provided file, line, and column.
   - Supports an `include_declaration` parameter (default `true`) to control whether the symbol's own declaration is included in the results.

2. **Enrichment** (`client.EnrichLocations`):
   - Post-processes the raw LSP location array into a categorized, human-readable list.
   - **Relative paths**: Converts absolute file paths to paths relative to the workspace root for readability.
   - **Symbol context**: For each location, calls `GetSymbolAt` (via `textDocument/hover`) to extract the containing symbol's signature. `GetSymbolAt` skips markdown code fence lines (e.g., ` ```go `) to avoid misidentifying the language tag as the symbol name.
   - **SOURCE / TESTS grouping**: Locations are classified into `[SOURCE]` and `[TESTS]` sections based on file naming:
     - Files ending with `_test.go` → TESTS (uses `strings.HasSuffix`, not `Contains`)
     - Files starting with `test_` (e.g., `test_*.py`) → TESTS
     - All others → SOURCE

3. **Output**:
   - Returns a count header (`Found N reference(s)`) followed by the categorized list.
   - Each entry includes: relative path, line, column, and the containing symbol signature (if resolved).
