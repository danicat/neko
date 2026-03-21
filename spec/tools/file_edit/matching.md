# Fuzzy Matching Logic

## Overview
When an LLM attempts to replace a block of code (providing `old_content` and `new_content`), the `old_content` rarely matches the exact byte sequence of the source file due to whitespace variations, indentation, or minor drift. Neko uses a fuzzy matching algorithm to robustly locate the target block.

## Implementation Steps

1. **Normalization**:
   - The `normalize(s string)` function strips all extraneous whitespace, converting tabs and spaces into a consistent, minimal format. This is applied to both the `old_content` and the target file contents.

2. **Sliding Window Search**:
   - The `findMatches` function iterates through the normalized file content using a sliding window roughly the size of the normalized `old_content`.

3. **Levenshtein Calculation**:
   - For each window, it calls `internal/core/textdist.Levenshtein` to compute the edit distance (the number of insertions, deletions, or substitutions required to match).

4. **Similarity Threshold**:
   - The `similarity(s1, s2)` function converts the raw edit distance into a percentage (0.0 to 1.0).
   - If the similarity exceeds a configurable threshold (e.g., 0.95), the window is recorded as a match candidate.

5. **Disambiguation**:
   - If multiple locations match above the threshold, the tool rejects the edit as ambiguous, requesting that the LLM provide a more unique `old_content` block.
