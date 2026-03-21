# LSP Manager & Client

The `internal/lsp` package provides generic Language Server Protocol client capabilities. It abstracts the complexities of communicating with diverse language servers (like `gopls` or `pyright`) behind a unified Go interface.

## Sub-Components
- [Client Implementation](client.md): Details the JSON-RPC communication, virtual file system management, and specific LSP request wrappers.
- [Manager Lifecycle](lifecycle.md): Explains how the singleton manager tracks, starts, and stops multiple LSP instances concurrently.
