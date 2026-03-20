package shared

import "testing"

func TestStripNekoTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"Single tag",
			"func main() { <NEKO>type: func()</NEKO> }",
			"func main() {  }",
		},
		{
			"Multiple tags",
			"var x = 10 <NEKO>type: int</NEKO>\nvar y = \"hi\" <NEKO>type: string</NEKO>",
			"var x = 10 \nvar y = \"hi\" ",
		},
		{
			"Multiline tag",
			"<NEKO>\nthis is\nmultiline\n</NEKO>code",
			"code",
		},
		{
			"No tags",
			"pure code",
			"code", // Wait, "pure code" should be "pure code"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redefining the 'expected' for "No tags" because I had a typo in thought
			expected := tt.expected
			if tt.name == "No tags" {
				expected = "pure code"
			}
			got := StripNekoTags(tt.input)
			if got != expected {
				t.Errorf("StripNekoTags(%q) = %q, want %q", tt.input, got, expected)
			}
		})
	}
}
