# Unified Semantic Engine

## Overview
The semantic engine is the subsystem that augments raw LSP responses with resolved type information, providing the LLM with deep type context without requiring it to navigate to definition files. It is used by both `read_file` (Type Info footer) and `describe` (Enhanced Hover).

## Key Components

### EnhancedHover (`client.EnhancedHover`)
- Wraps the standard `textDocument/hover` response with a **one-level definition lookup heuristic**.
- If the initial hover text is short (< 200 characters), it jumps to the symbol's definition via `Definition()` and fetches the hover there. If the definition's hover is richer (longer), it returns that instead — giving the LLM struct/interface details rather than just a variable declaration.
- For `var` declarations, it prepends the original signature to the definition's hover for full context.
- Returns a structured Markdown string combining the hover content and any resolved type information.

### Type Info Footer (in `read_file`)
- When reading a file with an active LSP backend, Neko identifies all identifier references in the file content.
- For each identifier, it queries the LSP for hover/type information.
- Standard library symbols are filtered out via `be.IsStdLibURI(loc.URI)` to reduce noise.
- The resolved type signatures are appended as a **Type Info** footer section after the file content.

### Session-Level Deduplication (`seenTypeInfo`)
- The `Server` maintains a `seenTypeInfo map[string]bool` that tracks which type signatures have already been shown to the LLM in the current session.
- `read_file` consults this map (via `Server.HasSeenTypeInfo`) before including type info in its footer. Note: `describe` declares `HasSeenTypeInfo` on its Server interface but does not currently use it — it always returns full detail.
- This prevents the same struct definition or interface contract from being repeated across multiple reads/describes, keeping token usage minimal.
- The map is protected by `Server.mu` and reset when the project is closed.

## Design Rationale
Without the semantic engine, the LLM would need to:
1. Read a file → see an unfamiliar type → call `find_definition` → read the definition file → come back to the original file.

With the semantic engine, step 2–4 are eliminated: the type information is delivered inline, saving multiple tool round-trips per file read.
