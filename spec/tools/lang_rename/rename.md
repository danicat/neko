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

3. **File Application**:
   - Neko iterates through all file changes sequentially.
   - For each file, it reads the current content, applies text edits via `lsp.ApplyTextEdits`, writes the result to disk, and sends a `didSave` notification.
   - After all files are written, it calls `WaitForDiagnostics` on each modified file to capture any errors caused by the rename (e.g., name collisions).

4. **Return**:
   - Returns a report detailing exactly which files and lines were updated, providing confidence to the LLM.
