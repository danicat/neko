package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client is a minimal LSP client that communicates with a language server over stdio.
type Client struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	reader     *bufio.Reader
	mu         sync.Mutex
	nextID     atomic.Int64
	pending    map[int64]chan *jsonrpcResponse
	openedDocs map[string]int // URIs of opened documents → version

	diagMu      sync.Mutex
	diagnostics map[string][]Diagnostic  // URI -> current diagnostics
	diagWatch   map[string]chan struct{} // URI -> signal channel

	capabilities ServerCapabilities
	rootURI      string
	rootPath     string
	langID       string
	options      map[string]any

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
		cmd:         cmd,
		stdin:       stdin,
		reader:      bufio.NewReader(stdout),
		pending:     make(map[int64]chan *jsonrpcResponse),
		openedDocs:  make(map[string]int),
		diagnostics: make(map[string][]Diagnostic),
		diagWatch:   make(map[string]chan struct{}),
		rootURI:     FileURI(absRoot),
		rootPath:    absRoot,
		langID:      langID,
		options:     options,

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

// GetVersion returns the current version of the document.
func (c *Client) GetVersion(file string) int {
	uri := FileURI(file)
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.openedDocs[uri]
}

// DidOpen sends a textDocument/didOpen notification.
func (c *Client) DidOpen(ctx context.Context, file, content string) error {
	absPath, _ := filepath.Abs(file)
	uri := FileURI(absPath)
	langID := c.langID
	if langID == "" {
		langID = detectLanguageID(absPath)
	}

	err := c.notify("textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: langID,
			Version:    1,
			Text:       content,
		},
	})
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.openedDocs[uri] = 1
	c.mu.Unlock()
	return nil
}

// DidChange sends a textDocument/didChange notification.
func (c *Client) DidChange(ctx context.Context, file, content string) error {
	uri := FileURI(file)
	c.mu.Lock()
	version := c.openedDocs[uri]
	newVersion := version + 1
	c.openedDocs[uri] = newVersion
	c.mu.Unlock()

	return c.notify("textDocument/didChange", DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: newVersion,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: content},
		},
	})
}

// DidSave sends a textDocument/didSave notification.
func (c *Client) DidSave(ctx context.Context, file, content string) error {
	uri := FileURI(file)
	c.mu.Lock()
	version := c.openedDocs[uri]
	newVersion := version + 1
	c.openedDocs[uri] = newVersion
	c.mu.Unlock()

	params := DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	if content != "" {
		params.Text = &content
	}

	return c.notify("textDocument/didSave", params)
}

