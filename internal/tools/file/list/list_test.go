package list

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	reg  *backend.Registry
	root string
}

func (ts *testServer) Registry() *backend.Registry {
	return ts.reg
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

func TestListHandler_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	s := &testServer{reg: reg, root: tmpDir}

	res, _, err := listHandler(context.TODO(), nil, Params{Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	text := textContent(res)
	if !strings.Contains(text, "Found 0 files") {
		t.Errorf("expected 'Found 0 files' in output, got: %s", text)
	}
}

func TestListHandler_WithFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	// Create some files
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	reg := backend.NewRegistry()
	s := &testServer{reg: reg, root: tmpDir}

	res, _, err := listHandler(context.TODO(), nil, Params{Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	text := textContent(res)
	if !strings.Contains(text, "a.txt") || !strings.Contains(text, "b.txt") {
		t.Errorf("expected files in output, got: %s", text)
	}
}

func TestListHandler_DepthLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list-depth-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	// Create nested directories: a/b/c/deep.txt
	deep := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a", "top.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	s := &testServer{reg: reg, root: tmpDir}

	// Depth 1 should not include deep.txt
	res, _, err := listHandler(context.TODO(), nil, Params{Dir: tmpDir, Depth: 1}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textContent(res)
	if strings.Contains(text, "deep.txt") {
		t.Errorf("depth=1 should not include deep.txt, got: %s", text)
	}
}

func TestListHandler_DefaultDirUsesProjectRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list-default-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	s := &testServer{reg: reg, root: tmpDir}

	// Empty Dir should fall back to ProjectRoot
	res, _, err := listHandler(context.TODO(), nil, Params{}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := textContent(res)
	if !strings.Contains(text, "root.txt") {
		t.Errorf("expected root.txt in output, got: %s", text)
	}
}

func TestListHandler_SkipsDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list-skip-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	// Create a .git directory with a file inside
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	s := &testServer{reg: reg, root: tmpDir}

	// walkDir path (non-git repo temp dir)
	res, _, _ := walkDir(tmpDir, 5, defaultSkipDirs())
	text := textContent(res)
	if strings.Contains(text, "HEAD") {
		t.Errorf("expected .git to be skipped, got: %s", text)
	}
	if !strings.Contains(text, "visible.txt") {
		t.Errorf("expected visible.txt in output, got: %s", text)
	}
	_ = s // used above
}

func TestWalkDir_NonExistentDir(t *testing.T) {
	res, _, err := walkDir("/nonexistent-dir-abc123", 5, nil)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	text := textContent(res)
	if !strings.Contains(text, "Error walking") && !strings.Contains(text, "Found 0 files") {
		t.Errorf("expected error or empty result for nonexistent dir, got: %s", text)
	}
}
