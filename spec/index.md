# Neko Specification Index

Welcome to the comprehensive specification of the Neko project. This documentation provides a step-by-step breakdown of every component, sufficient to recreate the project from scratch with 100% feature parity.

## Overview
- [High-Level Architecture & Executive Summary](highlevel.md)
- [Future Improvements & Enhancements](future.md)

## Core Components
- [Server (MCP Integration)](core/server/index.md)
- [Project Lifecycle & State Management (RAG)](core/rag/index.md)
- [LSP Manager & Client](core/lsp/index.md)
- [Language Backends](backend/language/index.md)

## File Tools
- [File Edit Tool (`edit_file` / `multi_edit`)](tools/file_edit/index.md)
- [File Read Tool (`read_file`)](tools/file_read/index.md)
- [File Create Tool (`create_file`)](tools/file_create/index.md)
- [File List Tool (`list_files`)](tools/file_list/index.md)

## Language Tools
- [Semantic Search (`semantic_search`)](tools/lang_search/index.md)
- [Symbol Rename Tool (`rename_symbol`)](tools/lang_rename/index.md)
- [Navigation: Definition & Symbol Info](tools/lang_navigation/index.md)
- [Find References (`find_references`)](tools/lang_references/index.md)
- [Read Docs (`read_docs`)](tools/lang_docs/index.md)
- [Add Dependencies (`add_dependencies`)](tools/lang_dependencies/index.md)
- [Quality Gate (`build` / `query_tests` / `test_mutations`)](tools/lang_quality/index.md)
- [Code Review (`review_code`)](tools/lang_review/index.md)