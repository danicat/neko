# Symbol Info Logic (Describe)

## Overview
The `describe` tool (`internal/tools/lang/symbolinfo/symbolinfo.go`, package `describe`) provides enhanced hover information — the IDE's "hover" feature augmented with deep type resolution.

## Implementation Details

1. **LSP Request**:
   - Sends a `textDocument/hover` request via `client.EnhancedHover()` for a specific file, line, and column.

2. **Enhanced Hover Processing**:
   - `EnhancedHover` augments the standard hover result with a **one-level definition lookup heuristic**.
   - If the initial hover text is short (< 200 characters), it attempts to jump to the symbol's definition via `Definition()` and fetches the hover there. If the definition's hover is richer (longer), it returns that instead — giving the LLM struct/interface details rather than just a variable declaration.
   - For `var` declarations, it prepends the original signature to the definition's hover for full context.
   - Note: The `HasSeenTypeInfo` method is declared on the Server interface but is not currently used by this tool — deduplication only applies to the `read_file` Type Info footer.

3. **Output**:
   - Returns a structured Markdown response containing the hover content plus any resolved type information.
   - This allows the LLM to understand the inputs and outputs of a function, the fields of a struct, and the full type context — without needing to open the files where those types are defined.
