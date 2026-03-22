package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClient_Versioning(t *testing.T) {
	// We need a real LSP server to test this properly, but we can at least test our state management.
	c := &Client{
		openedDocs: make(map[string]int),
	}

	tmpFile := filepath.Join(t.TempDir(), "test.go")
	if err := os.WriteFile(tmpFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	uri := FileURI(tmpFile)

	// Mock notify to avoid actual exec
	// This test is limited without a real server but verifies our internal state.
	c.mu.Lock()
	c.openedDocs[uri] = 1
	c.mu.Unlock()

	if v := c.GetVersion(tmpFile); v != 1 {
		t.Errorf("expected version 1, got %d", v)
	}

	// We'll trust the sequential logic in DidChange/DidSave for now
	// as they rely on the same mutex-protected openedDocs map.
}

func TestFormatDiagnostics(t *testing.T) {
	diags := map[string][]Diagnostic{
		"file:///test/workspace/test.go": {
			{
				Range:    Range{Start: Position{Line: 9, Character: 5}},
				Severity: 1,
				Message:  "error message",
			},
		},
	}

	report := FormatDiagnostics(diags, "/test/workspace")
	if !contains(report, "test.go:10:6") {
		t.Errorf("expected line:col 10:6 in report, got: %s", report)
	}
	if !contains(report, "[Error] error message") {
		t.Errorf("expected error message in report, got: %s", report)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
