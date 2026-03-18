# Detailed Mechanics: Neko Core Tools

This document provides a deep-dive into the step-by-step internal operations of all 15 tools provided by the `neko` MCP server.

---

## 1. `smart_read`
**Purpose:** Context-aware file reading with structural analysis.

1.  **Path Resolution**: Validates the provided path against registered workspace roots.
2.  **Backend Detection**: Selects a `LanguageBackend` based on the file extension.
3.  **Mode: Outline (`outline=true`)**:
    *   Calls `backend.Outline()` to generate a structural map (AST-truncated signatures).
    *   Calls `backend.ParseImports()` to extract third-party dependencies.
    *   Returns a Markdown report with the outline and a list of external imports.
4.  **Mode: Content Read**:
    *   Reads the file from disk (`os.ReadFile`).
    *   **Line Slicing**: If `start_line` or `end_line` are provided, it calculates byte offsets using `shared.GetLineOffsets` and slices the content.
    *   **Line Numbering**: Prepends `   1 | ` style line numbers to every line for agent reference.
    *   **Context Injection (Full Read only)**:
        *   Calls `backend.ParseImports()` to identify all external dependencies.
        *   Calls `backend.ImportDocs()` which retrieves documentation summaries for those imports.
        *   Appends "Imported Packages" documentation to the output.
5.  **Return**: A Markdown-formatted block containing the code/outline and the additional context.

---

## 2. `smart_edit`
**Purpose:** Precise, safe code modification using fuzzy matching and a quality gate.

1.  **Read Original**: Reads the current file content.
2.  **Locate Target**:
    *   **Append Mode**: Sets target to the end of the file.
    *   **Match Mode**:
        *   **Normalization**: Strips all whitespace from both `old_content` and the file content for comparison.
        *   **Exact Match**: Tries `strings.Index` on normalized content.
        *   **Fuzzy Search**: If exact fails, uses a sliding window with Levenshtein distance to find the "closest" block.
        *   **Threshold Check**: If similarity < `threshold` (default 0.95), it aborts and returns the "Best Match" snippet to the agent for correction.
3.  **Apply Change**: Swaps the identified block with `new_content`.
4.  **Safety Gate**:
    *   **Temporary Write**: Writes the edited content to disk.
    *   **Syntax Validation**: Calls `backend.Validate()`. If it returns an error:
        *   Restores the original file from memory.
        *   Extracts an "Error Snippet" (surrounding lines of the error) and returns it to the agent.
    *   **Auto-Formatting**: Calls `backend.Format()`. This triggers tools like `goimports`, `ruff`, or `prettier`.
    *   **Sync**: Re-reads the file from disk to ensure the agent's view matches the formatted version.
5.  **Return**: Success confirmation or a detailed error report with suggestions.

---

## 3. `smart_create` (formerly `file_create`)
**Purpose:** Safe file initialization.

1.  **Directory Creation**: Runs `os.MkdirAll` for the parent directory path.
2.  **Write**: Writes the initial `content`.
3.  **Backend Hook**:
    *   Detects `LanguageBackend`.
    *   Runs `backend.Format()` to ensure the file starts with the correct style (e.g., adding imports or fixing indentation).
    *   Runs `backend.Validate()` to catch immediate syntax errors in the provided template.
4.  **Return**: Confirmation of bytes written and verification status.

---

## 4. `list_files`
**Purpose:** Efficient project structure discovery.

1.  **Git Check**: Attempts `git ls-files`. If successful, it uses Git's internal index (highly efficient and respects `.gitignore`).
2.  **Fallback Walk**: If not a Git repo, uses `filepath.WalkDir`.
3.  **Filtering**:
    *   Applies `skipDirs` from the `LanguageBackend` (e.g., Go's `vendor`, Python's `venv`).
    *   Excludes standard system/IDE folders (`.git`, `node_modules`, `.vscode`).
4.  **Depth Control**: Respects the `depth` parameter (default 5) to prevent token overflow in massive trees.
5.  **Limit**: Caps output at 1000 items.

---

## 5. `smart_build`
**Purpose:** Multi-stage quality pipeline.

1.  **Backend Resolution**: Finds the backend based on project markers (e.g., `go.mod`).
2.  **Execution**: Triggers the `BuildPipeline()` method, which typically performs:
    *   **Tidy**: Cleans up dependency manifests.
    *   **Format**: Ensures consistency.
    *   **Compile**: Runs the language compiler.
    *   **Test**: Runs unit tests.
    *   **Lint**: Runs static analysis tools (e.g., `golangci-lint`, `ruff`).
