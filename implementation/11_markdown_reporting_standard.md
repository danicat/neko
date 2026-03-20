# Task 11: Markdown Reporting Standard

## Context
JSON output for errors is inconsistent and prone to glitches. v0.2.0 mandates a single, unified Markdown template for all semantic feedback.

## TODO
- [ ] Create a shared utility for formatting diagnostics as Markdown.
- [ ] Standardize `build`, `edit_file`, `create_file`, `multi_edit`, and `rename_symbol` to use this utility.
- [ ] Structure: Success Message -> Contextual Advice (Syntax snippets) -> Flat Global Error List -> Total Count.
- [ ] Ensure all regex-extracted line numbers are used to provide the `->` pointer in syntax errors.

## NOT TODO
- [ ] Do not categorize errors as "New" or "Pre-existing."
- [ ] Do not prune any diagnostics from the LSP list.

## Acceptance Criteria
- [ ] All error-reporting tools return identical formatting.
- [ ] No JSON is present in the final agent-facing response.
- [ ] Error messages are easy for the agent to parse and act on.
