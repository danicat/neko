# Task 10: Unified Semantic Engine & Type Info

## Context
To provide deep type awareness without mutilating source code, Neko will append a "Type Info" footer to `read_file` output using an "Enhanced Describe Backend."

## TODO
- [ ] Implement `lsp.EnhancedHover` or extend the `describe` tool logic to support depth-limited (+1) recursive resolution.
- [ ] Implement method set extraction from the LSP response for structs/interfaces.
- [ ] Implement the "Type Info" footer aggregator in `read_file`. It must:
    - Determine symbols within the read line bounds.
    - Fetch their Enhanced Describe data.
    - Format them sequentially in the "Dense Log" format (`- Line X sym (*Type): Doc...`).
- [ ] Clean up legacy code: remove all `<NEKO>` tag stripping logic from `shared/lines.go` and `edit_file/edit.go` since the footer approach guarantees pristine code blocks.

## NOT TODO
- [ ] Do not inject tags or banners inline with the source code.
- [ ] Do not recurse more than 1 level deep when resolving types.

## Acceptance Criteria
- [ ] `read_file` output ends with a structured `## Type Info` block.
- [ ] Complex types show their extracted method sets.
- [ ] The core source block perfectly matches the file on disk.t from source code.
