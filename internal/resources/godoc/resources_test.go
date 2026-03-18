package godoc

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResourceHandler(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		uri         string
		wantErr     bool
		wantContent string
	}{
		{
			name:        "Valid Package",
			uri:         "godoc://fmt",
			wantErr:     false,
			wantContent: "# fmt",
		},
		{
			name:        "Valid Symbol (Split Path)",
			uri:         "godoc://fmt/Println",
			wantErr:     false,
			wantContent: "func Println",
		},
		{
			name:        "Invalid Scheme",
			uri:         "http://google.com",
			wantErr:     true,
			wantContent: "invalid URI scheme",
		},
		{
			name:    "Missing Package",
			uri:     "godoc://non/existent",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			result, err := ResourceHandler(ctx, req)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result.Contents) == 0 {
					t.Error("Expected content, got empty")
				}
				if !strings.Contains(result.Contents[0].Text, tc.wantContent) {
					t.Errorf("Expected content to contain %q, got %q", tc.wantContent, result.Contents[0].Text)
				}
			}
		})
	}
}
