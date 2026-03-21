# High-Level Architecture

## Executive Summary
Neko is a language-aware, intelligent development assistant powered by the Model Context Protocol (MCP). It operates by running an MCP server that integrates deeply with Language Server Protocols (LSP) (e.g., `gopls` or `pyright`) and provides robust project lifecycle management.

Neko enforces a strict two-phase state machine:
1. **Lobby**: The default state. Only project initialization (`create_project`) and opening (`open_project`) are allowed.
2. **Project Opened**: Once a project context is established (`establishProject`), Neko enables a suite of navigation, editing, and language tools tailored to the active language backend (Go, Python, etc.).

## Core Principles
1. **Semantic Integrity**: Every file mutation triggers synchronous LSP validation, returning immediate diagnostic feedback (the "Quality Gate"). This ensures changes don't break the build before they are finalized.
2. **Deterministic Edits**: `edit_file` uses fuzzy matching for robustness to minor formatting drifts, while `rename_symbol` handles project-wide renames atomically via the LSP, eliminating regex-based find-and-replace errors.
3. **Intent-Based Discovery**: Neko uses RAG (Retrieval-Augmented Generation) backed by local vector databases for semantic search across the codebase, allowing searches by meaning rather than strict symbol names.

## System Topology
- **MCP Server**: The interface layer communicating with LLM clients (`internal/server/server.go`). It routes JSON-RPC messages to appropriate tool handlers.
- **LSP Manager**: Manages concurrent language server lifecycles per project (`internal/lsp/manager.go`). It starts, stops, and communicates with background LSP processes over standard I/O.
- **Backend Registry**: Pluggable language backends handling specific build, test, and lint operations (`internal/backend/backend.go`). This allows Neko to be polyglot by abstracting language-specific toolchains.
