# File Create Logic

## Overview
The `create_file` tool (`internal/tools/file/create/create.go`) initializes a new source file while ensuring immediate LSP awareness and syntactic correctness.

## Implementation Details

1. **Directory Generation**:
   - Ensures all parent directories for the target file path are created (`os.MkdirAll`).

2. **File Initialization**:
   - Writes the initial content to disk.

3. **LSP Synchronization**:
   - Sends a `textDocument/didOpen` notification to the LSP.
   - Like `edit_file`, it captures synchronous diagnostics. This is crucial for catching errors like a missing package declaration or an invalid import immediately upon file creation.

4. **Formatting**:
   - Triggers the Language Backend's `Format` or `Modernize` routine to auto-style the newly created file to project standards.
