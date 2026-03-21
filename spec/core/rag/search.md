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

4. **Result Assembly & Context Windowing**:
   - The engine retrieves the top-N closest metadata records (file path, line ranges).
   - To provide actionable context, it uses the line ranges to extract not just the raw chunk, but also the surrounding lines of code (a "context window") from the source file.
   - These enriched results are formatted into snippet blocks and returned to the LLM.
