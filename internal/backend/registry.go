package backend

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Registry manages multiple language backends and routes requests to the appropriate one.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]LanguageBackend
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]LanguageBackend),
	}
}

// Register adds a backend to the registry.
func (r *Registry) Register(be LanguageBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[be.Name()] = be
}

// ForFile returns the backend that handles the given file, based on extension.
// Returns nil if no backend matches.
func (r *Registry) ForFile(filename string) LanguageBackend {
	ext := strings.ToLower(filepath.Ext(filename))
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best LanguageBackend
	for _, be := range r.backends {
		for _, e := range be.FileExtensions() {
			if e == ext {
				if best == nil || be.Tier() > best.Tier() {
					best = be
				}
			}
		}
	}
	return best
}

// DetectBackends returns all backends that match the given directory based on project markers.
func (r *Registry) DetectBackends(dir string) []LanguageBackend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []LanguageBackend
	for _, be := range r.backends {
		for _, marker := range be.ProjectMarkers() {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				matched = append(matched, be)
				break
			}
		}
	}
	return matched
}

// Available returns the names of all registered backends.

func (r *Registry) Available() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.backends {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns a specific backend by name, or nil if not found.
func (r *Registry) Get(name string) LanguageBackend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends[name]
}

// AllSkipDirs returns the combined skip directories from all registered backends.
func (r *Registry) AllSkipDirs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var dirs []string
	for _, be := range r.backends {
		for _, d := range be.SkipDirs() {
			if !seen[d] {
				seen[d] = true
				dirs = append(dirs, d)
			}
		}
	}
	return dirs
}
