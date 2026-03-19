package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client is a minimal LSP client that communicates with a language server over stdio.
type Client struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	reader   *bufio.Reader
	mu       sync.Mutex
	nextID   atomic.Int64
	pending  map[int64]chan *jsonrpcResponse
	opened   map[string]int // URIs of opened documents → version
	rootURI  string
	rootPath string
	langID   string
	options  map[string]any

	done      chan struct{}
	closeOnce sync.Once
}

// NewClient starts an LSP server and initializes the connection.
// langID is the LSP language identifier (e.g. "go", "python", "javascript").
func NewClient(ctx context.Context, command string, args []string, workspaceRoot string, langID string, options map[string]any) (*Client, error) {
	// We don't use exec.CommandContext(ctx) here because the LSP server should
	// outlive the context of the request that triggered its start.
	// The client is closed explicitly via Close().
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server %q: %w", command, err)
	}

	absRoot, _ := filepath.Abs(workspaceRoot)

	c := &Client{
		cmd:      cmd,
		stdin:    stdin,
		reader:   bufio.NewReader(stdout),
		pending:  make(map[int64]chan *jsonrpcResponse),
		opened:   make(map[string]int),
		rootURI:  fileURI(absRoot),
		rootPath: absRoot,
		langID:   langID,
		options:  options,

		done: make(chan struct{}),
	}

	go c.readLoop()

	if err := c.initialize(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("LSP initialization failed: %w", err)
	}

	return c, nil
}

// Close shuts down the LSP server gracefully. Safe to call multiple times.
func (c *Client) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Send shutdown request
		c.call(ctx, "shutdown", nil)
		// Send exit notification
		c.notify("exit", nil)

		c.stdin.Close()
		close(c.done)
		closeErr = c.cmd.Wait()
	})
	return closeErr
}

// Hover returns hover information for a position in a file.
func (c *Client) Hover(ctx context.Context, file string, line, col int) (*Hover, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line - 1, Character: col - 1},
	}

	result, err := c.call(ctx, "textDocument/hover", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, fmt.Errorf("no hover information at %s:%d:%d", file, line, col)
	}

	var hover Hover
	if err := json.Unmarshal(result, &hover); err != nil {
		return nil, fmt.Errorf("failed to parse hover result: %w", err)
	}
	return &hover, nil
}

// Definition returns the definition location(s) for a symbol.
func (c *Client) Definition(ctx context.Context, file string, line, col int) ([]Location, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line - 1, Character: col - 1},
	}

	result, err := c.call(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, fmt.Errorf("no definition found at %s:%d:%d", file, line, col)
	}

	// Definition can return a single Location or []Location
	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		var single Location
		if err2 := json.Unmarshal(result, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse definition result: %w", err)
		}
		locations = []Location{single}
	}
	return locations, nil
}

// References returns all references to a symbol.
func (c *Client) References(ctx context.Context, file string, line, col int, includeDecl bool) ([]Location, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line - 1, Character: col - 1},
		Context:      ReferenceContext{IncludeDeclaration: includeDecl},
	}

	result, err := c.call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, fmt.Errorf("no references found at %s:%d:%d", file, line, col)
	}

	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, fmt.Errorf("failed to parse references result: %w", err)
	}
	return locations, nil
}

