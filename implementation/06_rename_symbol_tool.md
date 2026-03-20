# Task 6: Rename Symbol Tool

## Context
Renaming symbols is a high-frequency refactoring task. Doing it manually is error-prone. This tool leverages the LSP's built-in `textDocument/rename` to perform the task deterministically.

## TODO
- [ ] Create the `rename_symbol` MCP tool.
- [ ] Implement `lsp.Client.Rename(file, line, col, newName)` using the `textDocument/rename` request.
- [ ] Handle the resulting `WorkspaceEdit` by applying changes to all affected files on disk.
- [ ] Trigger `didSave` notifications for every modified file.
- [ ] Return the final Project Diagnostic Snapshot.

## NOT TODO
- [ ] Do not implement a custom search-and-replace; use the LSP's mapping exclusively.
- [ ] Do not rename without user confirmation (handled by MCP tool call).

## Acceptance Criteria
- [ ] Symbols are renamed across the entire project in a single turn.
- [ ] Imports and references are updated correctly by the LSP.
- [ ] The final health report confirms if the rename caused any regressions.
