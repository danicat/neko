# Test Query Logic

## Overview
The `query_tests` tool (`internal/tools/lang/testquery/testquery.go`) allows the LLM to inspect test coverage without repeatedly executing slow test suites. It delegates to the `testquery` CLI tool (`github.com/danicat/testquery`) to build and query a SQLite database.

## Database Schema
The underlying test database consists of four main tables:
1. **`all_tests`**: Records the results of every test executed (`package`, `test`, `action`, `elapsed`, `output`).
2. **`all_coverage`**: Aggregated coverage data for the entire project (`file`, `function_name`, `start_line`, `end_line`, `count`).
3. **`test_coverage`**: Mapping of specific tests to the code blocks they cover (`test_name`, `file`, `start_line`, `end_line`, `count`).
4. **`all_code`**: A searchable copy of the project's source code (`file`, `line_number`, `content`).

## Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `query` | Yes | — | SQL query to run against the test/coverage database |
| `dir` | No | Project root | Project directory containing the code to analyze |
| `language` | No | Auto-detect | Explicit language backend to use |
| `pkg` | No | `./...` | Package pattern to analyze (passed to `go test`) |
| `rebuild` | No | `false` | Force rebuild of the test database even if it exists |

## Implementation Details

1. **Automatic DB Building**:
   - If the database file (`testquery.db`) does not exist in the target directory, or if `rebuild=true`, the tool automatically runs `LanguageBackend.BuildTestDB` before executing the query.
   - For Go, `BuildTestDB` delegates to the `testquery` CLI tool, which runs `go test -coverprofile` and ingests the results into SQLite.
   - If tests fail but the database was still partially built, the tool proceeds with the query (the DB may still contain useful data).

2. **SQL Interface**:
   - The `query_tests` tool provides a standard SQL interface via `LanguageBackend.QueryTestDB`.
   - The LLM can send queries like `SELECT DISTINCT file, function_name FROM all_coverage WHERE count = 0;` to find exactly which lines of code are missing test coverage.
   - Results are returned in a tabular text format.
