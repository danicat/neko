# Initialization & State Management

## Overview
The Neko MCP Server initializes in a restricted "Lobby" state. A strict state machine ensures tools that depend on language analysis (like `build` or `rename_symbol`) cannot be called until a project is successfully opened and its language environment is verified.

## Implementation Steps

1. **Configuration & Server Instantiation**:
   - Parses CLI arguments and config files using `internal/core/config`.
   - Instantiates `mcp.NewServer` from the Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`).
   - Configures the transport protocol. Neko supports both standard I/O (Stdio via `Server.Run`) and HTTP (via `Server.ServeHTTP`).

2. **State Protection (Mutex)**:
   - The `Server` struct (`internal/server/server.go`) uses a `sync.Mutex` (`mu`) to protect shared state fields such as `projectOpen`, `projectRoot`, `ragEngine`, `activeBackends`, `seenDocs`, and `seenTypeInfo`.
   - This ensures thread safety when handling concurrent MCP tool requests or background indexing tasks.

3. **Project State Transitions**:
   - **Lobby Phase**: By default, `projectOpen` is `false`. The server only registers core project management tools (`open_project`, `create_project`).
   - **Transition (`establishProject`)**: When `open_project` is called, the handler invokes `registry.DetectBackends(absRoot)` to determine which language backends are active in the target directory (e.g., detecting `go.mod` for Go). The detected backends are passed to `establishProject`, which:
     - Sets `projectOpen = true` and records the `projectRoot`.
     - Stores the active backends in `activeBackends`.
     - Cancels any previous background crawl (if re-opening).
     - Initializes the RAG engine for semantic search.
     - Launches a background `crawlProject` goroutine with a cancellable context.
     - Eagerly initializes LSP clients for all detected backends via `startLSP`.
     - Calls `registerHandlersLocked` to dynamically register the full tool suite.
   - **Opened Phase**: The server exposes the full suite of language-specific tools (gated by backend capabilities) and kicks off background indexing.

4. **Crawl Lifecycle**:
   - The `Server` struct stores the app-level context (`appCtx`), set once during `Run()` or `ServeHTTP()`. This context is tied to OS signal handling (SIGINT/SIGTERM) for clean shutdown.
   - Each call to `establishProject` cancels any in-progress crawl via `crawlCancel` before starting a new one.
   - The crawl context is derived from `s.appCtx` (not `context.Background()`), so app shutdown cascades to cancel the crawl goroutine automatically.
   - `crawlProject` checks `ctx.Err()` at each file iteration, enabling clean cancellation from either project re-open or app shutdown.
   - On `close_project`, the crawl is cancelled, the RAG engine is set to nil, and `seenTypeInfo` is reset.

5. **Public Methods**:
   - `ProjectRoot() string`: Returns the current project root (mutex-protected).
   - `ProjectOpen() bool`: Returns whether a project is currently open (mutex-protected).
   - `RAGEnabled() bool`: Returns whether the RAG engine is initialized for the current project.
   - `IngestFile(ctx, path, content, symbols, imports) error`: Ingests a file into the RAG engine. Returns `ErrRAGNotInitialized` if no engine is active.
   - `RAGSearch(ctx, query, limit) ([]rag.SearchResult, error)`: Performs a semantic search. Returns `ErrRAGNotInitialized` if no engine is active.
   - `ResolveBackend(language string)`: Returns the appropriate backend, auto-selecting when only one is active.

   The RAG engine is **not** exposed directly. Instead, purpose-specific methods (`IngestFile`, `RAGSearch`, `RAGEnabled`) encapsulate all RAG operations and return the sentinel error `ErrRAGNotInitialized` when the engine is nil.
