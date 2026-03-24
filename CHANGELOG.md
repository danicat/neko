# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.4] - 2026-03-24nn### Addedn- Added Vertex AI configuration fallback to RAG engine initialization (`GCP_PROJECT` and `GCP_LOCATION`).n- Enhanced security shell hooks to block pipeline bypasses using `awk`, `printf`, and scripting interpreters (`python`, `node`, etc.).nn### Changedn- Modified `open_project` behavior: missing RAG credentials now gracefully degrade the server by disabling the `semantic_search` tool rather than failing the entire project initialization.nn## [0.4.3] - 2026-03-24nn### Fixedn- Ensured `hooks/`, `skills/` and `plugins/` directories are properly included in the release archive configuration (`.goreleaser.yaml`).nn## [0.4.2] - 2026-03-22

### Added
- **`multi_read`**: New tool for batch reading of multiple files or line ranges to reduce token overhead.
- **`line_edit`**: New surgical editing tool that operates on absolute line ranges.
- **Enhanced Outline Mode**: Go and Python AST parsers now extract and include member docstrings (struct fields, interface methods, class methods) in the outline.
- **`query_tests` Schema Awareness**: Tool descriptions now explicitly define the available SQL tables (`all_tests`, `all_coverage`, etc.) for deterministic LLM usage.
- **Backend-Aware Crawling**: The RAG ingestion pipeline now respects backend-specific skip directories (e.g., `vendor`, `__pycache__`).

### Fixed
- Fixed documentation and metadata inconsistencies across `README.md`, `NEKO.md`, and `gemini-extension.json`.
- Ensured `hooks/` are included in release archives.

## [0.4.1] - 2026-03-22

### Fixed
- Included `hooks/` directory in Goreleaser archives.

## [0.4.0] - 2026-03-22

### Added
- **Security Hooks**: Implemented a comprehensive hook system to intercept and block generic file modification tools (e.g., `write_file`, `replace`, `run_shell_command`). This forces agents to use Neko's language-aware tools for project modifications.
- **Build Quality Gate Hook**: Intercepts generic build commands and redirects them to `build(auto_fix=true)`.

## [0.3.1] - 2026-03-21

### Changed
- Improved test coverage and specification documentation.

## [0.3.0] - 2026-03-21

### Fixed
- Fixed external tool calls in certain backend environments.

## [0.2.0] - 2026-03-20

### Added
- **Unified Semantic Engine**: Implemented `<NEKO>` semantic annotations in `read_file` to provide inline type signatures.
- **Enhanced `describe`**: Deep contextual analysis using LSP `hover`.
- **Improved `find_references`**: Categorized output into [SOURCE] and [TESTS] with symbol context.

## [0.1.1] - 2026-03-19

### Fixed
- Minor bug fixes in Go backend initialization.

## [0.1.0] - 2026-03-19

### Added
- **Project Lifecycle**: Implemented a two-phase lifecycle (Lobby and Project Open).
- **Unified Quality Gate**: Integrated modernization checks and auto-fixes directly into the `build` pipeline.
- **Documentation Memoization**: Added session-based tracking in `read_file`.
- **Enhanced Coverage Reporting**: Improved Go build output for zero-coverage packages.
- **Language Plugins**: Initial support for JavaScript/TypeScript.
- **Agent Skills**: Introduced `neko-development` and `test-quality-optimizer`.
- **Release Automation**: Configured `goreleaser`.
