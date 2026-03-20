# Task 5: Multi-Edit Transactional Tool

## Context
Agents often need to change multiple files to complete a single logical refactor. `multi_edit` allows them to submit these changes in one turn and receive a single semantic health report.

## TODO
- [ ] Create the `multi_edit` MCP tool.
- [ ] Implement iterative execution of the `edit_file` logic for a list of file/edit pairs.
- [ ] Implement "Best Effort" failure policy: if an edit fails, report it but continue with the others.
- [ ] Ensure that only **one** global diagnostic pull happens at the very end of the batch.
- [ ] Return a unified Markdown report summarizing all successes and failures.

## NOT TODO
- [ ] Do not implement complex two-phase commit logic; the agent is responsible for Git-level recovery.
- [ ] Do not return early on the first failure.

## Acceptance Criteria
- [ ] Multiple files are modified in a single turn.
- [ ] The agent receives a complete list of what succeeded and what failed.
- [ ] A project-wide diagnostic snapshot is returned at the end of the batch.
