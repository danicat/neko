package godoc

import (
	"context"
	"strings"
	"testing"

	"github.com/danicat/neko/internal/core/textdist"
)

func TestGetDocumentation_StdLib(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		pkgPath     string
		symbolName  string
		wantContent []string
		wantErr     bool
	}{
		{
			name:        "Package fmt",
			pkgPath:     "fmt",
			symbolName:  "",
			wantContent: []string{"# fmt", "package fmt // import \"fmt\"", "Package fmt implements formatted I/O"},
			wantErr:     false,
		},
		{
			name:        "Function fmt.Println",
			pkgPath:     "fmt",
			symbolName:  "Println",
			wantContent: []string{"## function Println", "func Println(a ...any) (n int, err error)"},
			wantErr:     false,
		},
		{
			name:        "Type fmt.Stringer",
			pkgPath:     "fmt",
			symbolName:  "Stringer",
			wantContent: []string{"## type Stringer", "type Stringer interface", "String() string"},
			wantErr:     false,
		},
		{
			name:        "Non-existent symbol",
			pkgPath:     "fmt",
			symbolName:  "NonExistentVar",
			wantContent: []string{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetDocumentation(ctx, tt.pkgPath, tt.symbolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDocumentation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, want := range tt.wantContent {
					if !strings.Contains(got, want) {
						t.Errorf("GetDocumentation() missing content %q. Got:\n%s", want, got)
					}
				}
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"rosettacode", "raisethysword", 8},
		{"same", "same", 0},
		{"a", "", 1},
		{"", "b", 1},
	}

	for _, tt := range tests {
		got := textdist.Levenshtein(tt.s1, tt.s2)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
		}
	}
}

func TestFindFuzzyMatches(t *testing.T) {
	candidates := []string{"Println", "Printf", "Sprintf", "Stringer", "Scan", "fmt"}

	tests := []struct {
		query string
		want  []string
	}{
		{"Prntln", []string{"Println"}},
		{"printf", []string{"Println", "Printf", "Sprintf"}},
		{"sprint", []string{"Printf", "Sprintf"}},
		{"ftm", []string{"fmt"}},
		{"Xyz", nil},
	}

	for _, tt := range tests {
		got := findFuzzyMatches(tt.query, candidates)

		if len(got) != len(tt.want) {
			t.Errorf("findFuzzyMatches(%q) got %v, want %v", tt.query, got, tt.want)
			continue
		}

		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("findFuzzyMatches(%q) index %d: got %q, want %q", tt.query, i, got[i], tt.want[i])
			}
		}
	}
}
