# File List Logic

## Overview
The `list_files` tool (`internal/tools/file/list/list.go`) provides a hierarchical view of the project's source tree, explicitly filtering out noise that LLMs do not need to see.

## Implementation Details

1. **Recursion Control**:
   - Accepts a `depth` parameter (defaulting to 5) to prevent overwhelming the context window with massive directory trees.

2. **Backend-Aware Filtering**:
   - Skips standard version control folders (`.git`).
   - Skips Neko's internal directories (`.neko`).
   - Delegates to the active `LanguageBackend` to retrieve `SkipDirs()` (e.g., `vendor`, `node_modules`, `__pycache__`, `.pytest_cache`). This ensures the list only contains relevant source code.

3. **Output Formatting**:
   - Uses a tree-like ASCII representation (similar to the standard Unix `tree` command) which is highly readable for language models.
