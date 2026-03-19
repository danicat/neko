// Package codereview implements the AI-powered code review tool.
package codereview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/genai"
)

var (
	// ErrVertexAIMissingConfig indicates that Vertex AI is enabled but project/location configuration is missing.
	ErrVertexAIMissingConfig = fmt.Errorf("vertex AI enabled but missing configuration")

	// ErrAuthFailed indicates that no valid authentication credentials were found.
	ErrAuthFailed = fmt.Errorf("authentication failed")
)

// Server defines the interface required by the tool.
type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
}

// Register registers the review_code tool with the server.
func Register(mcpServer *mcp.Server, s Server, defaultModel string) {
	reviewHandler, err := NewHandler(context.Background(), defaultModel)
	if err != nil {
		if errors.Is(err, ErrAuthFailed) || errors.Is(err, ErrVertexAIMissingConfig) {
			fmt.Fprintf(os.Stderr, "WARN: Disabling review_code tool: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: Disabling review_code tool: failed to create handler: %v\n", err)
		}
		return
	}
	def := toolnames.Registry["review_code"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return reviewHandler.Tool(ctx, req, args, s)
	})
}

// Params defines the input parameters for the review_code tool.
type Params struct {
	File        string `json:"file,omitempty" jsonschema:"Optional: path to the file to review"`
	FileContent string `json:"file_content,omitempty" jsonschema:"Optional: raw source code to review (if file is not provided)"`
	ModelName   string `json:"model_name,omitempty" jsonschema:"Optional: Gemini model to use"`
	Hint        string `json:"hint,omitempty" jsonschema:"Optional: specific area to focus the review on"`
}

// Suggestion defines the structured output for a single review suggestion.
type Suggestion struct {
	LineNumber int    `json:"line_number"`
	Severity   string `json:"severity"`
	Finding    string `json:"finding"`
	Comment    string `json:"comment"`
}

// Result defines the structured output for the code_review tool.
type Result struct {
	Suggestions []Suggestion `json:"suggestions"`
}

// ContentGenerator abstracts the generative model for testing.
type ContentGenerator interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content,
		config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

// RealGenerator wraps the actual GenAI client.
type RealGenerator struct {
	client *genai.Client
}

// GenerateContent generates content using the underlying GenAI client.
func (r *RealGenerator) GenerateContent(ctx context.Context, model string, contents []*genai.Content,
	config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return r.client.Models.GenerateContent(ctx, model, contents, config)
}

// Handler holds the dependencies for the review code tool.
type Handler struct {
	generator    ContentGenerator
	defaultModel string
}

// Option is a function that configures a Handler.
type Option func(*Handler)

// WithGenerator sets the ContentGenerator for the Handler.
func WithGenerator(generator ContentGenerator) Option {
	return func(h *Handler) {
		h.generator = generator
	}
}

// NewHandler creates a new Handler.
func NewHandler(ctx context.Context, defaultModel string, opts ...Option) (*Handler, error) {
	handler := &Handler{
		defaultModel: defaultModel,
	}
	for _, opt := range opts {
		opt(handler)
	}

	if handler.generator == nil {
		var config *genai.ClientConfig

		useVertex := os.Getenv("GOOGLE_GENAI_USE_VERTEXAI")
		if useVertex == "true" || useVertex == "1" {
			project := os.Getenv("GOOGLE_CLOUD_PROJECT")
			location := os.Getenv("GOOGLE_CLOUD_LOCATION")

			if project == "" || location == "" {
				return nil, fmt.Errorf("%w: set GOOGLE_CLOUD_PROJECT and GOOGLE_CLOUD_LOCATION", ErrVertexAIMissingConfig)
			}

			config = &genai.ClientConfig{
				Project:  project,
				Location: location,
				Backend:  genai.BackendVertexAI,
			}
		} else {
			apiKey := os.Getenv("GOOGLE_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("GEMINI_API_KEY")
			}

			if apiKey == "" {
				return nil, fmt.Errorf("%w: set GOOGLE_API_KEY (or GEMINI_API_KEY) "+
					"for Gemini API, or set GOOGLE_GENAI_USE_VERTEXAI=true with GOOGLE_CLOUD_PROJECT "+
					"and GOOGLE_CLOUD_LOCATION for Vertex AI", ErrAuthFailed)
			}

			config = &genai.ClientConfig{
				APIKey:  apiKey,
				Backend: genai.BackendGeminiAPI,
			}
		}

		client, err := genai.NewClient(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create genai client: %w", err)
		}
		handler.generator = &RealGenerator{client: client}
	}
	return handler, nil
}

