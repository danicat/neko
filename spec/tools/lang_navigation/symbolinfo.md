# Symbol Info Logic (Describe)

## Overview
The `describe` tool (`internal/tools/lang/symbolinfo/symbolinfo.go`) acts as the IDE's "hover" feature.

## Implementation Details

1. **LSP Request**:
   - Sends a `textDocument/hover` request to the language server for a specific line/column.

2. **Parsing**:
   - Language servers typically return hover information as heavily formatted Markdown, often wrapping type signatures in code blocks and including parsed docstrings.

3. **Output**:
   - Neko returns this Markdown directly to the LLM. This allows the LLM to understand the inputs and outputs of a function, or the fields of a struct, without needing to open the file where that symbol is defined.