// DidClose sends a textDocument/didClose notification.
func (c *Client) DidClose(ctx context.Context, file string) error {
	uri := FileURI(file)
	c.mu.Lock()
	delete(c.openedDocs, uri)
	c.mu.Unlock()

	return c.notify("textDocument/didClose", DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

// DidChangeWatchedFiles sends a workspace/didChangeWatchedFiles notification.
func (c *Client) DidChangeWatchedFiles(ctx context.Context, file string, changeType int) error {
	uri := FileURI(file)
	return c.notify("workspace/didChangeWatchedFiles", DidChangeWatchedFilesParams{
		Changes: []FileEvent{
			{URI: uri, Type: changeType},
		},
	})
}

// PullDiagnostics requests workspace-wide diagnostics.
func (c *Client) PullDiagnostics(ctx context.Context) error {
	c.mu.Lock()
	supported := c.capabilities.DiagnosticProvider != nil
	c.mu.Unlock()

	if !supported {
		return fmt.Errorf("workspace/diagnostic not supported by server")
	}

	result, err := c.call(ctx, "workspace/diagnostic", WorkspaceDiagnosticParams{})
	if err != nil {
		return err
	}

	var report WorkspaceDiagnosticReport
	if err := json.Unmarshal(result, &report); err != nil {
		return fmt.Errorf("failed to parse diagnostic report: %w", err)
	}

	c.diagMu.Lock()
	defer c.diagMu.Unlock()
	for _, item := range report.Items {
		if item.Kind == "full" {
			c.diagnostics[item.URI] = item.Items
		}
	}
	return nil
}

// WaitForDiagnostics blocks until diagnostics for the given file are updated.
func (c *Client) WaitForDiagnostics(ctx context.Context, file string) ([]Diagnostic, error) {
	absPath, _ := filepath.Abs(file)
	uri := FileURI(absPath)

	// Try pull model first if supported
	c.mu.Lock()
	supported := c.capabilities.DiagnosticProvider != nil
	c.mu.Unlock()

	if supported {
		// For servers that support pull, we just pull.
		// Some servers might need a moment to re-index, but protocol says pull should be consistent.
		err := c.PullDiagnostics(ctx)
		if err == nil {
			c.diagMu.Lock()
			diags := c.diagnostics[uri]
			c.diagMu.Unlock()
			return diags, nil
		}
	}

	// Fallback to push model (waiting for notification)
	c.diagMu.Lock()
	ch := make(chan struct{})
	c.diagWatch[uri] = ch
	c.diagMu.Unlock()

	select {
	case <-ch:
		c.diagMu.Lock()
		diags := c.diagnostics[uri]
		c.diagMu.Unlock()
		return diags, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		c.diagMu.Lock()
		delete(c.diagWatch, uri)
		diags := c.diagnostics[uri]
		c.diagMu.Unlock()
		return diags, nil // Return current cache on timeout
	}
}

// GetAllDiagnostics returns all cached diagnostics.
func (c *Client) GetAllDiagnostics() map[string][]Diagnostic {
	c.diagMu.Lock()
	defer c.diagMu.Unlock()
	// Return a copy
	res := make(map[string][]Diagnostic)
	maps.Copy(res, c.diagnostics)
	return res
}

// FormatDiagnostics formats a map of diagnostics into a standardized Markdown report.
func FormatDiagnostics(diagnostics map[string][]Diagnostic) string {
	var sb strings.Builder
	total := 0
	filesWithIssues := make(map[string]bool)

	// Sort URIs for deterministic output
	var uris []string
	for uri := range diagnostics {
		uris = append(uris, uri)
	}
	sort.Strings(uris)

	sb.WriteString("\n🔍 **Current Project Health:**\n")

	for _, uri := range uris {
		diags := diagnostics[uri]
		if len(diags) == 0 {
			continue
		}

		path := URIToPath(uri)
		filesWithIssues[path] = true

		for _, d := range diags {
			total++
			severity := "Error"
			if d.Severity == 2 {
				severity = "Warning"
			}
			fmt.Fprintf(&sb, "- `%s:%d:%d`: [%s] %s\n", path, d.Range.Start.Line+1, d.Range.Start.Character+1, severity, d.Message)
		}
	}

	if total == 0 {
		return "\n✅ **Project is clean!** No errors or warnings found."
	}

	fmt.Fprintf(&sb, "\n*Total: %d issues found across %d files.*", total, len(filesWithIssues))
	return sb.String()
}

// Format requests document formatting.
func (c *Client) Format(ctx context.Context, file string) ([]TextEdit, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options: FormattingOptions{
			TabSize:      4,
			InsertSpaces: false,
		},
	}

	result, err := c.call(ctx, "textDocument/formatting", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var edits []TextEdit
	if err := json.Unmarshal(result, &edits); err != nil {
		return nil, fmt.Errorf("failed to parse formatting edits: %w", err)
	}
	return edits, nil
}

// OrganizeImports requests the organize imports code action.
func (c *Client) OrganizeImports(ctx context.Context, file string) ([]TextEdit, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 100000, Character: 0},
		},
		Context: CodeActionContext{
			Only: []string{"source.organizeImports"},
		},
	}

	result, err := c.call(ctx, "textDocument/codeAction", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var actions []CodeAction
	if err := json.Unmarshal(result, &actions); err != nil {
		return nil, fmt.Errorf("failed to parse code actions: %w", err)
	}

	for _, action := range actions {
		if action.Edit != nil && action.Edit.Changes != nil {
			if edits, ok := action.Edit.Changes[uri]; ok {
				return edits, nil
			}
		}
	}

	return nil, nil
}

// Rename requests a symbol rename.
func (c *Client) Rename(ctx context.Context, file string, line, col int, newName string) (*WorkspaceEdit, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := RenameParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line - 1, Character: col - 1},
		NewName:      newName,
	}

	result, err := c.call(ctx, "textDocument/rename", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var edit WorkspaceEdit
	if err := json.Unmarshal(result, &edit); err != nil {
		return nil, fmt.Errorf("failed to parse rename result: %w", err)
	}
	return &edit, nil
}

// DocumentSymbol requests hierarchical symbols for a document.
func (c *Client) DocumentSymbol(ctx context.Context, file string) ([]DocumentSymbol, error) {
	uri, err := c.ensureOpen(ctx, file)
	if err != nil {
		return nil, err
	}

	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	result, err := c.call(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}
	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var symbols []DocumentSymbol
	if err := json.Unmarshal(result, &symbols); err != nil {
		// Some servers return []SymbolInformation, but we prioritize DocumentSymbol
		return nil, fmt.Errorf("failed to parse document symbols: %w", err)
	}
	return symbols, nil
}

