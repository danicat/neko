# Definition and References Logic

## Overview
These tools allow the LLM to traverse the codebase graph.

## Implementation Details

1. **`find_definition` (`internal/tools/lang/definition`)**:
   - Triggers `textDocument/definition` using the provided file, line, and column.
   - Returns the absolute file path, line, and column where the symbol under the cursor was defined.

2. **`find_references` (`internal/tools/lang/references`)**:
   - Triggers `textDocument/references` using the provided file, line, and column.
   - **Enrichment**: Neko post-processes the array of returned locations. It often checks file names (e.g., matching `_test.go` or `test_*.py`) to categorize the output into `[SOURCE]` usages versus `[TESTS]` usages. This categorization helps the LLM decide if a change requires updating business logic or merely updating test assertions.
