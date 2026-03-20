package edit

import (
	"testing"
)

// FuzzFindBestMatch checks for panics and basic invariants.
func FuzzFindBestMatch(f *testing.F) {
	f.Add("func main() {}", "func main")
	f.Add("some long content with newlines\nand tabs\t", "content")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, content, search string) {
		matches := findMatches(content, search)

		var score float64
		var start, end int
		if len(matches) > 0 {
			score = matches[0].Score
			start = matches[0].Start
			end = matches[0].End
		}

		if score < 0.0 || score > 1.0 {
			t.Errorf("Score out of range: %f", score)
		}
		if start < 0 || end < 0 {
			t.Errorf("Negative bounds: %d-%d", start, end)
		}
		if start > end {
			t.Errorf("Inverted bounds: %d-%d", start, end)
		}
		if score > 0 {
			if end > len(content) {
				t.Errorf("End %d > ContentLen %d", end, len(content))
			}
		}
	})
}

// FuzzFindBestMatch_Exact checks that exact substrings are ALWAYS found.
func FuzzFindBestMatch_Exact(f *testing.F) {
	f.Add("prefix", "target", "suffix")

	f.Fuzz(func(t *testing.T, prefix, target, suffix string) {
		normTarget := normalize(target)
		if normTarget == "" {
			return
		}

		content := prefix + target + suffix

		matches := findMatches(content, target)
		var score float64
		if len(matches) > 0 {
			score = matches[0].Score
		}
		if score < 0.99 {
			t.Errorf("Failed to find exact match.\nContent: %q\nSearch: %q\nScore: %f", content, target, score)
		}
	})
}
