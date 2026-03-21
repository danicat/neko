# Semantic Search Logic

## Overview
The `search` tool (`internal/tools/lang/search/search.go`) provides the entry point for the RAG engine described in the core architecture.

## Implementation Details

1. **Handler Delegation**:
   - The tool receives a natural language string.
   - It delegates the query directly to `s.RAG().Search(ctx, query)`.

2. **Result Formatting**:
   - The engine returns a list of semantic matches.
   - The tool formats these matches into an easy-to-read Markdown block. Each match includes the file path, line numbers, and a code snippet of the surrounding context, allowing the LLM to read the implementation directly from the search results.
