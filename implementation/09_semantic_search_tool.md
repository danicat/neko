# Task 9: Semantic Search Tool

## Context
Natural language discovery is a core pillar of v0.2.0. This tool allows agents to find patterns and intent without knowing specific symbol names.

## TODO
- [ ] Create the `semantic_search` MCP tool.
- [ ] Implement query embedding (Gemini or Local).
- [ ] Query the `chromem-go` database for the Top K most similar chunks.
- [ ] Return a list of results including: File Path, Line Range, Snippet, and Similarity Score.

## NOT TODO
- [ ] Do not implement complex reranking logic in this phase.
- [ ] Do not return more than 10 results to avoid context overflow.

## Acceptance Criteria
- [ ] The agent can find relevant code logic using fuzzy natural language queries.
- [ ] Results are semantically meaningful units (complete functions/classes).