// HoverText extracts a human-readable string from a Hover result.
func HoverText(h *Hover) string {
	// contents can be: string, MarkupContent, MarkedString, or []MarkedString
	var mc MarkupContent
	if err := json.Unmarshal(h.Contents, &mc); err == nil && mc.Value != "" {
		return mc.Value
	}

	var s string
	if err := json.Unmarshal(h.Contents, &s); err == nil {
		return s
	}

	// MarkedString: {language, value}
	var ms struct {
		Language string `json:"language"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(h.Contents, &ms); err == nil && ms.Value != "" {
		return ms.Value
	}

	// Array of MarkedString
	var arr []json.RawMessage
	if err := json.Unmarshal(h.Contents, &arr); err == nil {
		var parts []string
		for _, item := range arr {
			var mc2 MarkupContent
			if json.Unmarshal(item, &mc2) == nil && mc2.Value != "" {
				parts = append(parts, mc2.Value)
				continue
			}
			var s2 string
			if json.Unmarshal(item, &s2) == nil {
				parts = append(parts, s2)
				continue
			}
			var ms2 struct {
				Language string `json:"language"`
				Value    string `json:"value"`
			}
			if json.Unmarshal(item, &ms2) == nil && ms2.Value != "" {
				parts = append(parts, ms2.Value)
			}
		}
		return strings.Join(parts, "\n\n")
	}

	return string(h.Contents)
}

// FormatLocations formats locations as a human-readable string.
func FormatLocations(locations []Location) string {
	var sb strings.Builder
	for _, loc := range locations {
		path := uriToPath(loc.URI)
		sb.WriteString(fmt.Sprintf("- %s:%d:%d\n",
			path,
			loc.Range.Start.Line+1,
			loc.Range.Start.Character+1,
		))
	}
	return sb.String()
}

// --- internal ---

func (c *Client) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootPath:  c.rootPath,
		RootURI:   c.rootURI,
		ClientInfo: &ClientInfo{
			Name:    "neko",
			Version: "0.1.0",
		},
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Hover: &HoverClientCapabilities{
					ContentFormat: []string{"markdown", "plaintext"},
				},
				Definition: &DefinitionClientCapabilities{
					LinkSupport: true,
				},
			},
		},
		InitializationOptions: c.options,
		WorkspaceFolders: []WorkspaceFolder{
			{
				URI:  c.rootURI,
				Name: filepath.Base(c.rootPath),
			},
		},
	}

	result, err := c.call(ctx, "initialize", params)

	if err != nil {
		return err
	}

	var initResult InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	return c.notify("initialized", struct{}{})
}

func (c *Client) ensureOpen(ctx context.Context, file string) (string, error) {
	absPath, _ := filepath.Abs(file)
	uri := fileURI(absPath)

	//nolint:gosec // G304
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", file, err)
	}

	c.mu.Lock()
	version, alreadyOpen := c.opened[uri]
	c.mu.Unlock()

	if alreadyOpen {
		// Re-send content in case the file was modified since last open
		newVersion := version + 1
		err = c.notify("textDocument/didChange", DidChangeTextDocumentParams{
			TextDocument: VersionedTextDocumentIdentifier{
				URI:     uri,
				Version: newVersion,
			},
			ContentChanges: []TextDocumentContentChangeEvent{
				{Text: string(content)},
			},
		})
		if err != nil {
			return "", err
		}
		c.mu.Lock()
		c.opened[uri] = newVersion
		c.mu.Unlock()
		return uri, nil
	}

	langID := c.langID
	if langID == "" {
		langID = detectLanguageID(absPath)
	}

	err = c.notify("textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: langID,
			Version:    1,
			Text:       string(content),
		},
	})
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.opened[uri] = 1
	c.mu.Unlock()

	// Give the server a moment to index the file on first open
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	return uri, nil
}

func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	ch := make(chan *jsonrpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("LSP server closed")
	}
}

func (c *Client) notify(method string, params any) error {
	msg := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.send(msg)
}

func (c *Client) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(data)
	return err
}

func (c *Client) readLoop() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			return
		}

		var base struct {
			ID     *json.RawMessage `json:"id"`
			Method string           `json:"method"`
		}
		if err := json.Unmarshal(msg, &base); err != nil {
			continue
		}

		// Response (has id, no method)
		if base.ID != nil && base.Method == "" {
			var id int64
			if err := json.Unmarshal(*base.ID, &id); err != nil {
				// Try as string (some servers do this)
				var idStr string
				if err := json.Unmarshal(*base.ID, &idStr); err == nil {
					id, _ = strconv.ParseInt(idStr, 10, 64)
				}
			}
			c.mu.Lock()
			ch := c.pending[id]
			delete(c.pending, id)
			c.mu.Unlock()
			if ch != nil {
				var resp jsonrpcResponse
				json.Unmarshal(msg, &resp)
				ch <- &resp
			}
		}
		// Notifications and server requests are silently consumed
	}
}

func (c *Client) readMessage() (json.RawMessage, error) {
	var contentLength int
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			fmt.Sscanf(line, "Content-Length: %d", &contentLength)
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func fileURI(path string) string {
	absPath, _ := filepath.Abs(path)
	u := &url.URL{
		Scheme: "file",
		Path:   absPath,
	}
	return u.String()
}

func uriToPath(uri string) string {
	if after, ok := strings.CutPrefix(uri, "file://"); ok {
		path := after
		if decoded, err := url.PathUnescape(path); err == nil {
			return decoded
		}
		return path
	}
	return uri
}

func detectLanguageID(path string) string {
	switch filepath.Ext(path) {
	case ".go":
		return "go"
	case ".py", ".pyi":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}
