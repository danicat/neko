# Server (MCP Integration)

The `internal/server` package orchestrates the Model Context Protocol (MCP) server lifecycle and maintains global application state. It acts as the central router and state machine for the entire Neko application.

## Sub-Components
- [Initialization & State Management](state.md): Details the startup sequence and the Lobby vs. Open project phase logic.
- [Tool Registration & Handler Mapping](tools.md): Explains how MCP tool requests are routed to specific internal Go functions.
