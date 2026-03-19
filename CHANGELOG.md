# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-03-19

### Added
- **Project Lifecycle**: Implemented a two-phase lifecycle (Lobby and Project Open) to ensure valid context before engineering operations.
- **Unified Quality Gate**: Integrated modernization checks and auto-fixes directly into the `build` pipeline for Go and Python.
- **Documentation Memoization**: Added session-based, language-keyed tracking in `read_file` to reduce redundant documentation injection.
- **Enhanced Coverage Reporting**: Improved Go build output to accurately list all zero-coverage packages, including those without tests.
- **Language Plugins**: Added initial support for JavaScript/TypeScript via a plugin-based system with built-in ESLint modernization.
- **Agent Skills**: Introduced specialized skills for AI agents:
  - `neko-development`: Teaches the professional Neko workflow.
  - `test-quality-optimizer`: Data-driven workflow for increasing test suite robustness using coverage and mutation analysis.
- **Release Automation**: Configured `goreleaser` matching professional standards for multi-platform distribution.

### Changed
- **Standardized Toolset**: Migrated all tools to snake_case naming and standardized arguments (`file`, `dir`, `language`, etc.).
- **Modernized Codebase**: Applied idiomatic Go 1.24 improvements across the entire project (iterators, `any`, `min/max`).
- **Improved Readability**: Redesigned documentation (`README.md`, `DOCUMENTATION.md`, `NEKO.md`) for clarity and technical accuracy.

### Removed
- Standalone `modernize_code` tool (now integrated into `build`).
- Redundant parsing artifacts from Go test coverage reports.
