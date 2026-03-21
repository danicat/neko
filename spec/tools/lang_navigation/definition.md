# Go-to-Definition Logic

## Overview
The `find_definition` tool (`internal/tools/lang/definition/definition.go`) allows the LLM to traverse the codebase graph by jumping to symbol definitions.

## Implementation Details

1. **LSP Request**:
   - Triggers `textDocument/definition` using the provided file, line, and column.
   - Returns the absolute file path, line, and column where the symbol under the cursor was defined.

2. **Error Handling**:
   - Validates the file path against `roots.Global.Validate` before making the LSP request.
   - Checks that the LSP server binary is available on PATH before attempting connection.
   - Returns a clear error if no definition is found or the LSP server is unavailable.

> **Note**: The `find_references` tool is documented separately in [lang_references](../lang_references/references.md).
