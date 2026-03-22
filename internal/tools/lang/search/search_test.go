package search

import (
	"context"
	"fmt"
	"testing"

	"github.com/danicat/neko/internal/core/rag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	ragEnabled bool
	root       string
	results    []rag.SearchResult
	searchErr  error
}

func (ts *testServer) RAGSearch(_ context.Context, _ string, _ int) ([]rag.SearchResult, error) {
	if ts.searchErr != nil {
		return nil, ts.searchErr
	}
	return ts.results, nil
}

func (ts *testServer) RAGEnabled() bool {
	return ts.ragEnabled
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

func textContent(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestSearchHandler_EmptyQuery(t *testing.T) {
	s := &testServer{ragEnabled: true, root: "/tmp/test"}
	_, _, err := searchHandler(context.TODO(), Params{Query: ""}, s)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if err.Error() != "query is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSearchHandler_NoResults(t *testing.T) {
	s := &testServer{ragEnabled: true, root: "/tmp/test", results: nil}
	res, _, err := searchHandler(context.TODO(), Params{Query: "something"}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textContent(res)
	if text != "No relevant code snippets found for your query." {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestSearchHandler_WithResults(t *testing.T) {
	s := &testServer{
		ragEnabled: true,
		root:       "/tmp/test",
		results: []rag.SearchResult{
			{
				Content:    "func Hello() {}",
				Metadata:   map[string]string{"path": "main.go", "line": "10", "name": "Hello"},
				Similarity: 0.95,
			},
		},
	}
	res, _, err := searchHandler(context.TODO(), Params{Query: "hello function"}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textContent(res)
	if text == "" {
		t.Fatal("expected non-empty output")
	}
	for _, want := range []string{"main.go", "Hello", "0.95", "func Hello()"} {
		if !contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestSearchHandler_SearchError(t *testing.T) {
	s := &testServer{
		ragEnabled: true,
		root:       "/tmp/test",
		searchErr:  fmt.Errorf("connection refused"),
	}
	_, _, err := searchHandler(context.TODO(), Params{Query: "test"}, s)
	if err == nil {
		t.Fatal("expected tool error on search failure")
	}
	if !contains(err.Error(), "connection refused") {
		t.Errorf("expected error message in output, got: %v", err)
	}
}

func TestSearchHandler_LimitDefaults(t *testing.T) {
	// Limit <= 0 should default to 5, limit > 10 should cap at 10.
	// We can't inspect the limit passed to RAGSearch directly, but we
	// can at least verify no crash.
	s := &testServer{ragEnabled: true, root: "/tmp/test", results: nil}

	for _, limit := range []int{-1, 0, 5, 15} {
		_, _, err := searchHandler(context.TODO(), Params{Query: "test", Limit: limit}, s)
		if err != nil {
			t.Fatalf("limit=%d: unexpected error: %v", limit, err)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
