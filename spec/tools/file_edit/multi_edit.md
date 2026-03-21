# Multi-Edit Transaction

## Overview
Often, a structural change (like altering a function's signature) requires updating the definition in one file and the callsites in several other files. Doing this sequentially with `edit_file` would result in transient LSP errors in the intermediate turns. The `multi_edit` tool solves this by batching edits atomically.

## Implementation Steps

1. **Registration**:
   - `MultiRegister` exposes `multi_edit` in `internal/tools/file/edit/edit.go`.

2. **Parsing Array**:
   - The handler `multiEditHandler` accepts a JSON array of edit objects (each containing a `file`, `old_content`, and `new_content`).

3. **Staged Modification**:
   - It iterates through the array, using the same fuzzy matching logic to apply changes to all files *in memory*.

4. **Atomic Synchronization**:
   - `didChange` notifications are sent for all modified files simultaneously.
   - Neko waits for the language server to process the entire transaction and resolve cross-file dependencies.

5. **Unified Diagnostic Report**:
   - All files are written to disk.
   - The resulting diagnostic report aggregates errors across the entire project, proving that the interdependent changes were semantically valid.
