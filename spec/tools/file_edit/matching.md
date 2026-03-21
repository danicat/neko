# Fuzzy Matching Logic

## Overview
When an LLM attempts to replace a block of code (providing `old_content` and `new_content`), the `old_content` rarely matches the exact byte sequence of the source file due to whitespace variations, indentation, or minor drift. Neko uses a multi-phase matching algorithm to robustly locate the target block.

## Implementation Steps

1. **Normalization & Character Mapping**:
   - The `normalize(s string)` function strips all whitespace, producing a condensed comparison string. This is applied to both `old_content` and the file content.
   - A `charMap` slice records each non-whitespace rune alongside its **original byte offset** in the unmodified file. This allows Neko to map match positions in normalized space back to precise byte ranges in the original source.
   - All index arithmetic uses **rune-based slicing** (via `[]rune()` conversions) to correctly handle multi-byte UTF-8 characters.

2. **Exact Substring Match (Fast Path)**:
   - `findMatches` first attempts `strings.Cut(normContent, normSearch)` — a direct substring search in normalized space.
   - If found, the match is returned immediately with a perfect score of `1.0`, using the `charMap` to recover original byte offsets.
   - This fast path avoids any distance calculation for the common case where whitespace is the only difference.

3. **Seed-Based Candidate Detection (Fuzzy Path)**:
   - If no exact match is found, Neko uses a **seed hashing** strategy instead of a brute-force sliding window.
   - Short, fixed-length "seeds" (substrings of the normalized search text) are extracted at regular intervals. Seed length and step size adapt to search length:
     - `searchLen >= 64`: seed = 16 runes, step = 8
     - `searchLen >= 16`: seed = 8 runes, step = 4
     - `searchLen < 16`: seed = 4 runes, step = 2
   - Each seed is searched in the normalized content. When a seed hits, Neko projects the **candidate start position** by subtracting the seed's offset within the search text.
   - A candidate accumulator (`map[int]int`) counts how many seeds vote for each start position. Positions with more seed hits are more likely to contain the target block.

4. **Levenshtein Scoring**:
   - For each candidate position, a window of `searchLen` runes is extracted from the normalized content.
   - `similarity(s1, s2)` computes the Levenshtein edit distance and converts it to a score from 0.0 to 1.0.
   - Only candidates scoring above a minimum threshold (0.1) are retained.

5. **Deduplication & Selection**:
   - Results are sorted by score (descending), then by position (ascending).
   - Overlapping candidates within 10 bytes of each other are merged (keeping the higher-scoring one).
   - A configurable similarity threshold (default 0.85) determines the final acceptance cutoff.

6. **Disambiguation**:
   - If multiple locations match above the threshold, the tool rejects the edit as ambiguous, displaying all candidates with their scores and line ranges, and requests that the LLM provide a more unique `old_content` block.
