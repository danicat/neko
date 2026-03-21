# Manager Lifecycle

## Overview
The `Manager` struct (`internal/lsp/manager.go`) acts as a registry and orchestrator for LSP clients. Because a workspace might be polyglot (e.g., containing both Go and Python), the manager tracks and manages the lifecycle of multiple language server instances simultaneously.

## Implementation Steps

1. **Singleton Management**:
   - The `Manager` holds a map of running `Client` instances, typically keyed by the language identifier or the language server command name.
   - Access to this map is protected by mutexes to support concurrent tool calls across different files.

2. **Startup Trigger (`ClientFor`)**:
   - When a language tool requires LSP capabilities, it calls `ClientFor(ctx, lang, workspaceRoot, command, args, langID, options)`.
   - If a client for that language/workspace combination is already running and healthy, it is reused.
   - If not, the Manager spawns a new process (e.g., `exec.Command("gopls")`), attaches the JSON-RPC streams, instantiates a new `Client`, and initiates the handshake sequence.

3. **Graceful Shutdown (`CloseAll`)**:
   - When a project is closed via the `close_project` tool, or when the main MCP server process receives a termination signal, the Manager is responsible for cleanup.
   - `CloseAll` iterates through all tracked clients, sending the standard LSP `shutdown` request followed by the `exit` notification, ensuring the subprocesses terminate cleanly without leaving zombies.
