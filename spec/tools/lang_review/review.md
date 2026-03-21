# Code Review Logic

## Overview
The `review_code` tool (`internal/tools/lang/codereview/review.go`) provides an AI-assisted critique of the implementation using Google's Gemini models.

## Prerequisites
- Requires authentication via one of:
  - `GOOGLE_API_KEY` or `GEMINI_API_KEY` (for Gemini API)
  - `GOOGLE_GENAI_USE_VERTEXAI=true` with `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION` (for Vertex AI)
- If no credentials are found, the tool is **silently disabled** during registration (not exposed to the LLM at all).

## Implementation Details

1. **Input**:
   - Accepts either a `file` path (read from disk) or raw `file_content`.
   - An optional `hint` parameter focuses the review on a specific area (e.g., "concurrency safety").
   - An optional `model_name` parameter overrides the default Gemini model.

2. **Analysis**:
   - The tool constructs a system prompt positioning the model as a Senior Staff Engineer.
   - The source code is sent to the Gemini model via the Google GenAI SDK (`google.golang.org/genai`).
   - The model evaluates correctness, safety, maintainability, idiomatic style, concurrency, error handling, and testing adequacy.

3. **Structured Output**:
   - The model returns a JSON array of `Suggestion` objects, each containing:
     - `line_number`: The source line in question.
     - `severity`: One of `"error"`, `"warning"`, or `"suggestion"`.
     - `finding`: A short title (e.g., "Resource Leak").
     - `comment`: Detailed explanation.
   - Neko parses this JSON and renders it as a Markdown report with severity icons (error, warning, suggestion) for LLM readability.

4. **Conditional Registration**:
   - `Register` calls `NewHandler` which attempts to create a GenAI client.
   - If authentication fails, `Register` returns early without adding the tool to the MCP server — the LLM never sees a broken tool.
