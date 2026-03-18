package edit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestEdit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(golang.New())

	content := `package main
import "fmt"

func main() {
	fmt.Println("Hello")
}
`
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		search   string
		replace  string
		expected string
	}{
		{
			"Simple Replace",
			"fmt.Println(\"Hello\")",
			"fmt.Println(\"Goodbye\")",
			"fmt.Println(\"Goodbye\")",
		},
		{
			"Whitespace Agnostic",
			"func main() {\n\tfmt.Println(\"Goodbye\")\n}",
			"func main() { fmt.Println(\"Modified\") }",
			"fmt.Println(\"Modified\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := editHandler(context.TODO(), nil, Params{
				Filename:   filePath,
				OldContent: tt.search,
				NewContent: tt.replace,
			}, reg)
			if err != nil {
				t.Fatalf("editHandler failed: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %v", res.Content[0].(*mcp.TextContent).Text)
			}

			//nolint:gosec // G304: Test file path.
			newContent, _ := os.ReadFile(filePath)
			if !strings.Contains(string(newContent), tt.expected) {
				t.Errorf("expected %q in content, got: %s", tt.expected, string(newContent))
			}
		})
	}
}

func TestEdit_Broken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edit-broken-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module broken\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(golang.New())

	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Invalid Syntax
	res, _, _ := editHandler(context.TODO(), nil, Params{
		Filename:   filePath,
		OldContent: "func main() {}",
		NewContent: "func main() { invalid syntax }",
	}, reg)
	if !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "edit produced invalid code") {
		t.Errorf("expected error for invalid syntax, got: %s", res.Content[0].(*mcp.TextContent).Text)
	}
}