3.  **Return**: Combined output of all stages and a boolean error status.

---

## 6. `read_docs`
**Purpose:** Direct API documentation access.

1.  **Detect Backend**: Identifies language context.
2.  **Fetch**:
    *   **Go (Advanced)**: 
        *   Uses internal `godoc` parser for local/standard library docs.
        *   **Vanity Imports**: Automatically resolves vanity paths to canonical paths.
        *   **Examples**: Injects code examples directly into the Markdown.
        *   **Non-code Packages**: Provides an index of sub-packages if the root package has no source files.
    *   **Python**: Executes `python3 -m pydoc`.
    *   **Plugins**: Executes the configured `fetchDocs` command.
3.  **Format**: Supports `markdown` or `json` (for programmatic analysis).

---

## 7. `add_dependency`
**Purpose:** Unified dependency management.

1.  **Command Execution**: Calls `backend.AddDependency()`.
    *   **Go**: `go get <pkg>`.
    *   **Python**: `pip install <pkg>`.
2.  **Documentation Injection**: For **all languages**, the tool calls `backend.FetchDocs()` for each newly installed package and appends the documentation to the success message. This ensures the agent immediately understands the new API.

---

## 8. `project_init`
**Purpose:** Project bootstrapping.

1.  **Directory Setup**: Validates target path.
2.  **Template Execution**: Calls `backend.InitProject()`.
    *   **Go**: `go mod init <name>` + dependency installation.
    *   **Python**: Creates `pyproject.toml` or `setup.py`.
3.  **Documentation Injection**: Similar to `add_dependency`, it fetches and appends documentation for all initial dependencies specified in the request.
4.  **Verification**: Confirms the project markers are present on disk.

---

## 9. `modernize_code`
**Purpose:** Automated technical debt reduction.

1.  **Analysis**: Calls `backend.Modernize()` with `fix=false` to find issues.
2.  **Application**: If `fix=true`, applies automated refactorings (e.g., `go fix`, `python-modernize`).
3.  **Return**: Diff or report of changes.

---

## 10. `mutation_test`
**Purpose:** Empirical test suite validation.

1.  **Execution**: Calls `backend.MutationTest()`.
2.  **Process**:
    *   Inverts logic (e.g., `==` to `!=`) in source code.
    *   Runs tests.
    *   Checks if tests fail (mutant "killed").
3.  **Report**: Returns a mutation score (percentage of killed mutants).

---

## 11. `test_query` (Go Only)
**Purpose:** SQL-based analysis of test results. *Note: This tool is currently only available for Go projects.*

1.  **Instrumentation**: Runs tests with coverage and output logging enabled.
2.  **Persistence**: Parses results into a SQLite database (`testquery.db`).
3.  **Query**: Executes the user's SQL against tables like `all_tests`, `all_coverage`, and `all_code`.

---

## 12. `symbol_info`
**Purpose:** LSP-based hover information.

1.  **LSP Start**: `lsp.DefaultManager` ensures a language server (e.g., `gopls`, `pyright`) is running.
2.  **Request**: Sends `textDocument/hover` with file/line/col.
3.  **Parse**: Extracts documentation and type signatures from the LSP response.

---

## 13. `find_definition`
**Purpose:** LSP-based navigation.

1.  **Request**: Sends `textDocument/definition`.
2.  **Response**: Receives a `Location` (URI + Range).
3.  **Resolution**: Converts URI back to a local file path.

---

## 14. `find_references`
**Purpose:** LSP-based impact analysis.

1.  **Request**: Sends `textDocument/references`.
2.  **Option**: Can include the original declaration.
3.  **Return**: A list of all call sites and usages across the project.

---

## 15. `code_review`
**Purpose:** AI-powered peer review.

1.  **Credential Check**: Verifies `GOOGLE_API_KEY` or Vertex AI config.
2.  **Prompt Construction**: Injects a "Senior Staff Engineer" system prompt and any user-provided hints.
3.  **API Call**: Sends the code to Gemini.
4.  **Response Processing**:
    *   Parses the model's JSON response (Finding, Severity, Line Number).
    *   Renders a Markdown report with icons (🚨 Error, ⚠️ Warning, 💡 Suggestion).
