# Core Tool Operations: Step-by-Step Logic

This document describes the high-level logic (pseudo-code) for all 14 primary MCP tools provided by `neko`. These tools are language-aware and use the `LanguageBackend` for advanced features.

---

## 1. File System Tools

### `list_files`
Recursively lists files in a directory, preferring git if available.

1.  **Validate Path**: Ensure the root path is within workspace roots.
2.  **Try Git**:
    -   Check if the directory is a git repository (`git rev-parse --show-toplevel`).
    -   If yes, run `git ls-files --cached --others --exclude-standard`.
    -   Filter results based on `depth` and format the output (adding `/` for directories).
3.  **Fallback Walk**:
    -   If not git, use `filepath.WalkDir`.
    -   Skip directories based on `skipDirs` (configured in backend) and common ignore patterns (`.git`, `node_modules`, etc.).
    -   Respect `maxDepth`.
4.  **Limit Results**: Truncate output if it exceeds 1000 files.
5.  **Return**: Hierarchical list of files and directories.

### `file_create`
Creates a new file and ensures it is properly formatted and valid.

1.  **Prepare Directory**: Create all parent directories if they don't exist (`mkdir -p`).
2.  **Initial Write**: Write the provided `content` to the specified `filename`.
3.  **Detect Backend**: Identify the `LanguageBackend`.
4.  **Post-Process (If Backend Available)**:
    -   Call `backend.Format(filename)` to apply language-specific styling.
    -   Call `backend.Validate(filename)` to check for immediate syntax errors.
    -   If formatting changed the file, re-read the final content for the success report.
5.  **Return**: Success report confirming file size, auto-formatting, and syntax verification.

---

## 2. Language & Toolchain Tools

### `smart_read`
Reads a file with optional structural analysis (Outline) and line ranges.

1.  **Validate Path**: Ensure path is within allowed roots.
2.  **Detect Backend**: Identify the appropriate `LanguageBackend`.
3.  **Handle Outline Mode**:
    -   If `outline=true` and no line range is specified:
        -   Call `backend.Outline(filename)` (signatures, types).
        -   Call `backend.ParseImports(filename)` (third-party dependencies).
4.  **Handle Content Read**:
    -   Read raw file content.
    -   If a line range is specified, extract snippet using line-to-byte offsets.
    -   Format output with line numbers.
5.  **Import Analysis (Full Read Only)**:
    -   If full read and backend exists:
        -   Call `backend.ParseImports(filename)`.
        -   Call `backend.ImportDocs(imports)` for package summaries.
6.  **Return**: Formatted content or outline.

### `read_docs`
Retrieves documentation for a package or symbol.

1.  **Validate Format**: Ensure format is 'markdown' (default) or 'json'.
2.  **Detect Backend**: Find backend for the target directory.
3.  **Fetch Documentation**:
    -   If `format=json` and backend supports it (e.g., Go): Call `backend.FetchDocsJSON`.
    -   Otherwise: Call `backend.FetchDocs(importPath, symbol)`.
4.  **Return**: Rendered documentation string.

### `add_dependency`
Installs new packages/modules into the project.

1.  **Detect Backend**: Find backend for the project directory.
2.  **Execute Install**: Call `backend.AddDependency(dir, packages)`.
3.  **Return**: Output from the package manager (e.g., `go get` or `pip install`).

### `project_init`
Bootstraps a new project skeleton.

1.  **Detect/Select Backend**:
    -   Use explicit `language` parameter if provided.
    -   Otherwise, try to detect from existing files in path.
    -   Fallback to `go` or `python`.
2.  **Execute Init**: Call `backend.InitProject(opts)`.
3.  **Return**: Success message.

### `modernize_code`
Upgrades legacy patterns to modern standards.

1.  **Detect Backend**: Find backend for the directory.
2.  **Execute Modernize**: Call `backend.Modernize(dir, fix_bool)`.
3.  **Return**: Report of suggested or applied changes.

### `mutation_test`
Analyzes test suite quality via mutation testing.

1.  **Detect Backend**: Find backend for the directory.
2.  **Execute Mutations**: Call `backend.MutationTest(dir)`.
3.  **Return**: Mutation score and report of surviving mutants.

### `test_query`
Queries test results and coverage data using SQL.

1.  **Detect Backend**: Find backend for the directory.
2.  **Manage Database**:
    -   Check if `testquery.db` exists.
    -   If `rebuild=true` or DB missing: Call `backend.BuildTestDB(dir, pkg)`.
3.  **Execute Query**: Call `backend.QueryTestDB(dir, query)`.
4.  **Return**: Tabular results from the SQL execution.

---

## 3. Development Tools

### `smart_edit`
Performs robust, safe edits using fuzzy matching and syntax verification.

1.  **Read Original**: Read current file content from disk.
2.  **Locate Target Block**:
    -   If `append=true`: Target end of file.
    -   Otherwise: Fuzzy search for `old_content` using Levenshtein distance.
    -   If score < `threshold`, fail with "Best Match" snippet.
3.  **Apply Edit**: Replace matched block with `newContent`.
4.  **Detect Backend**: Identify `LanguageBackend`.
5.  **Safety Gate (If Backend Available)**:
    -   **Validate**: Write new content and call `backend.Validate(filename)`.
    -   **Rollback**: If syntax broken, restore original content and fail.
    -   **Format**: Call `backend.Format(filename)`.
    -   **Re-read**: Read formatted content back for final consistency.
6.  **Final Write**: Write the final valid content to disk.
7.  **Return**: Success/Failure status.

### `smart_build`
Runs the language-specific build, test, and lint pipeline.

1.  **Detect Backend**: Find backend for the directory.
2.  **Execute Pipeline**: Call `backend.BuildPipeline(dir, opts)`.
    -   Backend runs: Tidy -> Format -> Build -> Test -> Lint.
3.  **Return**: Combined output and error status.

---

## 4. Code Intelligence (LSP) Tools

### `symbol_info` / `find_definition` / `find_references`
Delegates intelligence tasks to a Language Server.

1.  **Detect Backend**: Find backend for the file.
2.  **Check LSP Support**: Call `backend.LSPCommand()`.
3.  **Get LSP Client**:
    -   `lsp.DefaultManager` lazily starts/retrieves a server for the project root.
4.  **Execute Request**:
    -   `symbol_info`: Call `client.Hover(file, line, col)`.
    -   `find_definition`: Call `client.Definition(file, line, col)`.
    -   `find_references`: Call `client.References(file, line, col, includeDecl)`.
5.  **Return**: Formatted results (documentation string or list of locations).
