package roots

import (
	"os"
	"path/filepath"
	"testing"
)

func newState() *State {
	return &State{}
}

func TestAdd_NormalizeToAbsolute(t *testing.T) {
	s := newState()
	s.Add(".")

	roots := s.Get()
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}

	abs, _ := filepath.Abs(".")
	if roots[0] != abs {
		t.Errorf("expected absolute path %s, got %s", abs, roots[0])
	}
}

func TestAdd_Deduplication(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-dedup-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)
	s.Add(tmpDir)

	roots := s.Get()
	if len(roots) != 1 {
		t.Fatalf("expected 1 root after duplicate add, got %d", len(roots))
	}
}

func TestAdd_MultipleRoots(t *testing.T) {
	s := newState()

	dir1, err := os.MkdirTemp("", "roots-a-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir1) //nolint:errcheck

	dir2, err := os.MkdirTemp("", "roots-b-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir2) //nolint:errcheck

	s.Add(dir1)
	s.Add(dir2)

	roots := s.Get()
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestGet_ReturnsCopy(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-copy-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)

	r1 := s.Get()
	r1[0] = "/mutated"

	r2 := s.Get()
	if r2[0] == "/mutated" {
		t.Error("Get() did not return a copy; mutation was visible")
	}
}

func TestGet_EmptyState(t *testing.T) {
	s := newState()
	roots := s.Get()
	if len(roots) != 0 {
		t.Fatalf("expected 0 roots on fresh state, got %d", len(roots))
	}
}

func TestValidate_AcceptsPathInsideRoot(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-val-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)

	innerPath := filepath.Join(tmpDir, "subdir", "file.go")
	if err := s.Validate(innerPath); err != nil {
		t.Errorf("expected path inside root to be accepted, got error: %v", err)
	}
}

func TestValidate_AcceptsExactRoot(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-exact-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)

	if err := s.Validate(tmpDir); err != nil {
		t.Errorf("expected exact root path to be accepted, got error: %v", err)
	}
}

func TestValidate_RejectsPathOutsideRoot(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)

	outsidePath := "/some/completely/different/path"
	if err := s.Validate(outsidePath); err == nil {
		t.Error("expected path outside root to be rejected, but got nil error")
	}
}

func TestValidate_AllowsTempDir(t *testing.T) {
	s := newState()

	tmpDir, err := os.MkdirTemp("", "roots-restrict-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	s.Add(tmpDir)

	// Temp directory paths should always be allowed.
	tempFile := filepath.Join(os.TempDir(), "somefile.txt")
	if err := s.Validate(tempFile); err != nil {
		t.Errorf("expected temp dir path to be accepted, got error: %v", err)
	}
}

func TestValidate_EmptyRootsFallsBackToCwd(t *testing.T) {
	s := newState()

	// With no roots, Validate falls back to CWD.
	cwd, _ := filepath.Abs(".")
	cwdFile := filepath.Join(cwd, "somefile.go")

	if err := s.Validate(cwdFile); err != nil {
		t.Errorf("expected CWD-relative path to be accepted with no roots, got error: %v", err)
	}

	// A path outside CWD should be rejected.
	outsidePath := "/some/random/other/place"
	if err := s.Validate(outsidePath); err == nil {
		t.Error("expected path outside CWD to be rejected when no roots are set")
	}
}

func TestValidate_MultipleRoots(t *testing.T) {
	s := newState()

	dir1, err := os.MkdirTemp("", "roots-m1-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir1) //nolint:errcheck

	dir2, err := os.MkdirTemp("", "roots-m2-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir2) //nolint:errcheck

	s.Add(dir1)
	s.Add(dir2)

	// A file in either root should be accepted.
	if err := s.Validate(filepath.Join(dir1, "a.go")); err != nil {
		t.Errorf("expected path in root1 to be accepted: %v", err)
	}
	if err := s.Validate(filepath.Join(dir2, "b.go")); err != nil {
		t.Errorf("expected path in root2 to be accepted: %v", err)
	}

	// A path in neither root should be rejected.
	if err := s.Validate("/nowhere/file.go"); err == nil {
		t.Error("expected path outside all roots to be rejected")
	}
}
