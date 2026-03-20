# Task 2: Synchronous Diagnostic Capture

## Context
MCP is a synchronous, turn-based protocol. Neko must block until the LSP has finished re-indexing after a save to return an accurate semantic health report. This requires a deterministic pull or a proxied push mechanism.

## TODO
- [ ] Implement `PullDiagnostics(uri string)` using the `workspace/diagnostic` request (LSP 3.17+).
- [ ] Implement a `diagnosticWatch` map (`uri -> chan struct{}`) in the `lsp.Client`.
- [ ] Update `readLoop` to intercept `textDocument/publishDiagnostics` and signal the corresponding channel.
- [ ] Create a unified `WaitForDiagnostics(uri string)` method that tries the pull model first and falls back to the notification proxy.
- [ ] Implement a strict 2.0s hard timeout for the wait.

## NOT TODO
- [ ] Do not use `time.Sleep` for polling.
- [ ] Do not prune or filter the diagnostics; return the full raw list from the server.

## Acceptance Criteria
- [ ] `WaitForDiagnostics` returns only after the LSP has processed the latest change.
- [ ] The tool does not hang if the LSP server crashes (timeout handled).
- [ ] Diagnostics from other files (regressions) are successfully captured.
