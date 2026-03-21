# Read Docs Logic

## Overview
The `read_docs` tool (`internal/tools/lang/docs/docs.go`) fetches documentation for standard library packages or external modules, saving the LLM from hallucinating API signatures or needing to browse the web.

## Implementation Details

1. **Parameters**:
   - `import_path`: The package to inspect (e.g., `net/http` or `pathlib`).
   - `symbol` (optional): A specific function, type, or class within that package.
   - `format` (optional): `markdown` (default) or `json`.
   - `language` (optional): Forces a specific backend if auto-detection fails.

2. **Delegation**:
   - The tool resolves the target directory to identify the correct `LanguageBackend`.
   - It delegates to the backend's `FetchDocs(ctx, dir, import_path, symbol)` method.

3. **Backend Implementations**:
   - **Go**: Uses `go doc` under the hood. The `golang.godoc` package executes `go doc -all <pkg> <symbol>`, captures the plain text output, and converts it into readable Markdown for the LLM. It falls back to HTTP vanity URL resolution if local package lookup fails.
   - **Python**: Executes `python -c "import pydoc; pydoc.render_doc('<pkg>.<symbol>')"` to extract docstrings.

4. **Output**:
   - Returns a structured documentation block that includes signatures, types, and comments, keeping the agent within the CLI context.