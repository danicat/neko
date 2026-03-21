# Ingestion Pipeline

## Overview
To enable semantic search, Neko must first build a searchable index of the codebase. This is handled by a background ingestion pipeline that runs when a project is opened.

## Implementation Steps

1. **Background Crawl Trigger**:
   - When a project is established via `open_project`, the server invokes `crawlProject` as a background goroutine to avoid blocking the MCP response.
   - The crawl context is derived from the app context (`s.appCtx`), ensuring the goroutine is cancelled on both project re-open/close and app shutdown (SIGINT/SIGTERM).

2. **File Walk & Filtering**:
   - The crawler traverses the project directory using `filepath.Walk`, identifying supported source files via `registry.ForFile()`.
   - It skips directories using a combination of hardcoded entries (`.git`, `.neko`) and backend-provided skip dirs (`registry.AllSkipDirs()`), returning `filepath.SkipDir` for efficient pruning. It does not respect `.gitignore` rules.

3. **Parsing & Chunking**:
   - Supported files are parsed to extract meaningful semantic units rather than arbitrary line breaks. 
   - Code is split into chunks such as function declarations, classes, and docstrings.

4. **Vector Embedding**:
   - Each chunk of text is passed to a local embedding model (or an API, if configured in `config`).
   - The model generates a high-dimensional vector representation of the code's semantic meaning.

5. **Storage**:
   - The resulting vectors, along with essential metadata (file path, line ranges, and the raw chunk text), are stored in a local vector database.
   - This database is typically persisted in a `.neko/embeddings.db/` directory within the project root to cache embeddings between sessions.
