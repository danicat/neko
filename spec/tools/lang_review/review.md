# Code Review Logic

## Overview
The `review_code` tool (`internal/tools/lang/codereview/review.go`) provides an AI-assisted critique of the implementation.

## Implementation Details

1. **Context Gathering**:
   - The tool reads the target file and gathers related context using `ragEngine` or LSP structural knowledge.

2. **Analysis**:
   - Unlike static linters (which check syntax), this tool checks for architectural compliance, idiomatic style, and semantic correctness against the project's broader guidelines.

3. **Output**:
   - Returns a structured Markdown report highlighting suggestions for improvement, potential edge cases, and design feedback.
