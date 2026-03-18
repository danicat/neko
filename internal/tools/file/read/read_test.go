package read

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

func TestRead(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()
	reg.Register(golang.New())

	srcFile := filepath.Join(tmpDir, "main.go")
	src := `package main

import "fmt"

type MyStruct struct {
	Name string
}

func (s *MyStruct) Greet() string {
	return "Hello " + s.Name
}

func main() {
	fmt.Println("Hello")
}
`
	if err := os.WriteFile(srcFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	// Full read
	res, _, err := readHandler(context.Background(), nil, Params{Filename: srcFile}, reg)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if res.IsError {
		t.Errorf("tool returned error: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("no content returned")
	}

	output := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(output, "fmt.Println") {
		t.Errorf("expected source content, got: %s", output)
	}
}

func TestRead_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	roots.Global.Add(tmpDir)

	reg := backend.NewRegistry()

	srcFile := filepath.Join(tmpDir, "partial.go")
	src := `line 1
line 2
line 3
line 4
line 5`
	if err := os.WriteFile(srcFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	res, _, err := readHandler(context.Background(), nil, Params{
		Filename:  srcFile,
		StartLine: 2,
		EndLine:   4,
	}, reg)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent).Text

	if !strings.Contains(text, "   2 | line 2") {
		t.Errorf("expected line 2, got: %s", text)
	}
	if !strings.Contains(text, "   4 | line 4") {
		t.Errorf("expected line 4, got: %s", text)
	}
	if strings.Contains(text, "   1 | line 1") {
		t.Errorf("did not expect line 1, got: %s", text)
	}
	if strings.Contains(text, "   5 | line 5") {
		t.Errorf("did not expect line 5, got: %s", text)
	}
	if !strings.Contains(text, "Partial read - analysis skipped") {
		t.Error("expected partial read warning")
	}
}
