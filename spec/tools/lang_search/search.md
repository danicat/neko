# Semantic Search Logic

## Overview
The `semantic_search` tool (`internal/tools/lang/search/search.go`) provides the entry point for the RAG engine described in the core architecture.

## Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `query` | Yes | — | Natural language query (e.g., "handling of lsp client lifecycle") |
| `limit` | No | 5 | Maximum number of results to return (capped at 10) |

## Implementation Details

1. **Handler Delegation**:
   - The tool receives a natural language query string and an optional result limit.
   - It delegates the query to `s.RAG().Search(ctx, query, limit)`.
   - The tool is only registered if the RAG engine was successfully initialized (i.e., `ragEngine != nil`).

2. **Result Formatting**:
   - The engine returns a list of semantic matches, each with a similarity score and metadata (path, line, symbol name).
   - The tool formats these matches into a Markdown block. Each match includes:
     - File path and line number
     - Symbol name (if available)
     - Similarity score
     - Code snippet of the matching content
   - This allows the LLM to read relevant implementation directly from the search results without needing to open each file.
