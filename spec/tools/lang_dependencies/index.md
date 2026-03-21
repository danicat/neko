# Add Dependencies Logic

## Overview
The `add_dependencies` tool (`internal/tools/lang/get/get.go`) simplifies project package management and learning by combining installation with immediate documentation retrieval.

## Implementation Details

1. **Parameters**:
   - `packages`: A list of package strings to install (e.g., `["github.com/gin-gonic/gin@latest"]`).
   - `dir`: Optional directory context.

2. **Delegation**:
   - Resolves the `LanguageBackend` for the target directory.
   - Calls `be.AddDependency(ctx, dir, packages)`.

3. **Backend Implementations**:
   - **Go**: Executes `go get <package>` followed by `go mod tidy` to clean up the module file.
   - **Python**: Uses `uv add <package>` to install into the project's virtual environment.

4. **Synchronous Documentation (The "Get and Learn" pattern)**:
   - After a successful installation, the tool doesn't just return "success".
   - It iterates through the installed packages and immediately calls `be.FetchDocs(ctx, dir, pkg, "")` for each one.
   - Note: version tags (e.g., `@latest`) are currently passed through to `FetchDocs` without stripping — the backend's doc fetcher is expected to handle this.
   - By returning the API documentation of the newly installed package directly in the tool result, the LLM is instantly equipped with the package's signatures and can begin implementing the integration in the very next turn.