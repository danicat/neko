# Initialization & State Management

## Overview
The Neko MCP Server initializes in a restricted "Lobby" state. A strict state machine ensures tools that depend on language analysis (like `build` or `rename_symbol`) cannot be called until a project is successfully opened and its language environment is verified.

## Implementation Steps

1. **Configuration & Server Instantiation**:
   - Parses CLI arguments and config files using `internal/core/config`.
   - Instantiates `mcp.NewServer` from the Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`).
   - Configures the transport protocol. Neko supports both standard I/O (Stdio via `Server.Run`) and HTTP (via `Server.ServeHTTP`).

2. **State Protection (Mutex)**:
   - The `Server` struct (`internal/server/server.go`) uses a `sync.Mutex` (`mu`) to protect shared state fields such as `projectOpen`, `projectRoot`, `ragEngine`, and `activeBackends`.
   - This ensures thread safety when handling concurrent MCP tool requests or background indexing tasks.

3. **Project State Transitions**:
   - **Lobby Phase**: By default, `projectOpen` is `false`. The server only registers core project management tools (`open_project`, `create_project`).
   - **Transition (`establishProject`)**: When `open_project` is called, the handler invokes `establishProject`. This method:
     - Sets `projectOpen = true`.
     - Records the `projectRoot`.
     - Calls `ResolveBackend` to iterate through the registered language backends and determine which are active in the target directory (e.g., detecting `go.mod` for Go).
     - Initializes the RAG engine for the root.
   - **Opened Phase**: Once established, the server dynamically registers the full suite of language-specific tools and kicks off background tasks like starting the LSP (`startLSP`) and indexing the codebase for semantic search (`crawlProject`).
