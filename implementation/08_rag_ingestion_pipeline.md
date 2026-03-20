# Task 8: RAG Ingestion Pipeline

## Context
To support intent-based discovery, Neko needs a semantic search index. This requires integrating a vector store and implementing a symbol-aware ingestion logic.

## TODO
- [ ] Integrate `chromem-go` as the local vector store.
- [ ] Set up persistent storage at `.neko/embeddings.db`.
- [ ] Implement "Symbol-Aware Chunking": use `documentSymbol` boundaries to define chunks.
- [ ] Implement the Ingestion Header: `[File Path] + [Package] + [Imports]`.
- [ ] Implement Synchronous Re-indexing: trigger an update immediately following a successful `didSave` in `edit_file`.

## NOT TODO
- [ ] Do not use fixed-size token window chunking.
- [ ] Do not block the MCP turn indefinitely; strictly adhere to the 1.0s latency budget for local embeddings.

## Acceptance Criteria
- [ ] `embeddings.db` is populated on project open.
- [ ] Modifications to files are reflected in the vector store immediately (synchronously).
- [ ] Chunks are logically bounded by functions and classes.
