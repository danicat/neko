# Task 13: LSP-Powered Outline Mode

## Context
The `outline` parameter in `read_file` is highly valued by agents. We will upgrade its implementation from custom backend parsers to the LSP's authoritative `textDocument/documentSymbol`.

## TODO
- [ ] Implement `lsp.Client.DocumentSymbol(uri string)` to fetch hierarchical symbols.
- [ ] Update the `read` tool to use the LSP implementation when `outline=true` is requested.
- [ ] Create a formatter to convert the LSP symbol tree into the concise Markdown outline agents expect.
- [ ] Preserve the existing backend `Outline` methods as a fallback for non-LSP languages.

## NOT TODO
- [ ] Do not remove the `outline` parameter from the tool signature.
- [ ] Do not delete the legacy backend parsing logic until all languages have mature LSP support.

## Acceptance Criteria
- [ ] `read_file(outline=true)` returns high-fidelity, deterministic structural summaries via the LSP.
- [ ] The output format remains consistent with Neko v0.1 for backward compatibility.
- [ ] Non-LSP languages continue to provide outlines via their native backends.
