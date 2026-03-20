# Task 10: Virtual Semantic Annotations

## Context
To provide "IDE-like" type awareness, Neko will inject virtual comments into `read_file` output. This reduces the number of turns an agent spends checking types.

## TODO
- [ ] Implement the annotation engine in the `read` tool.
- [ ] Logic: `documentSymbol` -> Filter Symbols -> `hover` for types -> Inject `<NEKO>...</NEKO>` tags.
- [ ] Follow "High-Signal Rules": skip built-in primitives, include struct definitions for user types, fully qualify cross-package symbols.
- [ ] Implement the "Agent Alert" footer in the tool response.

## NOT TODO
- [ ] Do not use language-specific comment symbols (e.g. `//` or `#`); use the bare `<NEKO>` tag.
- [ ] Do not annotate every local variable; focus on declarations and complex inferences.

## Acceptance Criteria
- [ ] `read_file` output contains clear type-signature metadata.
- [ ] The agent can see the underlying definition of a user-defined type without calling `describe`.
- [ ] Annotations are clearly visually distinct from source code.
