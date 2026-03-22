---
name: neko-development-flow
description: Specialized workflow for ALL software development tasks using Neko. It provides the natural steps, tool usages, and a strict development loop for requirements analysis, exploration, and safe, verified modifications.
---

# Neko Development Flow

This skill provides the mandatory workflows and principles for using **Neko** as a Semantic Operating System for software development. It MUST be applied to ANY software development task.

## Mandatory Development Workflow

You must adhere to the following strict loop when undertaking any software development task:

1.  **Analyze the Requirements**: Understand the goal. Read the issue, prompt, or request carefully.
2.  **Disambiguate**: If any points are unclear, ask the user for feedback before proceeding.
3.  **Admit Knowledge Gaps**: If you don't know something, state it clearly. Do not hallucinate capabilities or facts.
4.  **Explore Before Changing**: Always establish context. Call `open_project(dir=".")` and explore the existing project structure (`list_files`, `read_file`, `semantic_search`) *before* making any modifications.
5.  **Ground All Claims**: Do not make assumptions. Ground your understanding and proposed solutions in hard evidence retrieved from the codebase (e.g., via `read_file` or `describe`).
6.  **The Target Loop**: Focus on doing **one thing at a time**.
    - Edit the code (`edit_file`, `line_edit`, `multi_edit`, `rename_symbol`).
    - Verify immediately. Go end-to-end: review the LSP diagnostic output from the edit, run `build(auto_fix=true)`, and verify acceptance criteria using tests.
    - **A target is not complete until it is fully validated end-to-end.**
7.  **Boy Scout Rule (Continuous Improvement)**: Always leave the codebase in a better state than you found it.
    - If you encounter warnings, deprecations, style errors, or minor bugs during your task, **fix them immediately**.
    - If they are major architectural flaws or require significant deviation, add an item to a "backlog" (note it in your response) and tackle it immediately *after* the current task is done.
8.  **Manual Commits**: **Only** commit code if explicitly instructed by the user. Do not assume automatic commits.
9.  **Complete the Task (Documentation)**: After completing the core logic for a task, you **must always** update the relevant documentation, `README.md`, `CHANGELOG.md`, and any supporting files. A task is incomplete if the docs are stale.
10. **Release Protocol**: When instructed to perform a release, ensure you update ALL related files (manifests, changelogs, configuration) and git tags accordingly.

---

## Tool Usage Guide

-   **Context Start**: Always begin by calling `open_project(dir=".")`.
-   **Discovery**:
    -   `list_files` to understand directory layout.
    -   `semantic_search` to find code by intent.
    -   `multi_read` or `read_file(outline=true)` to gather context efficiently.
-   **Code Intelligence**:
    -   Rely on the **Type Info** footer in `read_file`.
    -   Use `describe` for deep type analysis.
    -   Use `find_references` to safely map impact before refactoring.
-   **Editing**:
    -   `line_edit`: For surgical line-range replacements.
    -   `edit_file`: For fuzzy block matching.
    -   `multi_edit`: For interdependent changes across multiple files. Submit "Complete Thoughts" to keep diagnostics actionable.
    -   `rename_symbol`: The **mandatory** way to perform project-wide deterministic renames.
-   **Verification**:
    -   Use the **Full Disclosure** diagnostic report returned automatically by edit tools to immediately steer your next action.
    -   Use `build(dir=".", auto_fix=true)` as the ultimate quality gate.
    -   Use `test_mutations` to verify test suite robustness.

## Language Specifics

### 🐹 Go Development (Golang)
- Standard toolchains and `gopls` LSP are active.
- `rename_symbol` is authoritative.
- Use `query_tests` to analyze `go test` results using SQL.

### 🐍 Python Development
- Projects **must** use `uv`.
- `ruff` manages linting and formatting automatically during saves.
- `modernize` utilizes `ruff` with `pyupgrade` rules.