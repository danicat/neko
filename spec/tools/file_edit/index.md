# File Edit Tool Specification

The `edit_file` tool (`internal/tools/file/edit/edit.go`) is the most critical and complex tool in Neko. It provides a stateful, fuzzy-matching, LSP-aware mechanism for mutating code safely.

## Sub-Components
- [Fuzzy Matching Logic](matching.md): Explains how Neko finds code to replace despite minor formatting differences.
- [Execution & Validation](execution.md): Details the synchronous LSP feedback loop that ensures edits don't break the codebase.
- [Multi-Edit Transaction](multi_edit.md): Explains how Neko handles atomic changes across multiple files.
