package create

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
		Filename: filePath,
		Content:  "package lib\n\nfunc A() {}",
	}, reg)
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
	reg.Register(golang.New())

	filePath := filepath.Join(tmpDir, "main.go")

	// Valid syntax with missing import (goimports should add it)
	res, _, _ := createHandler(context.TODO(), nil, Params{
		Filename: filePath,
		Content:  "package main\n\nfunc main() { fmt.Println(NonExistent) }",
	}, reg)

	output := res.Content[0].(*mcp.TextContent).Text
	if strings.Contains(output, "WARNING") {
		t.Errorf("unexpected warning for valid syntax: %s", output)
	}

	// Invalid syntax
	resErr, _, _ := createHandler(context.TODO(), nil, Params{
		Filename: filePath,
		Content:  "package main\n\nfunc main() { this is invalid syntax }",
	}, reg)
	outputErr := resErr.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(outputErr, "WARNING") {
		t.Errorf("expected syntax check warning, got: %s", outputErr)
	}
}
