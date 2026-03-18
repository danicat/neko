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
	// --- FILE OPERATIONS ---
	"file_create": {
		Name:        "file_create",
		Title:       "Create File",
		Description: "Initializes a new source file, automatically creating parent directories and applying language-appropriate formatting. Ensures new files are immediately compliant with project style guides.",
		Instruction: "*   **`file_create`**: Initialize a new source file.\n    *   **Usage:** `file_create(filename=\"src/utils.go\", content=\"package utils\n...\")`\n    *   **Outcome:** A correctly formatted, directory-synced file is created.",
	},
	"smart_edit": {
		Name:        "smart_edit",
		Title:       "Smart Edit",
		Description: "An intelligent file editor providing robust block matching and safety guarantees. Automatically handles formatting, import optimization, and syntax verification to ensure edits do not break the code.",
		Instruction: "*   **`smart_edit`**: The primary tool for safe code modification.\n    *   **Capabilities:** Validates syntax and auto-formats (gofmt/goimports for Go, ruff for Python) *before* committing changes to disk.\n    *   **Robustness:** Uses fuzzy matching to locate target blocks despite minor whitespace or indentation variances.\n    *   **Usage:** `smart_edit(filename=\"...\", old_content=\"...\", new_content=\"...\")`.\n    *   **Append Mode:** Leave `old_content` empty to append to the end of the file.\n    *   **Outcome:** A syntactically valid, properly formatted file update.",
	},
	"smart_read": {
		Name:        "smart_read",
		Title:       "Read File",
		Description: "A structure-aware file reader that optimizes for context density. Supports returning full content, structural outlines (AST-based signatures), or specific line ranges to minimize token consumption.",
		Instruction: "*   **`smart_read`**: Inspect file content and structure.\n    *   **Read All:** `smart_read(filename=\"pkg/utils.go\")`\n    *   **Outline:** `smart_read(filename=\"pkg/utils.go\", outline=true)` (Retrieve types and function signatures only).\n    *   **Snippet:** `smart_read(filename=\"pkg/utils.go\", start_line=10, end_line=50)` (Targeted range reading).\n    *   **Outcome:** Targeted source content or structural map.",
	},
	"list_files": {
		Name:        "list_files",
		Title:       "List Files",
		Description: "Recursively lists files and directories while filtering out build artifacts and version control data. Provides an accurate view of the source code hierarchy.",
		Instruction: "*   **`list_files`**: Explore the project structure.\n    *   **Usage:** `list_files(path=\".\", depth=2)`\n    *   **Outcome:** A hierarchical list of source files and directories.",
	},

	// --- DOCS ---
	"read_docs": {
		Name:        "read_docs",
		Title:       "Get Documentation",
		Description: "Retrieves documentation for any package or symbol. Supports Go (godoc) and Python (pydoc). Streamlines development by providing API signatures and usage examples directly within the workflow.",
		Instruction: "*   **`read_docs`**: Access API documentation.\n    *   **Usage:** `read_docs(import_path=\"net/http\")` or `read_docs(import_path=\"pathlib\")`\n    *   **Outcome:** API reference and usage guidance.",
	},

	// --- TOOLCHAIN ---
	"smart_build": {
		Name:        "smart_build",
		Title:       "Smart Build",
		Description: "The primary build tool. Enforces a quality gate pipeline appropriate to the detected language. Ensures code is production-ready.",
		Instruction: "*   **`smart_build`**: Compile and verify code.\n    *   **Usage:** `smart_build(packages=\"./...\", auto_fix=true)` (Go) or `smart_build(dir=\".\", auto_fix=true)` (Python)\n    *   **Outcome:** A comprehensive report on build status, test results, and lint issues.",
	},
	"add_dependency": {
		Name:        "add_dependency",
		Title:       "Add Dependency",
		Description: "Manages package installation and manifest updates. Consolidates the workflow by immediately returning the public API documentation for the installed packages.",
		Instruction: "*   **`add_dependency`**: Install dependencies and fetch documentation.\n    *   **Usage:** `add_dependency(packages=[\"github.com/gin-gonic/gin@latest\"])`\n    *   **Outcome:** Dependency added and API documentation returned.",
	},
	"project_init": {
		Name:        "project_init",
		Title:       "Initialize Project",
		Description: "Bootstraps a new project by creating the directory, initializing the module, and installing essential dependencies. Reduces boilerplate and ensures a standard project structure.",
		Instruction: "*   **`project_init`**: Bootstrap a new project.\n    *   **Usage:** `project_init(path=\"my-app\", module_path=\"github.com/user/my-app\", dependencies=[\"...\"])`\n    *   **Outcome:** A valid project with requested dependencies and a skeleton structure.",
	},
	"modernize_code": {
		Name:        "modernize_code",
		Title:       "Modernize Code",
		Description: "Analyzes the codebase for outdated patterns and automatically refactors them to modern standards. Improves maintainability and performance by applying idiomatic upgrades.",
		Instruction: "*   **`modernize_code`**: Automatically upgrade legacy patterns.\n    *   **Usage:** `modernize_code(dir=\".\", fix=true)`\n    *   **Outcome:** Source code refactored to modern standards.",
	},

	// --- TESTING ---
	"mutation_test": {
		Name:        "mutation_test",
		Title:       "Mutation Test",
		Description: "Runs mutation testing. Introduces small code mutations and checks if existing tests catch them, objectively measuring test suite quality.",
		Instruction: "*   **`mutation_test`**: Verify test quality with mutation testing.\n    *   **Usage:** `mutation_test(dir=\".\")`\n    *   **Outcome:** A report showing which mutations survived (tests missed) and the mutation score.",
	},
	"test_query": {
		Name:        "test_query",
		Title:       "Test Query",
		Description: "Queries test results and coverage data using SQL. Uses a persistent database to avoid re-running tests on every query. Set rebuild=true after code changes to refresh the database.",
		Instruction: "*   **`test_query`**: Query test results with SQL.\n    *   **Usage:** `test_query(query=\"SELECT * FROM all_coverage WHERE count = 0\")`\n    *   **Outcome:** Tabular results from the SQL query over test and coverage data.",
	},

	// --- LSP ---
	"symbol_info": {
		Name:        "symbol_info",
		Title:       "Symbol Info",
		Description: "Returns type information and documentation for a symbol at a given position in a source file. Uses the Language Server Protocol for accurate, compiler-level results.",
		Instruction: "*   **`symbol_info`**: Get type/hover information for a symbol.\n    *   **Usage:** `symbol_info(file=\"main.go\", line=10, col=5)`\n    *   **Outcome:** Type signature and documentation for the symbol at that position.",
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

	// --- AGENTS ---
	"code_review": {
		Name:        "code_review",
		Title:       "Code Review",
		Description: "Provides an automated architectural and idiomatic review of source code. Identifies potential defects and optimization opportunities before code is committed.",
		Instruction: "*   **`code_review`**: Perform an automated expert review.\n    *   **Usage:** `code_review(file_content=\"...\")`\n    *   **Outcome:** A structured critique identifying potential bugs and optimization opportunities.",
	},
}
