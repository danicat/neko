# Rename Symbol Logic

## Overview
The `rename_symbol` tool (`internal/tools/lang/rename/rename.go`) is Neko's mandatory mechanism for refactoring identifiers. Because regular expression replacements are dangerous across complex codebases, this tool relies entirely on the language compiler's AST.

## Implementation Details

1. **LSP Request**:
   - The user provides the target file, line, column, and the `new_name`.
   - Neko sends a `textDocument/rename` request to the LSP.

2. **Workspace Edit Evaluation**:
   - The LSP returns a `WorkspaceEdit` structure containing a map of file URIs to an array of precise `TextEdit` objects (ranges and new text).

3. **Atomic Application**:
   - Neko iterates through all the file changes specified in the `WorkspaceEdit`.
   - It opens each file in memory, applies the edits in reverse-range order (to prevent offset shifting), and sends `didChange` notifications.
   - It captures diagnostics for all modified files to ensure the rename didn't cause collisions.
   - If successful, it commits all modified files to disk simultaneously.

4. **Return**:
   - Returns a report detailing exactly which files and lines were updated, providing confidence to the LLM.
