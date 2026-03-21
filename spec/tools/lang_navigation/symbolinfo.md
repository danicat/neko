# Symbol Info Logic (Describe)

## Overview
The `describe` tool (`internal/tools/lang/symbolinfo/symbolinfo.go`, package `describe`) provides enhanced hover information — the IDE's "hover" feature augmented with deep type resolution.

## Implementation Details

1. **LSP Request**:
   - Sends a `textDocument/hover` request via `client.EnhancedHover()` for a specific file, line, and column.

2. **Enhanced Hover Processing**:
   - Unlike a raw hover passthrough, `EnhancedHover` augments the standard hover result with **type info resolution**.
   - It follows type references in the hover content, recursively resolving definitions to build a complete picture of the symbol's type graph.
   - The `seenTypeInfo` deduplication map (on the Server, accessed via `HasSeenTypeInfo`) ensures that type signatures already shown to the LLM in the current session are not repeated.

3. **Output**:
   - Returns a structured Markdown response containing the hover content plus any resolved type information.
   - This allows the LLM to understand the inputs and outputs of a function, the fields of a struct, and the full type context — without needing to open the files where those types are defined.
