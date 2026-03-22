// Package edit implements the edit_file tool.
package edit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/core/shared"
	"github.com/danicat/neko/internal/core/textdist"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server interface {
	ForFile(ctx context.Context, path string) backend.LanguageBackend
	IngestFile(ctx context.Context, path string, content string, symbols []lsp.DocumentSymbol, imports []string) error
	RAGEnabled() bool
	ProjectRoot() string
}

// Register registers the edit_file tool with the server.
func Register(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["edit_file"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return editHandler(ctx, req, args, s)
	})
}

// MultiRegister registers the multi_edit tool with the server.
func MultiRegister(mcpServer *mcp.Server, s Server) {
	def := toolnames.Registry["multi_edit"]
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args MultiParams) (*mcp.CallToolResult, any, error) {
		return multiEditHandler(ctx, req, args, s)
	})
}

// Params defines the input parameters for the edit_file tool.
type Params struct {
	File       string  `json:"file" jsonschema:"The path to the file to edit"`
	OldContent string  `json:"old_content,omitempty" jsonschema:"Optional: The block of code to find (ignores whitespace). Required if append is false."`
	NewContent string  `json:"new_content" jsonschema:"The new code to insert"`
	Threshold  float64 `json:"threshold,omitempty" jsonschema:"Similarity threshold (0.0-1.0) for fuzzy matching, default 0.95"`
	StartLine  int     `json:"start_line,omitempty" jsonschema:"Optional: restrict search to this line number and after"`
	EndLine    int     `json:"end_line,omitempty" jsonschema:"Optional: restrict search to this line number and before"`
	Append     bool    `json:"append,omitempty" jsonschema:"If true, append new_content to the end of the file (ignores old_content)"`
}

// MultiParams defines the input parameters for the multi_edit tool.
type MultiParams struct {
	Edits []Params `json:"edits" jsonschema:"List of files and their proposed edits"`
}

// MatchResult represents a potential match in the file.
type MatchResult struct {
	Start int
	End   int
	Score float64
}

func editHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, s Server) (*mcp.CallToolResult, any, error) {
	_, resSb, err := performEdit(ctx, args, s)
	if err != nil {
		return nil, nil, err
	}

	// For single edit, we pull diags immediately
	be := s.ForFile(ctx, args.File)
	var lspClient *lsp.Client
	if be != nil {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
			lspClient, _ = lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
		}
	}

	if lspClient != nil {
		if _, err := lspClient.WaitForDiagnostics(ctx, args.File); err != nil {
			return nil, nil, fmt.Errorf("LSP diagnostics wait failed: %w", err)
		}
		allDiags := lspClient.GetAllDiagnostics()

		workspaceRoot := s.ProjectRoot()
		if workspaceRoot == "" {
			workspaceRoot, _ = filepath.Abs(".")
		}
		resSb.WriteString(lsp.FormatDiagnostics(allDiags, workspaceRoot))
		lspClient.DidClose(ctx, args.File)
	} else {
		resSb.WriteString("\n*Note: LSP unavailable. Global semantic verification skipped.*")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resSb.String()}},
	}, nil, nil
}

func multiEditHandler(ctx context.Context, _ *mcp.CallToolRequest, args MultiParams, s Server) (*mcp.CallToolResult, any, error) {
	if len(args.Edits) == 0 {
		return nil, nil, fmt.Errorf("no edits provided")
	}

	var resSb strings.Builder
	resSb.WriteString("### Multi-Edit Report\n")

	affectedBackends := make(map[string]backend.LanguageBackend)

	for i, edit := range args.Edits {
		fmt.Fprintf(&resSb, "\n#### %d. %s\n", i+1, edit.File)
		_, editSb, err := performEdit(ctx, edit, s)
		if err != nil {
			fmt.Fprintf(&resSb, "❌ FAILED: %v\n", err)
			continue
		}
		resSb.WriteString(editSb.String())

		if be := s.ForFile(ctx, edit.File); be != nil {
			affectedBackends[be.Name()] = be
		}
	}

	// One global diagnostic pull at the end
	resSb.WriteString("\n---\n")
	anyLSP := false
	for _, be := range affectedBackends {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
			if lspClient, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions()); err == nil {
				anyLSP = true
				// We don't have a specific file to wait for, so we just pull current state
				if err := lspClient.PullDiagnostics(ctx); err != nil {
					return nil, nil, fmt.Errorf("LSP diagnostics pull failed: %w", err)
				}
				workspaceRoot := s.ProjectRoot()

				if workspaceRoot == "" {
					workspaceRoot, _ = filepath.Abs(".")
				}
				resSb.WriteString(lsp.FormatDiagnostics(lspClient.GetAllDiagnostics(), workspaceRoot))

				// Close all documents in this backend
				for _, edit := range args.Edits {
					if be2 := s.ForFile(ctx, edit.File); be2 != nil && be2.Name() == be.Name() {
						lspClient.DidClose(ctx, edit.File)
					}
				}
			}
		}
	}

	if !anyLSP {
		resSb.WriteString("\n*Note: No LSP active for affected files. Global semantic verification skipped.*")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resSb.String()}},
	}, nil, nil
}