// ApplyTextEdits applies a list of TextEdits to the given content.
func ApplyTextEdits(content string, edits []TextEdit) string {
	if len(edits) == 0 {
		return content
	}

	// Sort edits in reverse order (bottom to top)
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Range.Start.Line != edits[j].Range.Start.Line {
			return edits[i].Range.Start.Line > edits[j].Range.Start.Line
		}
		return edits[i].Range.Start.Character > edits[j].Range.Start.Character
	})

	lines := strings.Split(content, "\n")
	for _, edit := range edits {
		startLine := edit.Range.Start.Line
		startCol := edit.Range.Start.Character
		endLine := edit.Range.End.Line
		endCol := edit.Range.End.Character

		if startLine >= len(lines) || endLine >= len(lines) {
			continue
		}

		// Bounds checks for columns
		if startCol > len(lines[startLine]) {
			startCol = len(lines[startLine])
		}
		if endCol > len(lines[endLine]) {
			endCol = len(lines[endLine])
		}

		// Reconstruct the affected lines
		prefix := lines[startLine][:startCol]
		suffix := lines[endLine][endCol:]

		newTextLines := strings.Split(edit.NewText, "\n")
		firstLine := prefix + newTextLines[0]
		lastLine := newTextLines[len(newTextLines)-1] + suffix

		var midLines []string
		if len(newTextLines) > 1 {
			midLines = newTextLines[1 : len(newTextLines)-1]
			// If there's only 2 lines, midLines will be empty, which is correct
		}
		if len(newTextLines) == 1 {
			// Edit is on a single line or replaces multiple lines with one
			firstLine = prefix + edit.NewText + suffix
			midLines = nil
			lastLine = "" // Not used
		}

		// Splice into lines slice
		var resultLines []string
		resultLines = append(resultLines, lines[:startLine]...)
		resultLines = append(resultLines, firstLine)
		if len(newTextLines) > 1 {
			resultLines = append(resultLines, midLines...)
			resultLines = append(resultLines, lastLine)
		}
		resultLines = append(resultLines, lines[endLine+1:]...)
		lines = resultLines
	}

	return strings.Join(lines, "\n")
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

// FormatSymbols formats a list of document symbols into a concise outline.
func FormatSymbols(symbols []DocumentSymbol) string {
	var sb strings.Builder
	formatSymbolRecursive(&sb, symbols, 0)
	return sb.String()
}

func formatSymbolRecursive(sb *strings.Builder, symbols []DocumentSymbol, depth int) {
	for _, s := range symbols {
		// Only show high-level interesting symbols in the outline
		// 1: File, 2: Module, 3: Namespace, 4: Package, 5: Class, 6: Method, 7: Property, 8: Field, 9: Constructor, 10: Enum, 11: Interface, 12: Function, 13: Variable, 14: Constant, 15: String, 16: Number, 17: Boolean, 18: Array, 19: Object, 20: Key, 21: Null, 22: EnumMember, 23: Struct, 24: Event, 25: Operator, 26: TypeParameter
		if s.Kind == 5 || s.Kind == 6 || s.Kind == 11 || s.Kind == 12 || s.Kind == 23 {
			indent := strings.Repeat("  ", depth)
			detail := ""
			if s.Detail != "" {
				detail = " " + s.Detail
			}
			fmt.Fprintf(sb, "%s- %s%s (Lines %d-%d)\n", indent, s.Name, detail, s.Range.Start.Line+1, s.Range.End.Line+1)
			if len(s.Children) > 0 {
				formatSymbolRecursive(sb, s.Children, depth+1)
			}
		}
	}
}

// FormatLocations formats locations as a human-readable string.
func FormatLocations(locations []Location) string {
	var sb strings.Builder
	for _, loc := range locations {
		path := URIToPath(loc.URI)
		fmt.Fprintf(&sb, "- %s:%d:%d\n", path, loc.Range.Start.Line+1, loc.Range.Start.Character+1)
	}
	return sb.String()
}

