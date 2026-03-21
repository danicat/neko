# File Create Logic

## Overview
The `create_file` tool (`internal/tools/file/create/create.go`) initializes a new source file while ensuring immediate LSP awareness and syntactic correctness.

## Implementation Details

1. **Directory Generation**:
   - Ensures all parent directories for the target file path are created (`os.MkdirAll`).

2. **LSP Lifecycle**:
   - If a language backend with LSP support is available for the file type, the full LSP lifecycle is executed:
     1. `didOpen` — Notifies the LSP of the new file.
     2. `OrganizeImports` — Auto-adds or removes imports via LSP code actions.
     3. `Format` — Applies language-specific formatting (e.g., gofmt for Go).
     4. File is written to disk with the formatted content.
     5. `didChangeWatchedFiles` — Triggers LSP file indexing for the new file.
     6. `didSave` — Notifies the LSP that the file has been saved.
     7. `WaitForDiagnostics` — Captures synchronous diagnostics (catching errors like missing package declarations or invalid imports immediately).
     8. `didClose` — Closes the LSP document session.
   - If LSP is unavailable but a backend exists, falls back to `be.Validate()` for basic syntax checking.

3. **RAG Re-Indexing**:
   - After writing the file, if the RAG engine is active, the new file is synchronously ingested via `engine.IngestFile`.
   - Document symbols (from LSP) and imports (from backend) are passed alongside the content for richer semantic indexing.

4. **Diagnostics Reporting**:
   - Returns a Markdown response with a success header and a formatted diagnostics report across the entire workspace.
   - If LSP is unavailable, notes this in the response and includes any backend validation warnings.