func performEdit(ctx context.Context, args Params, s Server) (string, *strings.Builder, error) {
	var resSb strings.Builder
	var absPath string
	if args.File == "" || args.File == "." {
		absPath = s.ProjectRoot()
		if absPath == "" {
			absPath, _ = filepath.Abs(".")
		}
	} else {
		var err error
		absPath, err = filepath.Abs(args.File)
		if err != nil {
			return "", nil, err
		}
	}

	if err := roots.Global.Validate(absPath); err != nil {
		return "", nil, err
	}
	args.File = absPath

	// Strip NEKO tags from input
	args.OldContent = shared.StripNekoTags(args.OldContent)
	args.NewContent = shared.StripNekoTags(args.NewContent)

	// Threshold is hardcoded for safety
	args.Threshold = 0.95

	content, err := os.ReadFile(args.File)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	var newContent string
	original := string(content)

	searchStart := 0
	searchEnd := len(original)
	if args.StartLine > 0 || args.EndLine > 0 {
		s, e, err := shared.GetLineOffsets(original, args.StartLine, args.EndLine)
		if err != nil {
			return "", nil, err
		}
		searchStart = s
		searchEnd = e
	}

	if args.Append || args.OldContent == "" {
		if len(original) > 0 && !strings.HasSuffix(original, "\n") {
			newContent = original + "\n" + args.NewContent
		} else {
			newContent = original + args.NewContent
		}
	} else {
		searchArea := original[searchStart:searchEnd]
		matches := findMatches(searchArea, args.OldContent)

		var bestMatch MatchResult
		if len(matches) > 0 {
			bestMatch = matches[0]
		}

		if bestMatch.Score < args.Threshold {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("match not found with sufficient confidence (best score: %.2f < %.2f).\n\n", bestMatch.Score, args.Threshold))
			if len(matches) > 0 {
				sb.WriteString("Top Suggestions:\n")
				for i, m := range matches {
					if i >= 3 {
						break
					}
					matchText := searchArea[m.Start:m.End]
					globalStart := searchStart + m.Start
					globalEnd := searchStart + m.End
					startLine := shared.GetLineFromOffset(original, globalStart)
					endLine := shared.GetLineFromOffset(original, globalEnd)
					sb.WriteString(fmt.Sprintf("%d. (Score: %.2f) Lines %d-%d:\n```\n%s\n```\n", i+1, m.Score, startLine, endLine, matchText))
				}
			}
			sb.WriteString("\nSuggestions: verify your old_content or lower threshold.")
			return "", nil, fmt.Errorf("%s", sb.String())
		}

		matchStart := bestMatch.Start + searchStart
		matchEnd := bestMatch.End + searchStart
		newContent = original[:matchStart] + args.NewContent + original[matchEnd:]
	}

	be := s.ForFile(ctx, args.File)
	var lspClient *lsp.Client
	if be != nil {
		if cmd, cmdArgs, ok := be.LSPCommand(); ok {
			workspaceRoot := s.ProjectRoot()
			if workspaceRoot == "" {
				workspaceRoot, _ = filepath.Abs(".")
			}
			lspClient, _ = lsp.DefaultManager.ClientFor(ctx, be.Name(), workspaceRoot, cmd, cmdArgs, be.LanguageID(), be.InitializationOptions())
		}
	}

	if lspClient != nil {
		if err := lspClient.DidOpen(ctx, args.File, original); err != nil {
			return "", nil, fmt.Errorf("LSP open failed: %w", err)
		}
		if err := lspClient.DidChange(ctx, args.File, newContent); err != nil {
			return "", nil, fmt.Errorf("LSP change failed: %w", err)
		}

		if edits, err := lspClient.OrganizeImports(ctx, args.File); err == nil && len(edits) > 0 {
			newContent = lsp.ApplyTextEdits(newContent, edits)
			if err := lspClient.DidChange(ctx, args.File, newContent); err != nil {
				return "", nil, fmt.Errorf("LSP change failed after import organization: %w", err)
			}
		}
		if edits, err := lspClient.Format(ctx, args.File); err == nil && len(edits) > 0 {
			newContent = lsp.ApplyTextEdits(newContent, edits)
			if err := lspClient.DidChange(ctx, args.File, newContent); err != nil {
				return "", nil, fmt.Errorf("LSP change failed after formatting: %w", err)
			}
		}
	}

	// Direct Write
	//nolint:gosec // G306
	if err := os.WriteFile(args.File, []byte(newContent), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write file: %w", err)
	}

	resSb.WriteString(fmt.Sprintf("✅ Successfully edited %s\n", args.File))

	// Synchronous RAG Re-indexing
	if s.RAGEnabled() {
		var symbols []lsp.DocumentSymbol
		if lspClient != nil {
			symbols, _ = lspClient.DocumentSymbol(ctx, args.File)
		}
		var imports []string
		if be != nil {
			imports, _ = be.ParseImports(ctx, args.File)
		}
		if err := s.IngestFile(ctx, args.File, newContent, symbols, imports); err != nil {
			return "", nil, fmt.Errorf("RAG indexing failed: %w", err)
		}
	}

	if lspClient != nil {
		if err := lspClient.DidSave(ctx, args.File, newContent); err != nil {
			return "", nil, fmt.Errorf("LSP save failed: %w", err)
		}
	} else if be != nil {

		// Manual validation fallback for non-LSP backends or when LSP is down
		if err := be.Validate(ctx, args.File); err != nil {
			return "", nil, err
		}
	}
	return newContent, &resSb, nil
}

