---
name: test-quality-optimizer
description: Data-driven workflow to increase project coverage and robustness. Use when mutation tests show survivors or coverage reports show gaps, and you need to propose or implement high-quality tests.
---

# Test Quality Optimizer Skill

This skill provides a surgical, data-driven workflow for improving test suites using **Coverage Analysis** and **Mutation Testing**.

## The Philosophy: Beyond Execution

Code coverage only proves a line was *executed*. Mutation testing proves the line was *verified*. This skill focuses on closing the gap between the two to ensure "Total Robustness."

## Core Workflow

### 1. Identify the Gaps (Discovery)

First, determine where the tests are missing or blind.

- **Go Projects**: Use `query_tests` to find uncovered blocks.
  - *Action*: `query_tests(query="SELECT file, start_line, function_name FROM all_coverage WHERE count = 0")`
- **Other Languages**: Use the `build()` report to find packages with < 80% coverage.

### 2. Find the Weaknesses (Mutation)

Run mutation tests to find logic that is executed but not properly asserted.

- **Action**: Run `test_mutations()`.
- **Analysis**:
  - **Uncovered**: No tests touch this code. (Highest Priority)
  - **Survived**: Tests touch the code, but failed to notice a logic change. (High Priority: Assertions are too weak).

### 3. Surgical Analysis

Read the source code at the exact coordinates of the survivor or gap.

- **Action**: Use `read_file` with `start_line` and `end_line` centered on the reported mutation.
- **Goal**: Identify the missing edge case or boundary condition (e.g., an `if` condition that is always true in tests, or an error path that is never triggered).

### 4. Implement "Mutation-Killing" Tests

Propose and write tests that specifically target the identified weakness.

- **For Go**: Use Table-Driven Tests. Add a new sub-test case to the existing `tests` slice that targets the uncovered boundary.
- **For Python**: Add a new `@pytest.mark.parametrize` case or a new test function targeting the survived logic.

### 5. Verify the Fix

Never assume a test is good until it passes the quality gate.

- **Action**: Run `build()` to ensure the new tests pass and coverage increased.
- **Action**: Re-run `test_mutations()` for the specific file/package to confirm the mutation is now **Killed**.

## Priorities

1.  **Zero Coverage Functions**: Code that is never touched by any test.
2.  **Survivors in Error Handling**: Code where error paths are executed but the error is ignored.
3.  **Boundary Survivors**: `>` changed to `>=` but tests still pass. This indicates missing edge-case coverage.
