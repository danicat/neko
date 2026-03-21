# Multi-Edit Transaction

## Overview
Often, a structural change (like altering a function's signature) requires updating the definition in one file and the callsites in several other files. The `multi_edit` tool batches these edits so that cross-file dependencies are resolved before returning a final health report to the LLM.

## Implementation Steps

1. **Registration**:
   - `MultiRegister` exposes `multi_edit` in `internal/tools/file/edit/edit.go`.

2. **Parsing Array**:
   - The handler accepts a JSON array of edit objects (each containing a `file`, `old_content`, and `new_content`).

3. **Iterative Application**:
   - Neko iterates through the array, calling the underlying `performEdit` logic for each file sequentially.
   - For each file, this includes fuzzy matching, LSP code actions (OrganizeImports, Formatting), disk writing, and RAG re-indexing.

4. **Global Validation**:
   - Instead of returning diagnostic errors file-by-file (which would be noisy and potentially incomplete if file A depends on file B), `multi_edit` waits until all edits are applied.
   - It then identifies all language backends affected by the changes.
   - For each affected backend, it triggers a global `PullDiagnostics` across the entire workspace.
   
5. **Unified Diagnostic Report**:
   - The resulting report aggregates errors across the entire project. This proves that the interdependent changes were semantically valid and cross-file contracts (like updated function signatures) were honored successfully.