func findMatches(content, search string) []MatchResult {
	normSearch := normalize(search)
	if normSearch == "" {
		return nil
	}

	type charMap struct {
		char   rune
		offset int
	}
	var mapped []charMap
	for offset, char := range content {
		if !isWhitespace(char) {
			mapped = append(mapped, charMap{char, offset})
		}
	}
	normContentRunes := make([]rune, len(mapped))
	for i, cm := range mapped {
		normContentRunes[i] = cm.char
	}
	normContent := string(normContentRunes)

	if before, _, ok := strings.Cut(normContent, normSearch); ok {
		runeIdx := len([]rune(before))
		lastIdx := runeIdx + len([]rune(normSearch)) - 1
		if runeIdx >= len(mapped) || lastIdx >= len(mapped) {
			return nil
		}
		start := mapped[runeIdx].offset
		end := mapped[lastIdx].offset + len(string(mapped[lastIdx].char))
		return []MatchResult{{Start: start, End: end, Score: 1.0}}
	}

	searchRunes := []rune(normSearch)
	searchLen := len(searchRunes)
	contentLen := len(normContentRunes)

	if searchLen > contentLen {
		score := similarity(normSearch, normContent)
		return []MatchResult{{Start: 0, End: len(content), Score: score}}
	}

	seedLen := 16
	step := 8
	if searchLen < 64 {
		seedLen = 8
		step = 4
	}
	if searchLen < seedLen {
		seedLen = 4
		step = 2
	}

	candidates := make(map[int]int)
	checkSeed := func(offset int) {
		if offset < 0 || offset+seedLen > len(searchRunes) {
			return
		}
		seed := string(searchRunes[offset : offset+seedLen])
		startSearch := 0
		for {
			idx := strings.Index(normContent[startSearch:], seed)
			if idx == -1 {
				break
			}
			byteIdx := startSearch + idx
			// Convert byte offset to rune offset
			runeIdx := len([]rune(normContent[:byteIdx]))
			projectedStart := runeIdx - offset
			if projectedStart >= 0 && projectedStart <= len(normContentRunes)-searchLen {
				candidates[projectedStart]++
			}
			startSearch = byteIdx + len(seed)
		}
	}

	for i := 0; i <= searchLen-seedLen; i += step {
		checkSeed(i)
	}
	if searchLen >= seedLen {
		tailOffset := searchLen - seedLen
		if tailOffset%step != 0 {
			checkSeed(tailOffset)
		}
	}

	var results []MatchResult
	for startIdx := range candidates {
		if startIdx >= len(mapped) {
			continue
		}
		endIdx := min(startIdx+searchLen, len(normContentRunes))
		if endIdx <= 0 || endIdx > len(mapped) {
			continue
		}
		window := string(normContentRunes[startIdx:endIdx])
		score := similarity(normSearch, window)
		if score > 0.1 {
			start := mapped[startIdx].offset
			lastMapped := mapped[endIdx-1]
			end := lastMapped.offset + len(string(lastMapped.char))
			results = append(results, MatchResult{Start: start, End: end, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Start < results[j].Start
	})

	var filtered []MatchResult
	for _, r := range results {
		tooClose := false
		for _, f := range filtered {
			diff := r.Start - f.Start
			if diff < 0 {
				diff = -diff
			}
			if diff < 10 {
				tooClose = true
				break
			}
		}
		if !tooClose {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func normalize(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if !isWhitespace(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func similarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	d := textdist.Levenshtein(s1, s2)
	maxLen := len([]rune(s1))
	if l2 := len([]rune(s2)); l2 > maxLen {
		maxLen = l2
	}
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(d)/float64(maxLen)
}
