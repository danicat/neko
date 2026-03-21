# Tool Registration & Handler Mapping

## Overview
Neko uses a dynamic tool registration system. Tools are mapped from their schema definitions to their execution handlers, and registration is gated by the current project state and backend capabilities.

## Implementation Steps

1. **Registry Integration**:
   - Tool schemas (Name, Title, Description, Instruction) are defined centrally in `internal/toolnames/registry.go`. This provides a single source of truth for tool metadata.
   - See [Server Interfaces](interfaces.md) for details on the `toolnames.Registry` pattern.

2. **Conditional Exposing (`registerHandlersLocked`)**:
   - The method `registerHandlersLocked` handles the conditional registration logic.
   - **Lobby Phase** (`projectOpen == false`): Only `open_project` and `create_project` are registered. All project-phase tools are explicitly removed.
   - **Project Phase** (`projectOpen == true`): The lobby tools are removed and the full toolset is registered in tiers:
     - **Agnostic tools** (always registered when project is open): `read_file`, `edit_file`, `multi_edit`, `list_files`, `create_file`.
     - **RAG-dependent**: `semantic_search` is only registered if `ragEngine != nil`.
     - **Auth-dependent**: `review_code` performs its own conditional registration (silently skips if GenAI credentials are absent).
     - **Capability-gated**: Tools are registered based on the union of capabilities across all active backends:
       - `CapToolchain` → `build`
       - `CapDocumentation` → `read_docs`
       - `CapDependencies` → `add_dependencies`
       - `CapMutationTest` → `test_mutations`
       - `CapTestQuery` → `query_tests`
       - `CapLSP` → `describe`, `find_definition`, `find_references`, `rename_symbol`

3. **Handler Mapping**:
   - The server uses `mcp.AddTool` to bind execution handlers to the MCP `CallTool` interface.
   - For example, when an MCP client requests `create_project`, the request is routed to `createProjectHandler`.
   - These handlers are responsible for unmarshaling the JSON-RPC arguments into typed Go structs, invoking the underlying business logic, and formatting the result or error back into an `mcp.CallToolResult`.

4. **Phase Transitions**:
   - On `open_project` or `close_project`, `registerHandlersLocked` is called to swap the registered tool set.
   - `RemoveTools` is called first to clear the previous phase's tools, ensuring no stale tools remain visible to the LLM.
