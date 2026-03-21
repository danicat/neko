# Go Implementation

## Overview
The Go language backend (`internal/backend/golang`) leverages standard Go toolchain commands and integrates heavily with `gopls`.

## Implementation Details

1. **Detection**:
   - Matches if `go.mod` exists in the workspace.

2. **LSP Binding**:
   - `LSPCommand()` returns `gopls` with arguments to ensure semantic tokens and hover capabilities are enabled.

3. **BuildPipeline**:
   - Executes `go build ./...` to verify compilation.
   - Executes `go test -cover ./...` to run tests.
   - Importantly, it parses the standard error output of the Go compiler and test runner, translating raw strings into structured diagnostic formats that Neko can present cleanly to the LLM.

4. **Modernization & Formatting**:
   - `Format()` invokes standard `go fmt` and `goimports` to ensure files adhere to Go conventions before they are read back or committed.

5. **Advanced**:
   - Supports ingesting test coverage data into Neko's local `testdb` for SQL querying via the `testquery` tool.
