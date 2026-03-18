# Proposal: Minimum Common Denominator for Neko Core Tools (Revised)

## Objective
Extract a language-agnostic core set of tools while maintaining the "smart_" branding for code-aware operations. This ensures `neko` provides a consistent, high-intelligence experience across all supported languages.

## 1. Core Tool Set (The "Neko Standard")

These 11 tools represent the universal operations every code-aware agent needs. The "smart_" prefix signifies that the tool uses the `LanguageBackend` for validation, formatting, or structural analysis.

| Tool Name | Original Name | Domain | Why it's Core |
| :--- | :--- | :--- | :--- |
| **`list_files`** | `list_files` | Navigation | Project structure discovery (git-aware). |
| **`smart_read`** | `smart_read` | Context | Reading code with line numbers and structure (outline). |
| **`smart_create`**| `file_create` | Modification | Initializing new files with auto-formatting/validation. |
| **`smart_edit`** | `smart_edit` | Modification | Surgical, safe edits with syntax verification. |
| **`smart_build`** | `smart_build` | Toolchain | Universal quality gate (build/lint/test). |
| **`read_docs`** | `read_docs` | Documentation | Fetching API/Library documentation (pydoc/godoc/etc). |
| **`add_dependency`**| `add_dependency` | Toolchain | Managing project requirements/package installation. |
| **`project_init`** | `project_init` | Toolchain | Bootstrapping new projects. |
| **`symbol_info`** | `symbol_info` | Intelligence | LSP-based type/hover info (Language Agnostic). |
| **`find_definition`**| `find_definition` | Intelligence | LSP-based navigation. |
| **`find_references`**| `find_references` | Intelligence | LSP-based impact analysis. |

## 2. Extended / Language-Specific Tools

These remain available but are considered "Level 2" tools that may not be available for all plugins:

- `modernize_code`: Automated refactoring.
- `mutation_test`: Test suite quality metrics.
- `test_query`: SQL-based test analysis.
- `code_review`: LLM-based architectural review.

## 3. Implementation Plan

### Phase 1: Harmonization
1. Update `internal/toolnames/registry.go` to rename `file_create` to `smart_create`.
2. Ensure all descriptions use language-neutral terminology (e.g., "codebase" instead of "packages", "manifest" instead of "go.mod").

### Phase 2: Backend Generalization
Ensure the `LanguageBackend` interface methods are named to reflect these core operations:
- `BuildPipeline` remains the engine for `smart_build`.
- `Format` and `Validate` remain the engines for `smart_create` and `smart_edit`.

## Success Criteria
- The "smart_" prefix is consistently used for all tools that perform automated code analysis or modification.
- Descriptions are 100% language-neutral, making the server equally welcoming to Go, Python, and JS developers.
- `smart_create` replaces `file_create` to complete the branded toolset.
