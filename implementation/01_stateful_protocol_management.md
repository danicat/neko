# Task 1: Stateful Protocol Management

## Context
Neko v0.2.0 requires the `lsp.Client` to be stateful to support save actions and synchronous diagnostics. We need to track document versions sequentially to comply with the LSP specification and ensure the server's virtual state matches our proposed edits.

## TODO
- [ ] Add `openedDocs map[string]int` to `lsp.Client` to track URI to version.
- [ ] Implement version reset to 1 on `textDocument/didOpen`.
- [ ] Implement version increment on `textDocument/didChange` and `textDocument/didSave`.
- [ ] Add a `GetVersion(uri string) int` helper to the client.
- [ ] Ensure thread-safety for the version map using a mutex.

## NOT TODO
- [ ] Do not implement incremental diff logic for `didChange`; always use full document sync.
- [ ] Do not persist this state across server restarts; it is session-based.

## Acceptance Criteria
- [ ] The `lsp.Client` correctly increments versions for sequential messages.
- [ ] No "Version Mismatch" errors are received from `gopls` or `pylsp` during multi-step operations.
- [ ] `didOpen` reliably establishes a v1 baseline.
