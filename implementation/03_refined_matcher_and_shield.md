# Task 3: Refined Matcher & Annotation Shield

## Context
The `edit_file` tool needs to be safer and more helpful. It must prevent matching against virtual metadata and provide suggestions when a match fails.

## TODO
- [ ] Implement `AnnotationShield`: A utility to strip `<NEKO>...</NEKO>` tags from any string.
- [ ] Update `editHandler` to strip tags from `old_content` and `new_content` before processing.
- [ ] Modify `findBestMatch` to identify the Top 3 "Local Maxima" (windows with peak similarity) when the threshold (0.95) isn't met.
- [ ] Format match failures as a ranked Markdown list with line numbers and snippets.

## NOT TODO
- [ ] Do not change the underlying Levenshtein sliding window algorithm.
- [ ] Do not modify the file if any match is below the 0.95 threshold.

## Acceptance Criteria
- [ ] Agent-copied code containing `<NEKO>` tags matches successfully against raw disk code.
- [ ] Match failures provide 3 clear "Did you mean?" suggestions.
- [ ] Virtual tags are never written to the disk.
