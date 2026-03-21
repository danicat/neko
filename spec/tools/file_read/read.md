# File Read Logic

## Overview
The `read_file` tool (`internal/tools/file/read/read.go`) is optimized for context density and token efficiency. It is aware of the structural AST of the files it reads.

## Implementation Details

1. **Standard Reading**:
   - Reads a file from disk or from the VFS if the file is currently being modified.
   - Supports `start_line` and `end_line` for targeted snippets.

2. **Outline Mode**:
   - If the `outline=true` flag is passed, Neko delegates to the Language Backend (`Outline(ctx, filename)`).
   - The backend uses AST parsing (e.g., Go's `go/parser` and `go/ast`) to strip out function bodies and implementations.
   - It returns only the structural skeleton: struct definitions, interface contracts, and function signatures. This allows the LLM to understand the shape of a large file without consuming thousands of tokens reading its implementation.

3. **Semantic Virtual Annotations** *(Future/Conceptual)*:
   - Capable of injecting `<NEKO>` tags containing LSP-derived type information directly into the read output.
