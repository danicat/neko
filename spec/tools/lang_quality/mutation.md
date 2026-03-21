# Mutation Testing Logic

## Overview
The `test_mutations` tool (`internal/tools/lang/mutation/mutation.go`) measures the objective quality of a test suite by actively trying to break the code.

## Implementation Details

1. **Delegation**:
   - Calls `LanguageBackend.MutationTest`.

2. **Backend Execution**:
   - The backend runs a mutation testing framework (e.g., `go-mutesting` for Go).
   - It introduces subtle logical bugs into the AST (e.g., changing `<` to `>`, or `==` to `!=`).
   - It runs the test suite against each mutation.

3. **Reporting**:
   - If the tests pass despite the code being broken, the mutation "survived."
   - The tool returns a list of survived mutations to the LLM, indicating exact lines where the test assertions are too weak or missing.
