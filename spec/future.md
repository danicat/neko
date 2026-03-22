# Future Improvements & Enhancements

This document tracks planned features, user-requested improvements, and architectural enhancements for the Neko project.

## Tool Improvements

### File Tools
- **`read_file` (Outline Mode + Docstrings)**:
  - **Feature**: Enhance the AST parser in outline mode to print docstrings alongside function and type signatures.
  - **Rationale**: Currently, outline mode strips out implementations to save tokens, but often strips crucial context held within docstrings. Exposing docstrings will improve the LLM's understanding of intent without needing to read the entire file.

### Language Tools
- **`query_tests` (Schema Awareness)**:
  - **Feature**: Bake the `testquery` database schema (`all_tests`, `all_coverage`, `test_coverage`, `all_code`) directly into the MCP tool descriptions or `NEKO.md` instructions.
  - **Rationale**: To prevent LLM hallucinations regarding table names (e.g., querying a non-existent `coverage` table instead of `all_coverage`), the system prompt or tool description needs to explicitly define the available tables and their columns. This ensures deterministic and accurate SQL generation right from the first turn.

## Architectural Enhancements

- **Backend-aware crawl filtering**: `crawlProject` currently uses hardcoded skip patterns (`.git`, `node_modules`). It should delegate to the active backends' `SkipDirs()` for consistent filtering with `list_files`.
- **Crawl resumption**: On re-open of the same project, reuse the existing RAG index instead of re-crawling from scratch.

## New Tool Ideas

### Batch Operations
- **`multi_read`**:
  - **Concept**: Similar to `multi_edit`, this tool would allow the agent to request the contents of multiple files in a single turn.
  - **Benefit**: Reduces round-trip latency and token overhead when the agent needs to analyze several related files (e.g., a source file and its corresponding test file) simultaneously.

### Line-Based Editing
- **`line_edit`**:
  - **Concept**: A surgical editing tool that operates on absolute line ranges (e.g., "replace lines 10-25 with this content") rather than fuzzy content matching.
  - **Benefit**: Simplifies the editing process when exact line numbers are known (from a previous `read_file` or diagnostic report), bypassing the complexity and potential failure points of `old_content` matching.
- **`multi_line_edit`**:
  - **Concept**: The batch version of `line_edit`, allowing multiple line-based replacements across one or more files in a single atomic transaction.

### Project Navigation
- **`search`**:
  - **Concept**: A high-performance string search tool (similar to `grep` or `ripgrep`) for finding literal text patterns across the project.
  - **Note**: This may be redundant with the existing `semantic_search` (RAG), but provides a deterministic fallback for finding specific identifiers or unique strings that might not rank highly in a vector-based search.
