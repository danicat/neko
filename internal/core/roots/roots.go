package roots

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// State manages the registered project roots.
type State struct {
	mu    sync.RWMutex
	roots []string
}

// Global is the singleton instance for the entire application.
var Global = &State{}

// Add adds a new project root after normalizing it to an absolute path.
func (s *State) Add(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if slices.Contains(s.roots, abs) {
		return
	}
	s.roots = append(s.roots, abs)
}

// Get returns a copy of the currently registered roots.
func (s *State) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roots := make([]string, len(s.roots))
	copy(roots, s.roots)
	return roots
}

// Sync synchronizes roots with the MCP client.
func (s *State) Sync(ctx context.Context, session *mcp.ServerSession) {
	if session == nil {
		s.Add(".")
		return
	}

	res, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list roots from client: %v. Falling back to CWD.\n", err)
		s.Add(".")
		return
	}

	var rts []string
	for _, r := range res.Roots {
		if after, ok := strings.CutPrefix(r.URI, "file://"); ok {
			path := after
			abs, err := filepath.Abs(path)
			if err == nil {
				rts = append(rts, abs)
			}
		}
	}

	if len(rts) == 0 {
		abs, _ := filepath.Abs(".")
		rts = append(rts, abs)
	}

	s.mu.Lock()
	s.roots = rts
	s.mu.Unlock()
}

// Validate checks if the given path is within any of the registered roots.
func (s *State) Validate(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	roots := s.Get()

	rawTemp := os.TempDir()
	if strings.HasPrefix(absPath, rawTemp) {
		return nil
	}
	if tempDir, err := filepath.EvalSymlinks(rawTemp); err == nil {
		if strings.HasPrefix(absPath, tempDir) || strings.HasPrefix(absPath, "/tmp") {
			return nil
		}
	}

	if len(roots) == 0 {
		cwd, _ := filepath.Abs(".")
		if absPath == cwd || strings.HasPrefix(absPath, cwd+string(filepath.Separator)) {
			return nil
		}
		return fmt.Errorf("access denied: path %s is outside the current working directory", path)
	}

	for _, root := range roots {
		if absPath == root || strings.HasPrefix(absPath, root+string(filepath.Separator)) {
			return nil
		}
	}

	return fmt.Errorf("access denied: path %s is outside of registered workspace roots", path)
}
