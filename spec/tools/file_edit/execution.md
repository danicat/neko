# Execution & Validation

## Overview
The true power of `edit_file` lies not just in changing text, but in its synchronous connection to the Language Server Protocol and the RAG engine. It acts as an immediate Quality Gate.

## Implementation Steps

1. **In-Memory Modification (`performEdit`)**:
   - Once a unique match is found via fuzzy matching, Neko splices `new_content` into the file's buffer in memory.

2. **LSP Code Actions (Auto-Fixing)**:
   - **Organize Imports**: Neko sends an `OrganizeImports` request to the LSP. If the edit introduced a new dependency or orphaned an old one, the LSP automatically adds or removes the import declaration.
   - **Formatting**: Neko sends a `textDocument/formatting` request. The LSP applies language-specific formatting rules (e.g., `gofmt`) to ensure the spliced code matches project conventions.

3. **Commit to Disk**:
   - The formatted, import-adjusted content is written to disk.

4. **Synchronous RAG Re-Indexing**:
   - If `s.RAGEnabled()` is true, Neko immediately ingests the updated file via `s.IngestFile()`.
   - It enriches this ingestion with document symbols (from the LSP) and import paths (from the backend), ensuring that subsequent semantic searches immediately reflect the new code structure.
   - Ingestion errors are surfaced in the tool response (not silently discarded).

5. **LSP Synchronization (`didSave`)**:
   - A `textDocument/didSave` notification is sent to the LSP to trigger its internal re-indexing and validation.

6. **Diagnostics Capture**:
   - If an LSP is active, Neko blocks briefly to capture synchronous diagnostics (`textDocument/publishDiagnostics`).
   - If no LSP is active, it falls back to the backend's manual `Validate()` routine.
   - The tool returns a human-readable Markdown report of these diagnostics. If the edit caused a syntax error, the LLM sees it immediately in the same turn (the "Full Disclosure" principle).