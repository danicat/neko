# Interface Definition

## Overview
To support a new programming language in Neko, a developer must implement the `LanguageBackend` interface defined in `internal/backend/backend.go`. This interface acts as a universal adapter for build tools, linters, and LSP configuration.

## Core Contract

1. **Identity & Detection**:
   - `Name() string`: Returns the unique identifier (e.g., "go", "python").
   - `LanguageID() string`: The standard LSP string (e.g., "go", "python").
   - `ProjectMarkers() []string`: Files that indicate the presence of this language (e.g., `go.mod`, `requirements.txt`). Used during project discovery.

2. **LSP Configuration**:
   - `LSPCommand() (command string, args []string, ok bool)`: Specifies the binary to execute (e.g., `gopls`, `pyright-langserver`).
   - `InitializationOptions() map[string]any`: Custom config to send during the LSP handshake.

3. **Core Operations**:
   - `BuildPipeline(ctx, dir, opts) (*BuildReport, error)`: The primary Quality Gate. Expected to compile, lint, and/or test the code.
   - `Format(ctx, filename) error`: Defines how the language auto-formats its files.
   - `Modernize(ctx, dir, fix) (string, error)`: Applies idiomatic updates or auto-fixes (e.g., `go fmt`, `ruff check --fix`).

4. **Advanced Tools**:
   - `MutationTest(ctx, dir) (string, error)`: Executes mutation testing if the language ecosystem supports it.
   - `BuildTestDB` and `QueryTestDB`: Hooks for the SQL-based test querying system.
