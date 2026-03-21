# Execution & Validation

## Overview
The true power of `edit_file` lies not just in changing text, but in its synchronous connection to the Language Server Protocol. It acts as an immediate Quality Gate.

## Implementation Steps

1. **In-Memory Modification (`performEdit`)**:
   - Once a unique match is found, Neko splices `new_content` into the file's buffer in memory. It does *not* write to disk immediately.

2. **LSP Synchronization (`didChange`)**:
   - The updated in-memory buffer is sent to the running LSP client via a `textDocument/didChange` notification.
   - This forces the language server (e.g., `gopls`) to recompile its internal AST and type-check the modified code.

3. **Synchronous Diagnostics Capture**:
   - Neko blocks and waits for a brief window to receive `textDocument/publishDiagnostics` events from the LSP.
   - This returns an array of syntax errors, type mismatches, or lint warnings that were introduced by the edit.

4. **Commit & Return**:
   - The file is written to disk.
   - Neko formats the raw diagnostics into a human-readable markdown report and returns it in the MCP tool result.
   - If the edit caused an error (e.g., "undefined variable"), the LLM sees it *immediately* in the same turn, allowing it to fix the mistake without needing to run a separate build command. This is known as Neko's "Full Disclosure" principle.
