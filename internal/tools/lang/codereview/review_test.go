package codereview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"google.golang.org/genai"
)

type testServer struct {
	be   backend.LanguageBackend
	root string
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.be
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

type mockGenerator struct {
	resp *genai.GenerateContentResponse
	err  error
}

func (g *mockGenerator) GenerateContent(_ context.Context, _ string, _ []*genai.Content,
	_ *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return g.resp, g.err
}

func TestTool_EmptyContent(t *testing.T) {

	h := &Handler{
		generator:    &mockGenerator{},
		defaultModel: "test-model",
	}
	s := &testServer{root: "/tmp/test"}

	_, _, err := h.Tool(context.TODO(), nil, Params{}, s)

	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "either 'file' or 'file_content' must be provided") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTool_WithFileContent(t *testing.T) {
	jsonResp := `[{"line_number": 1, "severity": "warning", "finding": "Test Issue", "comment": "Test comment"}]`
	h := &Handler{
		generator: &mockGenerator{
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: jsonResp},
							},
						},
					},
				},
			},
		},
		defaultModel: "test-model",
	}
	s := &testServer{root: "/tmp/test"}

	_, data, err := h.Tool(context.TODO(), nil, Params{FileContent: "package main"}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := data.(*Result)
	if !ok {
		t.Fatal("expected *Result from structured data")
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result.Suggestions))
	}
	if result.Suggestions[0].Finding != "Test Issue" {
		t.Errorf("unexpected finding: %s", result.Suggestions[0].Finding)
	}
}

func TestTool_WithFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "review-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	jsonResp := `[]`
	h := &Handler{
		generator: &mockGenerator{
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: jsonResp},
							},
						},
					},
				},
			},
		},
		defaultModel: "test-model",
	}
	s := &testServer{root: tmpDir}

	_, _, err = h.Tool(context.TODO(), nil, Params{File: filePath}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTool_FileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "review-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	h := &Handler{
		generator:    &mockGenerator{},
		defaultModel: "test-model",
	}
	s := &testServer{root: tmpDir}

	filePath := filepath.Join(tmpDir, "nonexistent.go")
	_, _, err = h.Tool(context.TODO(), nil, Params{File: filePath}, s)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTool_GenerateError(t *testing.T) {
	h := &Handler{
		generator: &mockGenerator{
			err: fmt.Errorf("API rate limit"),
		},
		defaultModel: "test-model",
	}
	s := &testServer{root: "/tmp/test"}

	_, _, err := h.Tool(context.TODO(), nil, Params{FileContent: "package main"}, s)
	if err == nil {
		t.Fatal("expected error for generation failure")
	}
	if !strings.Contains(err.Error(), "API rate limit") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTool_NilResponse(t *testing.T) {
	h := &Handler{
		generator: &mockGenerator{
			resp: nil,
		},
		defaultModel: "test-model",
	}
	s := &testServer{root: "/tmp/test"}

	_, _, err := h.Tool(context.TODO(), nil, Params{FileContent: "package main"}, s)
	if err == nil {
		t.Fatal("expected error for nil response")
	}
	if !strings.Contains(err.Error(), "no response content") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTool_CustomModel(t *testing.T) {
	// Verify that custom model name is accepted (mock doesn't validate model name)
	jsonResp := `[]`
	h := &Handler{
		generator: &mockGenerator{
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: jsonResp},
							},
						},
					},
				},
			},
		},
		defaultModel: "default-model",
	}
	s := &testServer{root: "/tmp/test"}

	_, _, err := h.Tool(context.TODO(), nil, Params{FileContent: "package main", ModelName: "custom-model"}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReviewResponse_MarkdownFencing(t *testing.T) {
	input := "```json\n[{\"line_number\": 1, \"severity\": \"error\", \"finding\": \"Bug\", \"comment\": \"Fix it\"}]\n```"
	suggestions, err := parseReviewResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Finding != "Bug" {
		t.Errorf("unexpected finding: %s", suggestions[0].Finding)
	}
}

func TestParseReviewResponse_InvalidJSON(t *testing.T) {
	_, err := parseReviewResponse("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRenderReviewMarkdown_Empty(t *testing.T) {
	output := renderReviewMarkdown(nil)
	if !strings.Contains(output, "No issues found") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestRenderReviewMarkdown_WithSuggestions(t *testing.T) {
	suggestions := []Suggestion{
		{LineNumber: 10, Severity: "error", Finding: "Resource Leak", Comment: "Close it"},
		{LineNumber: 20, Severity: "warning", Finding: "Naming", Comment: "Rename it"},
		{LineNumber: 30, Severity: "suggestion", Finding: "Style", Comment: "Consider this"},
	}
	output := renderReviewMarkdown(suggestions)
	if !strings.Contains(output, "Found 3 issues") {
		t.Errorf("expected issue count in output, got: %s", output)
	}
	if !strings.Contains(output, "Resource Leak") {
		t.Errorf("expected finding in output, got: %s", output)
	}
}

func TestConstructSystemPrompt_WithHint(t *testing.T) {
	prompt := constructSystemPrompt("concurrency")
	if !strings.Contains(prompt, "concurrency") {
		t.Errorf("expected hint in prompt, got: %s", prompt)
	}
}

func TestConstructSystemPrompt_NoHint(t *testing.T) {
	prompt := constructSystemPrompt("")
	if strings.Contains(prompt, "Focus strictly") {
		t.Errorf("did not expect focus directive without hint, got: %s", prompt)
	}
}

func TestNewHandler_WithGenerator(t *testing.T) {
	gen := &mockGenerator{}
	h, err := NewHandler(context.TODO(), "model", WithGenerator(gen))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.generator != gen {
		t.Error("expected custom generator to be set")
	}
	if h.defaultModel != "model" {
		t.Errorf("expected default model 'model', got %s", h.defaultModel)
	}
}
