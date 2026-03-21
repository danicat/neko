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
