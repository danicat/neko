package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/core/rag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	reg *backend.Registry
}

func (ts *testServer) ForFile(_ context.Context, path string) backend.LanguageBackend {
	return ts.reg.ForFile(path)
}

func (ts *testServer) RAG() *rag.Engine {
	return nil
}

type noLSPBackend struct {
	golang.Backend
}

func (b *noLSPBackend) LSPCommand() (string, []string, bool) {
	return "", nil, false
}

func (b *noLSPBackend) Validate(ctx context.Context, file string) error {
	content, _ := os.ReadFile(file)
	if strings.Contains(string(content), "invalid syntax") {
		return fmt.Errorf("mock syntax error at line 1")
	}
	return nil
}

func (b *noLSPBackend) Name() string { return "nolsp" }

func (b *noLSPBackend) IsStdLibURI(uri string) bool { return false }

func TestCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module create-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	reg.Register(golang.New())

	filePath := filepath.Join(tmpDir, "lib.go")

	res, _, _ := createHandler(context.TODO(), nil, Params{
		File:    filePath,
		Content: "package lib\n\nfunc A() {}",
	}, &testServer{reg: reg})
	if res.IsError {
		t.Fatalf("Initial write failed: %v", res.Content[0].(*mcp.TextContent).Text)
	}

	//nolint:gosec // G304: Test file path.
	content, _ := os.ReadFile(filePath)
	if !strings.Contains(string(content), "func A()") {
		t.Errorf("expected func A() in file, got: %s", string(content))
	}
}

func TestCreate_Validation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-val-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module val-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reg := backend.NewRegistry()
	reg.Register(&noLSPBackend{})

	filePath := filepath.Join(tmpDir, "main.go")

	// Valid syntax with missing import (goimports should add it)
	res, _, _ := createHandler(context.TODO(), nil, Params{
		File:    filePath,
		Content: "package main\n\nfunc main() { fmt.Println() }",
	}, &testServer{reg: reg})

	output := res.Content[0].(*mcp.TextContent).Text
	if strings.Contains(output, "WARNING") {
		t.Errorf("unexpected warning for valid syntax: %s", output)
	}

	// Invalid syntax - in v0.2.0, we don't block on syntax, but we report it via LSP diagnostics.
	// Since testServer has no LSP, it should fall back to backend.Validate and show a WARNING.
	resErr, _, _ := createHandler(context.TODO(), nil, Params{
		File:    filePath,
		Content: "package main\n\nfunc main() { invalid syntax }",
	}, &testServer{reg: reg})
	if resErr.IsError {
		t.Errorf("expected success despite invalid syntax, got error: %s", resErr.Content[0].(*mcp.TextContent).Text)
	}
	outputErr := resErr.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(outputErr, "WARNING") {
		t.Errorf("expected syntax check warning, got: %s", outputErr)
	}
}
