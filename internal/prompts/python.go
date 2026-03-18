package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const pythonImportThisPrompt = `Your mission is to read the following documents:
https://peps.python.org/pep-0008/
https://peps.python.org/pep-0020/
https://peps.python.org/pep-0484/
https://peps.python.org/pep-0604/
https://docs.python.org/3/library/dataclasses.html
https://docs.python.org/3/library/typing.html
https://docs.pytest.org/en/stable/
https://docs.astral.sh/ruff/
https://mypy.readthedocs.io/en/stable/

And produce a comprehensive set of instructions for LLMs to code Python in an idiomatic,
type-safe, maintainable, testable, and easy to read way. Emphasize modern Python (3.12+)
patterns including: type annotations, match statements, dataclasses, pathlib, f-strings,
and the walrus operator where appropriate.`

const pythonCodeReviewPrompt = `You are conducting a senior-level Python code review. Apply this checklist systematically.

## Type Annotations (PEP 484/604)
- Are all function signatures type-annotated (parameters and return types)?
- Are complex types using modern syntax (X | Y instead of Union[X, Y])?
- Are TypeVar, Generic, Protocol used appropriately for generics?
- Is there a py.typed marker for typed packages?

## Error Handling
- No bare except: clauses?
- Are specific exceptions caught (ValueError, KeyError, not Exception)?
- Are custom exceptions defined for domain-specific errors?
- Is contextlib.suppress used instead of try/except/pass for known exceptions?

## Async/Await Correctness
- Are async functions only called with await?
- Is asyncio.gather or TaskGroup used for concurrent operations?
- Are blocking calls avoided in async code (use asyncio.to_thread)?
- Are async context managers used properly (async with)?

## Data Modeling
- Are dataclasses or Pydantic models used instead of plain dicts for structured data?
- Are frozen=True dataclasses used for immutable data?
- Is __slots__ used for performance-critical classes?

## Import Organization
- Are imports organized: stdlib, third-party, local (PEP 8)?
- Are relative imports used within packages?
- No circular imports?
- No wildcard imports (from x import *)?

## Testing (pytest)
- Are fixtures used for setup/teardown (not setUp/tearDown)?
- Is parametrize used for testing multiple inputs?
- Are assertions using plain assert (not self.assertEqual)?
- Are mocks used sparingly and only at boundaries?

## Security
- No eval() or exec() with untrusted input?
- No pickle.loads() from untrusted sources?
- Are secrets generated with secrets module, not random?
- Is subprocess called with shell=False?
- Are SQL queries parameterized, not formatted?

## After Review
- Run smart_build to verify all fixes pass format, lint, type check, and tests.
- Run modernize_code to catch outdated patterns.
- For an unbiased second opinion from a different model, use code_review.`

// PythonImportThis creates the definition for the Python 'import_this' prompt.
func PythonImportThis() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "python_import_this",
		Title:       "Import Python Philosophy",
		Description: "Produces a set of instructions for LLMs to write idiomatic and maintainable Python code.",
		Arguments:   nil,
	}
}

// PythonImportThisHandler is the handler for the Python 'import_this' prompt.
func PythonImportThisHandler(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: pythonImportThisPrompt},
			},
		},
	}, nil
}

// PythonCodeReview creates the definition for the Python code review prompt.
func PythonCodeReview() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "python_code_review",
		Title:       "Python Code Review",
		Description: "Senior-level Python code review checklist covering type annotations, error handling, async patterns, security, and pytest patterns.",
		Arguments: []*mcp.PromptArgument{
			{Name: "focus", Description: "Optional area to focus the review on (e.g. async, security, testing)", Required: false},
		},
	}
}

// PythonCodeReviewHandler generates the content for the Python code review prompt.
func PythonCodeReviewHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	prompt := pythonCodeReviewPrompt
	if focus := req.Params.Arguments["focus"]; focus != "" {
		prompt = fmt.Sprintf("**Focus this review specifically on: %s**\n\n%s", focus, prompt)
	}

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}
