# File List Logic

## Overview
The `list_files` tool (`internal/tools/file/list/list.go`) provides a hierarchical view of the project's source tree, explicitly filtering out noise that LLMs do not need to see.

## Implementation Details

1. **Recursion Control**:
   - Accepts a `depth` parameter (defaulting to 5) to prevent overwhelming the context window with massive directory trees.

2. **Backend-Aware Filtering**:
   - Skips a hardcoded list of directories (`.git`, `.idea`, `.vscode`, `node_modules`, `__pycache__`, `.venv`, `venv`, `.mypy_cache`, `.pytest_cache`, `.ruff_cache`, `.tox`, `.nox`, `dist`, `build`, `.eggs`).
   - Appends backend-specific skip directories via `Registry().AllSkipDirs()` (e.g., `vendor` for Go).
   - Also supports a git-aware fast path via `git ls-files` when available.

3. **Output Formatting**:
   - Uses a tree-like ASCII representation (similar to the standard Unix `tree` command) which is highly readable for language models.
