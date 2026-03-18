// Package edit implements the smart_edit tool.
package edit

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/core/shared"
	"github.com/danicat/neko/internal/core/textdist"
	"github.com/danicat/neko/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the smart_edit tool with the server.
func Register(server *mcp.Server, reg *backend.Registry) {
	def := toolnames.Registry["smart_edit"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
		return editHandler(ctx, req, args, reg)
	})
}

// Params defines the input parameters for the smart_edit tool.
type Params struct {
	Filename   string  `json:"filename" jsonschema:"The path to the file to edit"`
	OldContent string  `json:"old_content,omitempty" jsonschema:"Optional: The block of code to find (ignores whitespace). Required if append is false."`
	NewContent string  `json:"new_content" jsonschema:"The new code to insert"`
	Threshold  float64 `json:"threshold,omitempty" jsonschema:"Similarity threshold (0.0-1.0) for fuzzy matching, default 0.95"`
	StartLine  int     `json:"start_line,omitempty" jsonschema:"Optional: restrict search to this line number and after"`
	EndLine    int     `json:"end_line,omitempty" jsonschema:"Optional: restrict search to this line number and before"`
	Append     bool    `json:"append,omitempty" jsonschema:"If true, append new_content to the end of the file (ignores old_content)"`
}

func editHandler(ctx context.Context, _ *mcp.CallToolRequest, args Params, reg *backend.Registry) (*mcp.CallToolResult, any, error) {
	absPath, err := roots.Global.Validate(args.Filename)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}
	args.Filename = absPath

	if args.Threshold == 0 {
		args.Threshold = 0.95
	}
	if args.Threshold > 1.0 {
		args.Threshold = 1.0
	}
	if args.Threshold < 0.0 {
		args.Threshold = 0.0
	}

	content, err := os.ReadFile(args.Filename)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), nil, nil
	}

	var newContent string
	original := string(content)

	searchStart := 0
	searchEnd := len(original)
	if args.StartLine > 0 || args.EndLine > 0 {
		s, e, err := shared.GetLineOffsets(original, args.StartLine, args.EndLine)
		if err != nil {
			return errorResult(fmt.Sprintf("line range error: %v", err)), nil, nil
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
		matchStart, matchEnd, score := findBestMatch(searchArea, args.OldContent)

		if score < args.Threshold {
			bestMatch := ""
			if matchStart < matchEnd && matchEnd <= len(searchArea) {
				bestMatch = searchArea[matchStart:matchEnd]
			}
			globalMatchStart := searchStart + matchStart
			globalMatchEnd := searchStart + matchEnd
			bestStartLine := shared.GetLineFromOffset(original, globalMatchStart)
			bestEndLine := shared.GetLineFromOffset(original, globalMatchEnd)

			return errorResult(fmt.Sprintf("match not found with sufficient confidence (score: %.2f < %.2f).\n\nBest Match Found (Lines %d-%d):\n```\n%s\n```\n\nSuggestions: verify your old_content or lower threshold.", score, args.Threshold, bestStartLine, bestEndLine, bestMatch)), nil, nil
		}

		matchStart += searchStart
		matchEnd += searchStart
		newContent = original[:matchStart] + args.NewContent + original[matchEnd:]
	}

	// Validate & Format using the appropriate backend
	be := reg.ForFile(args.Filename)
	var warning string
	if be != nil {
		//nolint:gosec // G306
		if err := os.WriteFile(args.Filename, []byte(newContent), 0644); err != nil {
			return errorResult(fmt.Sprintf("failed to write file: %v", err)), nil, nil
		}

		if err := be.Validate(ctx, args.Filename); err != nil {
			if restoreErr := os.WriteFile(args.Filename, content, 0644); restoreErr != nil {
				return errorResult(fmt.Sprintf("edit produced invalid code AND failed to restore original: %v (restore: %v)", err, restoreErr)), nil, nil
			}
			snippet := shared.ExtractErrorSnippet(newContent, err)
			return errorResult(fmt.Sprintf("edit produced invalid code: %v\n\nContext:\n```\n%s\n```\nHint: Ensure your new_content is syntactically valid in context.", err, snippet)), nil, nil
		}

		if fmtErr := be.Format(ctx, args.Filename); fmtErr != nil {
			warning = fmt.Sprintf("\n\n**WARNING:** formatting failed: %v", fmtErr)
		}

		formatted, err := os.ReadFile(args.Filename)
		if err == nil {
			newContent = string(formatted)
		}
	}

	//nolint:gosec // G306
	if err := os.WriteFile(args.Filename, []byte(newContent), 0644); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), nil, nil
	}

	if be != nil {
		if err := be.Validate(ctx, args.Filename); err != nil {
			warning += fmt.Sprintf("\n\n**WARNING:** Post-edit syntax check failed: %v", err)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Successfully edited %s%s", args.Filename, warning)},
		},
	}, nil, nil
}

func findBestMatch(content, search string) (int, int, float64) {
	normSearch := normalize(search)
	if normSearch == "" {
		return 0, 0, 0
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

	if idx := strings.Index(normContent, normSearch); idx != -1 {
		runeIdx := len([]rune(normContent[:idx]))
		start := mapped[runeIdx].offset
		end := mapped[runeIdx+len([]rune(normSearch))-1].offset + 1
		return start, end, 1.0
	}

	searchRunes := []rune(normSearch)
	searchLen := len(searchRunes)
	contentLen := len(normContentRunes)

	if searchLen > contentLen {
		score := similarity(normSearch, normContent)
		return 0, len(content), score
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
		seed := string(searchRunes[offset : offset+seedLen])
		startSearch := 0
		for {
			idx := strings.Index(normContent[startSearch:], seed)
			if idx == -1 {
				break
			}
			realIdx := startSearch + idx
			projectedStart := realIdx - offset
			if projectedStart >= 0 && projectedStart <= len(normContentRunes)-searchLen {
				candidates[projectedStart]++
			}
			startSearch = realIdx + 1
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

	bestScore := 0.0
	bestStartIdx := 0
	bestEndIdx := 0

	for startIdx := range candidates {
		endIdx := startIdx + searchLen
		if endIdx > len(normContentRunes) {
			endIdx = len(normContentRunes)
		}
		window := string(normContentRunes[startIdx:endIdx])
		score := similarity(normSearch, window)
		if score > bestScore {
			bestScore = score
			bestStartIdx = startIdx
			bestEndIdx = endIdx
		}
	}

	if bestScore > 0 {
		start := mapped[bestStartIdx].offset
		end := mapped[bestEndIdx-1].offset + 1
		return start, end, bestScore
	}

	return 0, 0, 0
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

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
