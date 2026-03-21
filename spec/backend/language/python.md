# Python Implementation

## Overview
The Python language backend (`internal/backend/python`) acts as an adapter for the Python ecosystem, heavily utilizing `ruff` for speed and `pyright` for intelligence.

## Implementation Details

1. **Detection**:
   - Checks for common markers like `requirements.txt`, `pyproject.toml`, or `setup.py`.

2. **LSP Binding**:
   - `LSPCommand()` attempts to spawn `pyright-langserver` via `npx` or standard binary paths.

3. **BuildPipeline**:
   - Since Python is interpreted, the "build" step is functionally a static analysis gate.
   - Executes `ruff check` as the primary, incredibly fast linter to catch syntax errors and stylistic violations.
   - Executes `pytest` for unit testing.

4. **Modernization & Formatting**:
   - `Modernize()` and `Format()` execute `ruff format` to auto-fix and style code deterministically.
