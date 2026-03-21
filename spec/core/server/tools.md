# Tool Registration & Handler Mapping

## Overview
Neko uses a dynamic tool registration system. Tools are mapped from their schema definitions to their execution handlers, and registration is gated by the current project state.

## Implementation Steps

1. **Registry Integration**:
   - Tool schemas (Name, Description, Input Schema) are defined centrally in `internal/toolnames/registry.go`. This provides a single source of truth for tool metadata.

2. **Conditional Exposing (`registerHandlersLocked`)**:
   - The method `registerHandlersLocked` handles the conditional registration logic.
   - Tools like `list_files`, `open_project`, and `create_project` are registered unconditionally.
   - If `ProjectOpen()` returns `true`, the method proceeds to register the broader toolset (`edit_file`, `build`, `search`, `rename_symbol`, etc.).

3. **Handler Mapping**:
   - The server uses `mcp.AddTool` to bind execution handlers to the MCP `CallTool` interface.
   - For example, when an MCP client requests `create_project`, the request is routed to `createProjectHandler`.
   - These handlers are responsible for unmarshaling the JSON-RPC arguments into typed Go structs, invoking the underlying business logic, and formatting the result or error back into an `mcp.CallToolResult`.
