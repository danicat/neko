package lsp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// Manager manages LSP client instances, one per language per workspace root.
type Manager struct {
	mu      sync.Mutex
	clients map[string]*Client
}

// NewManager creates a new LSP manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// DefaultManager is the global LSP client manager.
var DefaultManager = NewManager()

// ClientFor returns an LSP client for the given language and workspace root.
// It creates and caches clients lazily.
func (m *Manager) ClientFor(ctx context.Context, lang, workspaceRoot, command string, args []string, langID string, options map[string]any) (*Client, error) {
	key := lang + ":" + workspaceRoot

	m.mu.Lock()
	if c, ok := m.clients[key]; ok {
		m.mu.Unlock()
		return c, nil
	}
	m.mu.Unlock()

	// Verify the command exists
	if _, err := exec.LookPath(command); err != nil {
		return nil, fmt.Errorf("LSP server %q not found in PATH", command)
	}

	c, err := NewClient(ctx, command, args, workspaceRoot, langID, options)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	// Double-check in case another goroutine created it
	if existing, ok := m.clients[key]; ok {
		m.mu.Unlock()
		c.Close()
		return existing, nil
	}
	m.clients[key] = c
	m.mu.Unlock()

	return c, nil
}

// CloseAll shuts down all managed LSP clients.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, c := range m.clients {
		c.Close()
		delete(m.clients, key)
	}
}
