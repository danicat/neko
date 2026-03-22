// Package toolnames defines the registry of available tools for the neko server.
package toolnames

// ToolDef defines the textual representation of a tool.
type ToolDef struct {
	Name        string
	Title       string
	Description string
	Instruction string
}

// Registry holds all tool definitions, keyed by Name.
var Registry = map[string]ToolDef{
	// --- PROJECT LIFECYCLE ---
	"open_project": {
		Name:        "open_project",
		Title:       "Open Project",
		Description: "Opens an existing project directory, detects active languages, and initializes language servers. This must be called before using most engineering tools.",
		Instruction: "*   **`open_project`**: Open an existing project.\n    *   **Usage:** `open_project(dir=\".\")`\n    *   **Outcome:** Project context is established and relevant tools are enabled.",
	},
	"close_project": {
		Name:        "close_project",
		Title:       "Close Project",
		Description: "Closes the current project, shuts down language servers, and returns to the lobby. Use this when switching between unrelated projects.",
		Instruction: "*   **`close_project`**: Close the active project.\n    *   **Usage:** `close_project()`\n    *   **Outcome:** All project-specific resources are released.",
	},
	"create_project": {
		Name:        "create_project",
		Title:       "Create Project",
		Description: "Bootstraps a new project by creating the directory, initializing the module, and installing essential dependencies. Reduces boilerplate and ensures a standard project structure.",
		Instruction: "*   **`create_project`**: Bootstrap a new project.\n    *   **Usage:** `create_project(dir=\"my-app\", language=\"go\", dependencies=[\"...\"])`\n    *   **Outcome:** A valid project with requested dependencies and a skeleton structure.",
	},

	// --- FILE OPERATIONS ---
	"create_file": {
		Name:        "create_file",
		Title:       "Create File",
		Description: "Initializes a new source file, automatically creating parent directories and applying language-appropriate formatting. Ensures new files are immediately compliant with project style guides.",
		Instruction: "*   **`create_file`**: Initialize a new source file.\n    *   **Usage:** `create_file(file=\"src/utils.go\", content=\"package utils\n...\")`\n    *   **Outcome:** A correctly formatted, directory-synced file is created.",
	},
	"line_edit": {
		Name:        "line_edit",
		Title:       "Line-Based Edit",
		Description: "A surgical editing tool that operates on absolute line ranges rather than fuzzy content matching. Ideal when exact line numbers are known.",
		Instruction: "*   **`line_edit`**: Replace a specific line range with new content.\n    *   **Usage:** `line_edit(file=\"main.go\", start_line=10, end_line=15, new_content=\"...\")`",
	},
	"edit_file": {
		Name:        "edit_file",
		Title:       "Edit File",
		Description: "An intelligent file editor providing robust block matching and safety guarantees. Automatically handles formatting, import optimization, and syntax verification to ensure edits do not break the code.",
		Instruction: "*   **`edit_file`**: The primary tool for safe code modification.\n    *   **Capabilities:** Validates syntax and auto-formats (gofmt/goimports for Go, ruff for Python) *before* committing changes to disk.\n    *   **Robustness:** Uses fuzzy matching to locate target blocks despite minor whitespace or indentation variances.\n    *   **Usage:** `edit_file(file=\"...\", old_content=\"...\", new_content=\"...\")`.\n    *   **Outcome:** A syntactically valid, properly formatted file update.",
	},
	"multi_edit": {
		Name:        "multi_edit",
		Title:       "Multi-File Edit",
		Description: "Applies multiple file edits in a single turn. Ensures project-wide semantic integrity by providing a unified diagnostic report after all changes are applied.",
		Instruction: "*   **`multi_edit`**: Apply multiple file changes in a single Turn.\n    *   **Usage:** `multi_edit(edits=[{\"file\": \"...\", \"old_content\": \"...\", \"new_content\": \"...\"}, ...])`\n    *   **Outcome:** Multiple files are updated, followed by a full project diagnostic list.",
	},
	"multi_read": {
		Name:        "multi_read",
		Title:       "Read Multiple Files",
		Description: "Reads multiple files or line ranges in a single turn to reduce token overhead.",
		Instruction: "*   **`multi_read`**: Inspect multiple files in a single turn.\n    *   **Usage:** `multi_read(reads=[{\"file\": \"main.go\", \"start_line\": 10, \"end_line\": 20}, {\"file\": \"utils.go\", \"outline\": true}])`",
	},
	"read_file": {

		Name:        "read_file",
		Title:       "Read File",
		Description: "A structure-aware file reader that optimizes for context density. Supports returning full content, structural outlines (AST-based signatures), or specific line ranges to minimize token consumption.",
		Instruction: "*   **`read_file`**: Inspect file content and structure.\n    *   **Read All:** `read_file(file=\"pkg/utils.go\")`\n    *   **Outline:** `read_file(file=\"pkg/utils.go\", outline=true)` (Retrieve types and function signatures only).\n    *   **Snippet:** `read_file(file=\"pkg/utils.go\", start_line=10, end_line=50)` (Targeted range reading).\n    *   **Outcome:** Targeted source content or structural map.",
	},
	"list_files": {
		Name:        "list_files",
		Title:       "List Files",
		Description: "Recursively lists files and directories while filtering out build artifacts and version control data. Provides an accurate view of the source code hierarchy.",
		Instruction: "*   **`list_files`**: Explore the project structure.\n    *   **Usage:** `list_files(dir=\".\", depth=2)`\n    *   **Outcome:** A hierarchical list of source files and directories.",
	},
	"semantic_search": {
		Name:        "semantic_search",
		Title:       "Semantic Search",
		Description: "Finds relevant code snippets based on a natural language query using a local vector database. Useful for high-level pattern discovery and understanding intent.",
		Instruction: "*   **`semantic_search`**: Find code by meaning.\n    *   **Usage:** `semantic_search(query=\"how is lsp lifecycle handled?\")`\n    *   **Outcome:** A ranked list of semantically similar code chunks with metadata.",
	},
	// --- DOCS ---
	"read_docs": {
		Name:        "read_docs",
		Title:       "Get Documentation",
		Description: "Retrieves documentation for any package or symbol. Supports Go (godoc) and Python (pydoc). Streamlines development by providing API signatures and usage examples directly within the workflow.",
		Instruction: "*   **`read_docs`**: Access API documentation.\n    *   **Usage:** `read_docs(import_path=\"net/http\", language=\"go\")` or `read_docs(import_path=\"pathlib\", language=\"python\")`\n    *   **Outcome:** API reference and usage guidance.",
	},

	// --- TOOLCHAIN ---
	"build": {
		Name:        "build",
		Title:       "Build Project",
		Description: "The primary build tool. Enforces a quality gate pipeline appropriate to the detected language. Ensures code is production-ready.",
		Instruction: "*   **`build`**: Compile and verify code.\n    *   **Usage:** `build(dir=\".\", language=\"go\", auto_fix=true)`\n    *   **Outcome:** A comprehensive report on build status, test results, and lint issues.",
	},
	"add_dependencies": {
		Name:        "add_dependencies",
		Title:       "Add Dependencies",
		Description: "Manages package installation and manifest updates. Consolidates the workflow by immediately returning the public API documentation for the installed packages.",
		Instruction: "*   **`add_dependencies`**: Install dependencies and fetch documentation.\n    *   **Usage:** `add_dependencies(packages=[\"github.com/gin-gonic/gin@latest\"], language=\"go\")`\n    *   **Outcome:** Dependency added and API documentation returned.",
	},

	// --- TESTING ---
	"test_mutations": {
		Name:        "test_mutations",
		Title:       "Mutation Test",
		Description: "Runs mutation testing. Introduces small code mutations and checks if existing tests catch them, objectively measuring test suite quality.",
		Instruction: "*   **`test_mutations`**: Verify test quality with mutation testing.\n    *   **Usage:** `test_mutations(dir=\".\", language=\"go\")`\n    *   **Outcome:** A report showing which mutations survived (tests missed) and the mutation score.",
	},
	"query_tests": {
		Name:        "query_tests",
		Title:       "Query Tests",
		Description: "Queries test results and coverage data using SQL. Schema: all_tests, all_coverage, test_coverage, all_code. Uses a persistent database to avoid re-running tests on every query. Set rebuild=true after code changes to refresh the database.",
		Instruction: "*   **`query_tests`**: Query test results with SQL.\n    *   **Available Tables:** `all_tests`, `all_coverage`, `test_coverage`, `all_code`.\n    *   **Usage:** `query_tests(query=\"SELECT * FROM all_coverage WHERE count = 0\", language=\"go\")`\n    *   **Outcome:** Tabular results from the SQL query over test and coverage data.",
	},

	// --- LSP ---
	"describe": {
		Name:        "describe",
		Title:       "Describe Symbol",
		Description: "Returns type information and documentation for a symbol at a given position in a source file. Uses the Language Server Protocol for accurate, compiler-level results.",
		Instruction: "*   **`describe`**: Get type/hover information for a symbol.\n    *   **Usage:** `describe(file=\"main.go\", line=10, col=5)`\n    *   **Outcome:** Type signature and documentation for the symbol at that position.",
	},
	"find_definition": {
		Name:        "find_definition",
		Title:       "Find Definition",
		Description: "Navigates to the definition of a symbol at a given position. Uses the Language Server Protocol to resolve across files and packages.",
		Instruction: "*   **`find_definition`**: Jump to the definition of a symbol.\n    *   **Usage:** `find_definition(file=\"main.go\", line=10, col=5)`\n    *   **Outcome:** File path and line number where the symbol is defined.",
	},
	"find_references": {
		Name:        "find_references",
		Title:       "Find References",
		Description: "Finds all references to a symbol at a given position across the codebase. Uses the Language Server Protocol for precise cross-file analysis.",
		Instruction: "*   **`find_references`**: Find all usages of a symbol.\n    *   **Usage:** `find_references(file=\"main.go\", line=10, col=5)`\n    *   **Outcome:** List of file locations where the symbol is referenced.",
	},
	"rename_symbol": {
		Name:        "rename_symbol",
		Title:       "Rename Symbol",
		Description: "Performs a deterministic, project-wide rename of a symbol using the Language Server Protocol. Ensures all references and imports are updated atomically.",
		Instruction: "*   **`rename_symbol`**: Perform a project-wide rename.\n    *   **Usage:** `rename_symbol(file=\"main.go\", line=10, col=5, new_name=\"NewName\")`\n    *   **Outcome:** All references are updated, followed by a project health report.",
	},
	// --- AGENTS ---
	"review_code": {
		Name:        "review_code",
		Title:       "Review Code",
		Description: "Provides an automated architectural and idiomatic review of source code. Identifies potential defects and optimization opportunities before code is committed.",
		Instruction: "*   **`review_code`**: Perform an automated expert review.\n    *   **Usage:** `review_code(file=\"...\")`\n    *   **Outcome:** A structured critique identifying potential bugs and optimization opportunities.",
	},
}
