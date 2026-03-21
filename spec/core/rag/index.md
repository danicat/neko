# RAG & Semantic Search

The `internal/core/rag` package implements local retrieval-augmented generation (RAG). This allows agents to perform intent-based discovery—searching the codebase by meaning rather than relying solely on exact keyword matches.

## Sub-Components
- [Ingestion Pipeline](ingestion.md): Explains how source code is parsed, chunked, and embedded into a local vector database.
- [Search Mechanism](search.md): Details how natural language queries are processed to retrieve semantic matches.