var jsonMarkdownRegex = regexp.MustCompile("(?s)```json" + "\\s*(.*?)" + "```")

// Tool performs an AI-powered code review and returns structured data.
func (h *Handler) Tool(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (
	*mcp.CallToolResult, any, error) {

	content := args.FileContent
	if args.File != "" {
		data, err := os.ReadFile(args.File)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to read file %s: %v", args.File, err)},
				},
			}, nil, nil
		}
		content = string(data)
	}

	if content == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "either 'file' or 'file_content' must be provided"},
			},
		}, nil, nil
	}

	modelName := h.defaultModel
	if args.ModelName != "" {
		modelName = args.ModelName
	}

	systemPrompt := constructSystemPrompt(args.Hint)

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: content},
			},
		},
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: systemPrompt},
			},
		},
	}

	resp, err := h.generator.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to generate content: %v", err)},
			},
		}, nil, nil
	}

	return processResponse(resp)
}

func processResponse(resp *genai.GenerateContentResponse) (*mcp.CallToolResult, *Result, error) {
	if !isValidResponse(resp) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "no response content from model. Check model parameters and API status"},
			},
		}, nil, nil
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part.Text == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "unexpected response format from model, expected text content"},
			},
		}, nil, nil
	}

	suggestions, err := parseReviewResponse(part.Text)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to parse model response: %v", err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: renderReviewMarkdown(suggestions)},
		},
	}, &Result{Suggestions: suggestions}, nil
}

func parseReviewResponse(text string) ([]Suggestion, error) {
	cleanedJSON := jsonMarkdownRegex.ReplaceAllString(text, "$1")

	var suggestions []Suggestion
	if err := json.Unmarshal([]byte(cleanedJSON), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal suggestions: %w", err)
	}
	return suggestions, nil
}

func renderReviewMarkdown(suggestions []Suggestion) string {
	if len(suggestions) == 0 {
		return "## Code Review\n\nNo issues found. Great job!"
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("## Code Review\n\nFound %d issues.\n\n", len(suggestions)))

	for _, s := range suggestions {
		icon := "ℹ️"
		switch strings.ToLower(s.Severity) {
		case "error":
			icon = "🚨"
		case "warning":
			icon = "⚠️"
		case "suggestion":
			icon = "💡"
		}

		buf.WriteString(fmt.Sprintf("### %s Line %d: %s\n", icon, s.LineNumber, s.Finding))
		buf.WriteString(fmt.Sprintf("**Severity:** %s\n\n", s.Severity))
		buf.WriteString(s.Comment)
		buf.WriteString("\n\n---\n\n")
	}
	return buf.String()
}

func isValidResponse(resp *genai.GenerateContentResponse) bool {
	return resp != nil && len(resp.Candidates) > 0 &&
		resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0
}

func constructSystemPrompt(hint string) string {
	prompt := `You are a Senior Staff Engineer conducting a code review. Your standards are high, focused on correctness, safety, and long-term maintainability.

**Review Standards:**
- Language-specific best practices and idioms
- Concurrency safety and resource management
- Error handling completeness
- API design and naming conventions
- Testing adequacy

**Severity Levels:**
- **"error":** Bugs, race conditions, panics, security vulnerabilities, or silent error drops.
- **"warning":** Non-idiomatic code, performance traps, or testing anti-patterns.
- **"suggestion":** Naming improvements, comment clarity, or simplification opportunities.

**Output Format:**
Return a purely RAW JSON array (no markdown fencing).
Example:
[
  {
    "line_number": 42,
    "severity": "error",
    "finding": "Resource Leak",
    "comment": "This resource is never closed. Ensure proper cleanup."
  }
]

If the code is perfect, return [].`

	if hint != "" {
		prompt = fmt.Sprintf("Focus strictly on this specific area: \"%s\".\n\n%s", hint, prompt)
	}
	return prompt
}
