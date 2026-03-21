# Rename Symbol Logic

## Overview
The `rename_symbol` tool (`internal/tools/lang/rename/rename.go`) is Neko's mandatory mechanism for refactoring identifiers. Because regular expression replacements are dangerous across complex codebases, this tool relies entirely on the language compiler's AST.

## Implementation Details

1. **LSP Request**:
   - The user provides the target file, line, column, and the `new_name`.
   - Neko sends a `textDocument/rename` request to the LSP.

2. **Workspace Edit Processing**:
   - The LSP returns a `WorkspaceEdit` structure. Per the LSP specification, edits may be provided in two formats:
     - **`DocumentChanges`** (preferred): An array of `TextDocumentEdit` objects, each containing a versioned document identifier and an array of `TextEdit` entries. This is the format preferred by modern language servers (e.g., gopls).
     - **`Changes`** (legacy fallback): A map of file URIs to arrays of `TextEdit` objects.
   - Neko processes `DocumentChanges` first. If empty, it falls back to `Changes`.
   - If **both** are empty, the tool returns an error (`"rename produced no changes"`) instead of silently reporting success.

3. **Atomic Application**:
   - Neko iterates through all the file changes.
   - It opens each file in memory, applies the edits in reverse-range order (to prevent offset shifting), and sends `didChange` notifications.
   - It captures diagnostics for all modified files to ensure the rename didn't cause collisions.
   - If successful, it commits all modified files to disk simultaneously.

4. **Return**:
   - Returns a report detailing exactly which files and lines were updated, providing confidence to the LLM.
