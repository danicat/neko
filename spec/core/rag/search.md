# Search Mechanism

## Overview
Once the ingestion pipeline has built the vector database, the `search` tool (`internal/tools/lang/search`) allows clients to query the codebase using natural language.

## Implementation Steps

1. **Query Processing**:
   - The MCP tool handler receives a natural language query string from the client.

2. **Query Embedding**:
   - The query string is passed to the same embedding model used during ingestion to generate a query vector.

3. **Similarity Search**:
   - The RAG engine performs a cosine similarity search (or a similar nearest-neighbor algorithm) within the local vector database.
   - It compares the query vector against all stored chunk vectors to find the closest matches.

4. **Result Assembly**:
   - The engine returns `[]rag.SearchResult`, a wrapper type that encapsulates content, metadata, and similarity score without leaking the underlying `chromem` dependency.
   - Results are formatted into Markdown snippet blocks containing the raw chunk content, file path, line number, symbol name, and similarity score.
   - No additional context window extraction is performed — the result contains the chunk as stored in the vector database.
