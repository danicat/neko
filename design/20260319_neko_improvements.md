# Design: Neko Core Improvements (March 19, 2026)

## Overview
This document outlines the architectural and tool-level improvements implemented on March 19, 2026, to enhance the output quality and performance of Neko.

## 1. Optimized Test Coverage Reporting

### Context
Previous Go test coverage reporting in the `build` tool included redundant and confusing output such as `coverage: statements`.

### Solution
- **Consolidation**: Implemented a two-pass parser for Go test output.
- **Reporting**:
    - Packages with non-zero coverage are listed individually with their percentage.
    - Packages with 0% coverage or no test files are consolidated into a single comma-separated list.
- **Benefits**: Reduces vertical space in reports and provides a clearer picture of project health.

## 2. Language-Aware Documentation Memoization

### Context
`read_file` automatically injects documentation for imported packages. For large projects with many similar imports, this led to redundant output and excessive token usage.

### Solution
- **Stateful Tracking**: Added a `seenDocs` map to the `Server` struct, keyed by `[language][package]`.
- **Logic**:
    - `ShouldShowDoc(language, pkg)` method checks if documentation has been shown in the current project session.
    - If shown, the documentation is omitted from `read_file` output.
- **Scope**: Memoization only affects the automated `read_file` injection. Explicit calls to `read_docs` bypass memoization to allow users to re-view documentation on demand.
- **Session Lifecycle**: Memoization is reset whenever a project is opened, created, or closed.

## 3. Integrated Modernization Pipeline

### Context
`modernize_code` was a standalone tool, which added friction to the standard development workflow.

### Solution
- **Unified Toolchain**: Integrated the modernization step directly into the `build` tool's pipeline.
- **Control**: Added a `run_modernize` (bool) parameter to the `build` tool.
- **Auto-Fix**: Modernization fixes are automatically applied if `auto_fix` is true, providing a seamless "check and fix" experience.
- **De-registration**: The standalone `modernize_code` tool was removed to simplify the tool surface.

## 4. Neko Agent Skill

### Context
Ensuring that AI agents follow the recommended Neko development workflow (Project First -> Explore -> Edit -> Verify).

### Solution
- **Standardized Skill**: Created a formal `neko-development` skill.
- **Content**:
    - Defines the core philosophy and multi-step workflow.
    - Provides language-specific nuances (e.g., `uv` for Python, SQL for Go tests).
- **Packaging**: Distributed as a `.skill` package for easy installation and consistent behavior across agent instances.
