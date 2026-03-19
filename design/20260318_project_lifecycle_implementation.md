# Design: Project Lifecycle Implementation

## Overview
This document summarizes the implementation of the two-phase project lifecycle in Neko: **Lobby Phase** and **Project Phase**. This structure ensures that language-aware tools are only available when a project context is established, optimizing tool discovery and resource management.

## 1. Two-Phase Architecture

### Lobby Phase
- **Entry Point**: The server starts in this phase.
- **Available Tools**: `open_project`, `create_project`.
- **Restricted Tools**: All engineering and file tools are hidden to prevent execution without context.

### Project Phase
- **Trigger**: Established via `open_project` or `create_project`.
- **Available Tools**:
    - **Lifecycle**: `close_project` (returns to Lobby).
    - **Agnostic**: `list_files`, `read_file`, `edit_file`, `create_file`, `review_code`.
    - **Capability-Based**: `build`, `read_docs`, `add_dependencies`, `test_mutations`, `query_tests`, `describe`, `find_definition`, `find_references`.
- **Dynamic Activation**: Backends are activated "on-touch" when a relevant file is read or edited.

## 2. Technical Implementation

### Server State Management
- **`Server` struct**: Added `projectOpen` (bool), `projectRoot` (string), and `activeBackends` (map).
- **Tool Registration**: `RegisterHandlers` and `registerHandlersLocked` perform strict tool management using `mcpServer.RemoveTools` and conditional `Register` calls based on phase and backend capabilities.

### Language Backend Capabilities
- **`Capability` type**: Defined in `internal/backend/backend.go` (e.g., `CapToolchain`, `CapLSP`).
- **Backend interface**: `Capabilities()` method added to allow backends to declare supported features.

### Tool Standardization
- **Naming**: Standardized on snake_case (e.g., `test_mutations`, `query_tests`, `review_code`).
- **Arguments**: Standardized parameters like `dir`, `file`, `line`, `col`, and `language`.
- **Registry**: `internal/toolnames/registry.go` serves as the single source of truth for tool definitions and instructions.

## 3. Verification
- **Strict Phase Transitions**: Verified that lobby tools are removed in Project Phase and vice-versa.
- **Dynamic On-Touch**: Verified that reading a file of a new language correctly surfaces its specific tools.