// EnrichLocations adds context (containing symbol name) to a list of locations.
func (c *Client) EnrichLocations(ctx context.Context, locations []Location) string {
	var source []string
	var tests []string

	for _, loc := range locations {
		path := URIToPath(loc.URI)
		symbol, _ := c.GetSymbolAt(ctx, path, loc.Range.Start.Line+1, loc.Range.Start.Character+1)

		context := ""
		if symbol != "" {
			context = fmt.Sprintf(" (in '%s')", symbol)
		}

		entry := fmt.Sprintf("- %s:%d:%d%s", path, loc.Range.Start.Line+1, loc.Range.Start.Character+1, context)

		if strings.Contains(path, "_test.go") || strings.Contains(filepath.Base(path), "test_") {
			tests = append(tests, entry)
		} else {
			source = append(source, entry)
		}
	}

	var sb strings.Builder
	if len(source) > 0 {
		sb.WriteString("[SOURCE]\n")
		for _, s := range source {
			sb.WriteString(s + "\n")
		}
	}
	if len(tests) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[TESTS]\n")
		for _, t := range tests {
			sb.WriteString(t + "\n")
		}
	}

	return sb.String()
}

// GetSymbolAt returns the name of the symbol containing the given position.
func (c *Client) GetSymbolAt(ctx context.Context, file string, line, col int) (string, error) {
	// Simple implementation: use hover to get symbol context if possible,
	// or we could use documentSymbol and find the range. Hover is often enough.
	hover, err := c.Hover(ctx, file, line, col)
	if err != nil {
		return "", err
	}
	// Hover text often contains the signature, e.g. "func (*Server).establishProject"
	// We'll try to extract a clean name.
	text := HoverText(hover)
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		// Clean up common markdown formatting from LSP
		firstLine = strings.Trim(firstLine, "`")
		return firstLine, nil
	}
	return "", nil
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
				DocumentSymbol: &DocumentSymbolClientCapabilities{
					HierarchicalDocumentSymbolSupport: true,
				},
				Diagnostic: &struct{}{},
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

	c.mu.Lock()
	c.capabilities = initResult.Capabilities
	c.mu.Unlock()

	return c.notify("initialized", struct{}{})
}

func (c *Client) ensureOpen(ctx context.Context, file string) (string, error) {
	absPath, _ := filepath.Abs(file)
	uri := FileURI(absPath)

	//nolint:gosec // G304
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", file, err)
	}

	c.mu.Lock()
	_, alreadyOpen := c.openedDocs[uri]
	c.mu.Unlock()

	if alreadyOpen {
		// Re-send content in case the file was modified since last open
		err = c.DidChange(ctx, absPath, string(content))
		if err != nil {
			return "", err
		}
		return uri, nil
	}

	err = c.DidOpen(ctx, absPath, string(content))
	if err != nil {
		return "", err
	}

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

		// Notification
		if base.Method == "textDocument/publishDiagnostics" {
			var notify jsonrpcNotification
			if err := json.Unmarshal(msg, &notify); err == nil {
				var params PublishDiagnosticsParams
				data, _ := json.Marshal(notify.Params)
				if err := json.Unmarshal(data, &params); err == nil {
					c.diagMu.Lock()
					c.diagnostics[params.URI] = params.Diagnostics
					if watch, ok := c.diagWatch[params.URI]; ok {
						close(watch)
						delete(c.diagWatch, params.URI)
					}
					c.diagMu.Unlock()
				}
			}
		}
		// Other notifications and server requests are silently consumed
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

func FileURI(path string) string {
	absPath, _ := filepath.Abs(path)
	u := &url.URL{
		Scheme: "file",
		Path:   absPath,
	}
	return u.String()
}

func URIToPath(uri string) string {
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

// EnhancedHover performs a hover, and if the result is a simple variable declaration,
// it attempts to jump to the definition of that variable's type to fetch the full struct/interface
// documentation and method set. It returns the combined markdown.
func (c *Client) EnhancedHover(ctx context.Context, file string, line, col int) (string, error) {
	hover, err := c.Hover(ctx, file, line, col)
	if err != nil {
		return "", err
	}

	text := HoverText(hover)

	// If the hover is a simple variable or function signature, try to find the definition
	// to get richer type/struct information.
	if len(text) < 200 {
		locs, defErr := c.Definition(ctx, file, line, col)
		if defErr == nil && len(locs) > 0 {
			defLoc := locs[0]
			defPath := URIToPath(defLoc.URI)
			
			// Only jump if it's a different location or if we are forced to (depth 1)
			defHover, hErr := c.Hover(ctx, defPath, defLoc.Range.Start.Line+1, defLoc.Range.Start.Character+1)
			if hErr == nil {
				defText := HoverText(defHover)
				if len(defText) > len(text) {
					// Prepend the original signature if it's a variable
					if strings.HasPrefix(text, "```go\nvar") {
						return text + "\n\n" + defText, nil
					}
					return defText, nil
				}
			}
		}
	}

	return text, nil
}
