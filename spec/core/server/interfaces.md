# Server Interfaces & Conventions

## Overview
Neko uses a set of consistent patterns across all tool packages to decouple tools from the server implementation. This document describes the key conventions.

## Per-Tool Server Interface

Each tool package defines its own minimal `Server` interface containing **only the methods it needs**. For example:

```go
// In internal/tools/lang/search/search.go
type Server interface {
    RAGSearch(ctx context.Context, query string, limit int) ([]rag.SearchResult, error)
    RAGEnabled() bool
    ProjectRoot() string
}
```

```go
// In internal/tools/file/create/create.go
type Server interface {
    ForFile(ctx context.Context, path string) backend.LanguageBackend
    IngestFile(ctx context.Context, path string, content string, symbols []lsp.DocumentSymbol, imports []string) error
    RAGEnabled() bool
    ProjectRoot() string
}
```

This follows the Go principle of small, composable interfaces. The main `server.Server` struct satisfies all of these interfaces, but each tool only sees the surface it requires. Notably, no tool has direct access to the RAG engine — they interact through purpose-specific methods (`RAGSearch`, `IngestFile`, `RAGEnabled`) that handle nil-engine checks internally via the `ErrRAGNotInitialized` sentinel error.

## ProjectRoot Pattern

All tools that need to resolve file paths use the `ProjectRoot() string` method:

```go
workspaceRoot := s.ProjectRoot()
if workspaceRoot == "" {
    workspaceRoot, _ = filepath.Abs(".")
}
```

This two-step fallback ensures tools always have a valid workspace root:
1. Ask the server for the project root set during `open_project`.
2. Fall back to the process working directory if no project is open.

The project root is stored in `Server.projectRoot` behind a mutex and set by `establishProject`.

## Tool Definition Registry (`toolnames.Registry`)

All tool metadata (name, title, description, instruction text) is centralized in `internal/toolnames/registry.go` as a `map[string]ToolDef`. Tools retrieve their definitions during registration:

```go
def := toolnames.Registry["edit_file"]
mcp.AddTool(mcpServer, &mcp.Tool{
    Name:        def.Name,
    Title:       def.Title,
    Description: def.Description,
}, handler)
```

This ensures:
- Tool names and descriptions are defined in one place.
- The `instructions` package can iterate over the registry to build the system prompt.
- No tool has hardcoded metadata scattered across its handler.

## Root Validation Security Boundary

All tools that access the filesystem call `roots.Global.Validate(absPath)` before proceeding:

```go
if err := roots.Global.Validate(absPath); err != nil {
    return errorResult(err.Error()), nil, nil
}
```

The `roots` system is **client-driven**: the MCP client declares allowed roots via `roots/list`, and the server synchronizes via `roots.Global.Sync()`. The server never adds roots itself — it only validates paths against what the client has authorized.

## Capability-Based Tool Registration

Tools are not unconditionally registered. The server checks backend capabilities:

```go
caps := make(map[backend.Capability]bool)
for _, be := range s.activeBackends {
    for _, c := range be.Capabilities() {
        caps[c] = true
    }
}
if caps[backend.CapLSP] {
    describe.Register(s.mcpServer, s)
    definition.Register(s.mcpServer, s)
    references.Register(s.mcpServer, s)
    rename.Register(s.mcpServer, s)
}
```

Additionally, some tools perform their own conditional registration. For example, `codereview.Register` attempts to initialize a GenAI client and silently returns without registering if no credentials are available.
