# Test Query Logic

## Overview
The `query_tests` tool (`internal/tools/lang/testquery/testquery.go`) allows the LLM to inspect test coverage without repeatedly executing slow test suites. It relies on the `testquery` CLI tool to build and query a SQLite database.

## Database Schema
The underlying test database consists of four main tables:
1. **`all_tests`**: Records the results of every test executed (`package`, `test`, `action`, `elapsed`, `output`).
2. **`all_coverage`**: Aggregated coverage data for the entire project (`file`, `function_name`, `start_line`, `end_line`, `count`).
3. **`test_coverage`**: Mapping of specific tests to the code blocks they cover (`test_name`, `file`, `start_line`, `end_line`, `count`).
4. **`all_code`**: A searchable copy of the project's source code (`file`, `line_number`, `content`).

## Implementation Details

1. **Ingestion Hook**:
   - When the tests are executed (e.g., `go test -coverprofile`), the `LanguageBackend.BuildTestDB` parses the output.
   - It ingests coverage lines and test results into a local SQLite database (`testquery.db`).

2. **SQL Interface**:
   - The `query_tests` tool provides a standard SQL interface.
   - The LLM can send queries like `SELECT DISTINCT file, function_name FROM all_coverage WHERE count = 0;` to find exactly which lines of code are missing test coverage.