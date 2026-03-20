package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// regex for common compiler error formats
var (
	// Go: filename.go:12:3: error message
	goErrorRegex = regexp.MustCompile(`:(\d+):(\d+):`)
	// Python: File "filename.py", line 12
	pythonErrorRegex = regexp.MustCompile(`line (\d+)`)
	// General fallback: anything followed by a colon and a number
	genericErrorRegex = regexp.MustCompile(`:(\d+)`)

	nekoTagRegex = regexp.MustCompile(`(?s)<NEKO>.*?</NEKO>`)
)

// StripNekoTags removes all <NEKO>...</NEKO> tags from the given string.
func StripNekoTags(s string) string {
	return nekoTagRegex.ReplaceAllString(s, "")
}

// GetLineOffsets calculates the byte offsets for a given line range.
// line numbers are 1-based.
func GetLineOffsets(content string, startLine, endLine int) (int, int, error) {
	currentLine := 1
	startOffset := 0
	endOffset := len(content)

	foundStart := false

	if startLine <= 1 {
		startOffset = 0
		foundStart = true
	}

	for i, char := range content {
		if char == '\n' {
			currentLine++
			if !foundStart && currentLine == startLine {
				startOffset = i + 1
				foundStart = true
			}
			if endLine > 0 && currentLine > endLine {
				endOffset = i + 1
				break
			}
		}
	}

	if startLine > currentLine && startLine > 1 {
		return 0, 0, fmt.Errorf("start_line %d is beyond file length (%d lines)", startLine, currentLine)
	}

	return startOffset, endOffset, nil
}

// GetSnippet returns a context window around the specified line number.
func GetSnippet(content string, lineNum int) string {
	lines := strings.Split(content, "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return ""
	}

	start := max(lineNum-5, 1)
	end := min(lineNum+5, len(lines))

	var sb strings.Builder
	for i := start; i <= end; i++ {
		prefix := "  "
		if i == lineNum {
			prefix = "-> "
		}
		sb.WriteString(fmt.Sprintf("%s%d | %s\n", prefix, i, lines[i-1]))
	}
	return sb.String()
}

// ExtractErrorSnippet attempts to parse a line number from an error message
// and returns a snippet of the content around that line.
func ExtractErrorSnippet(content string, err error) string {
	errMsg := err.Error()

	var lineNum int

	// Try Go format
	if matches := goErrorRegex.FindStringSubmatch(errMsg); len(matches) > 1 {
		lineNum, _ = strconv.Atoi(matches[1])
	}

	// Try Python format
	if lineNum == 0 {
		if matches := pythonErrorRegex.FindStringSubmatch(errMsg); len(matches) > 1 {
			lineNum, _ = strconv.Atoi(matches[1])
		}
	}

	// Try generic format
	if lineNum == 0 {
		if matches := genericErrorRegex.FindStringSubmatch(errMsg); len(matches) > 1 {
			lineNum, _ = strconv.Atoi(matches[1])
		}
	}

	if lineNum == 0 {
		return "Could not determine error line from message: " + errMsg
	}

	return GetSnippet(content, lineNum)
}

// GetLineFromOffset calculates the 1-based line number for a given byte offset.
func GetLineFromOffset(content string, offset int) int {
	if offset < 0 || offset > len(content) {
		return 0
	}
	return strings.Count(content[:offset], "\n") + 1
}
