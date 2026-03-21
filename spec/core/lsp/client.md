# Client Implementation

## Overview
The LSP Client (`internal/lsp/client.go`) is responsible for the direct, low-level communication with a running language server process. It acts as the bridge between Neko's MCP tool handlers and the LSP JSON-RPC stream.

## Implementation Steps

1. **JSON-RPC Transport**:
   - The client uses a standard JSON-RPC 2.0 implementation over standard I/O (stdin/stdout) connected to the spawned language server process (e.g., executing `gopls`).

2. **Initialization Sequence**:
   - Upon connection, the client sends an `initialize` request. This payload includes the `workspaceFolders` (the project root) and defines the `ClientCapabilities` (features Neko supports, like document formatting or hover).
   - Once initialized, it sends an `initialized` notification to signal readiness.

3. **Virtual File System (VFS) Syncing**:
   - The client maintains an internal mapping of open file buffers.
   - It implements `textDocument/didOpen`, `textDocument/didChange`, and `textDocument/didClose` notifications.
   - When tools like `edit_file` modify code, these notifications ensure the language server analyzes the *in-memory* state before changes are committed to disk.

4. **Feature Request Wrappers**:
   - The client provides strongly typed Go methods to encapsulate complex LSP JSON-RPC requests, abstracting the raw JSON structures from the rest of Neko:
     - `Hover()` -> `textDocument/hover`
     - `Definition()` -> `textDocument/definition`
     - `References()` -> `textDocument/references`
     - `Formatting()` -> `textDocument/formatting`
     - `Rename()` -> `textDocument/rename`
