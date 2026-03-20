# Task 4: LSP-Aware Modification Chain

## Context
This task integrates the LSP lifecycle into `edit_file` and `create_file`. It moves Neko from "blind writes" to "semantic-aware commits."

## TODO
- [ ] Implement the sequence in `edit_file`: `didOpen` (disk) -> `didChange` (proposed) -> Save Actions -> `WriteFile` -> `didSave` -> `WaitForDiagnostics`.
- [ ] Implement the sequence in `create_file`: `WriteFile` -> `didChangeWatchedFiles` -> `didOpen` -> `didSave` -> `WaitForDiagnostics`.
- [ ] Implement "Automated Save Actions": Chain `codeAction` (organizeImports) and `formatting` before the final disk write.
- [ ] Update tool responses to use the standardized Markdown Diagnostic template.

## NOT TODO
- [ ] Do not implement complex rollback logic; rely on the user's Git repository.
- [ ] Do not skip any step in the LSP lifecycle to "save time."

## Acceptance Criteria
- [ ] Every edit/creation returns a full project diagnostic list.
- [ ] Imports are automatically cleaned up on save if the LSP supports it.
- [ ] The LSP's internal model is always synchronized with the disk after a tool call.
