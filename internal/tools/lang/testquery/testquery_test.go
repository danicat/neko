package testquery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testServer struct {
	be         backend.LanguageBackend
	resolveErr error
	root       string
}

func (ts *testServer) ForFile(_ context.Context, _ string) backend.LanguageBackend {
	return ts.be
}

func (ts *testServer) ResolveBackend(_ string) (backend.LanguageBackend, error) {
	if ts.resolveErr != nil {
		return nil, ts.resolveErr
	}
	return ts.be, nil
}

func (ts *testServer) ProjectRoot() string {
	return ts.root
}

type mockBackend struct {
	name       string
	buildDBErr error
	queryOut   string
	queryErr   error
}

func (b *mockBackend) Name() string                          { return b.name }
func (b *mockBackend) LSPCommand() (string, []string, bool)  { return "", nil, false }
func (b *mockBackend) FileExtensions() []string              { return []string{".go"} }
func (b *mockBackend) SkipDirs() []string                    { return nil }
func (b *mockBackend) ProjectMarkers() []string              { return nil }
func (b *mockBackend) Tier() int                             { return 0 }
func (b *mockBackend) IsStdLibURI(_ string) bool             { return false }
func (b *mockBackend) LanguageID() string                    { return "go" }
func (b *mockBackend) InitializationOptions() map[string]any { return nil }
func (b *mockBackend) Capabilities() []backend.Capability    { return nil }

func (b *mockBackend) Outline(_ context.Context, _ string) (string, error)        { return "", nil }
func (b *mockBackend) ImportDocs(_ context.Context, _ []string) ([]string, error) { return nil, nil }
func (b *mockBackend) ParseImports(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (b *mockBackend) Validate(_ context.Context, _ string) error                 { return nil }
func (b *mockBackend) Format(_ context.Context, _ string) error                   { return nil }
func (b *mockBackend) BuildPipeline(_ context.Context, _ string, _ backend.BuildOpts) (*backend.BuildReport, error) {
	return nil, nil
}
func (b *mockBackend) FetchDocs(_ context.Context, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (b *mockBackend) AddDependency(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}
func (b *mockBackend) InitProject(_ context.Context, _ backend.InitOpts) error { return nil }
func (b *mockBackend) Modernize(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}
func (b *mockBackend) MutationTest(_ context.Context, _ string) (string, error) { return "", nil }
func (b *mockBackend) BuildTestDB(_ context.Context, _ string, _ string) error {
	return b.buildDBErr
}
func (b *mockBackend) QueryTestDB(_ context.Context, _ string, query string) (string, error) {
	return b.queryOut, b.queryErr
}
func (b *mockBackend) EnsureTools(_ context.Context, _ string) error { return nil }

func textContent(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestQueryHandler_EmptyQuery(t *testing.T) {
	s := &testServer{root: "/tmp/test"}

	_, _, err := queryHandler(context.TODO(), nil, Params{Query: ""}, s)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryHandler_ResolveBackendError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	s := &testServer{resolveErr: fmt.Errorf("no backend"), root: tmpDir}

	_, _, err = queryHandler(context.TODO(), nil, Params{Query: "SELECT 1", Dir: tmpDir}, s)
	if err == nil {
		t.Fatal("expected error for resolve failure")
	}
	if !strings.Contains(err.Error(), "no backend") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryHandler_BuildDBAndQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{
		name:     "test",
		queryOut: "| name | count |\n| foo | 5 |",
	}
	s := &testServer{be: be, root: tmpDir}

	// No DB file exists, so it should build it first
	res, _, err := queryHandler(context.TODO(), nil, Params{Query: "SELECT * FROM tests", Dir: tmpDir}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), "foo") {

		t.Errorf("expected query output, got: %s", textContent(res))
	}
}

func TestQueryHandler_QueryError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	// Create a fake DB file so build is skipped
	if err := os.WriteFile(filepath.Join(tmpDir, "testquery.db"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	be := &mockBackend{
		name:     "test",
		queryErr: fmt.Errorf("syntax error in SQL"),
	}
	s := &testServer{be: be, root: tmpDir}

	_, _, err = queryHandler(context.TODO(), nil, Params{Query: "INVALID SQL", Dir: tmpDir}, s)

	if err == nil {
		t.Fatal("expected error for query failure")
	}
	if !strings.Contains(err.Error(), "syntax error in SQL") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryHandler_BuildDBError_NoDBFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{
		name:       "test",
		buildDBErr: fmt.Errorf("tests failed"),
	}
	s := &testServer{be: be, root: tmpDir}

	// No DB file exists and build fails -> error
	_, _, err = queryHandler(context.TODO(), nil, Params{Query: "SELECT 1", Dir: tmpDir}, s)
	if err == nil {
		t.Fatal("expected error for build failure with no DB")
	}
	if !strings.Contains(err.Error(), "failed to build test database") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryHandler_BuildDBError_DBFileExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{
		name:       "test",
		buildDBErr: fmt.Errorf("some tests failed"),
		queryOut:   "partial results",
	}
	s := &testServer{be: be, root: tmpDir}

	// Rebuild requested but DB file will be created by side effect - simulate with pre-existing file
	// Actually, we need the file to exist so the fallback works
	// The handler checks fileExists after build fails, so create the file
	if err := os.WriteFile(filepath.Join(tmpDir, "testquery.db"), []byte("db"), 0644); err != nil {
		t.Fatal(err)
	}

	// Force rebuild, build fails but DB exists -> should proceed to query
	res, _, err := queryHandler(context.TODO(), nil, Params{Query: "SELECT 1", Dir: tmpDir, Rebuild: true}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(textContent(res), "partial results") {
		t.Errorf("expected partial results, got: %s", textContent(res))
	}
}

func TestQueryHandler_DefaultDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tq-test-*")
	if err != nil {
		t.Fatal(err)
	}
	//nolint:errcheck
	defer os.RemoveAll(tmpDir)

	roots.Global.Add(tmpDir)

	be := &mockBackend{name: "test", queryOut: "ok"}
	s := &testServer{be: be, root: tmpDir}

	// Empty dir should use project root
	res, _, err := queryHandler(context.TODO(), nil, Params{Query: "SELECT 1"}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = res
}
