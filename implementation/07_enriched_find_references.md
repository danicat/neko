# Task 7: Enriched Find References

## Context
Raw coordinates are not enough for complex planning. Agents need to know the context of a reference to prioritize their edits.

## TODO
- [ ] Upgrade the `find_references` tool to support "Enrichment."
- [ ] For every reference address, perform an internal `hover` or `documentSymbol` lookup to find the containing function/method name.
- [ ] Format the output into two clear categories: `[SOURCE]` and `[TESTS]`.
- [ ] Include the context string (e.g., `(in 'func main')`) for every entry.

## NOT TODO
- [ ] Do not classify references as "Internal" or "External"; keep it to Source vs Tests.
- [ ] Do not read the file content unless the LSP context lookup fails.

## Acceptance Criteria
- [ ] The output clearly distinguishes between implementation breakage and verification breakage.
- [ ] The agent can identify the purpose of a reference without calling `read_file`.
