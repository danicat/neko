// Package godoc implements the godoc resource handler.
package godoc

import (
	"context"
	"fmt"
	"path"
	"strings"

	godoclib "github.com/danicat/neko/internal/backend/golang/godoc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the godoc resources with the server.
func Register(server *mcp.Server) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "godoc://{path}",
		Name:        "Go Documentation",
		Description: "Documentation for Go packages and symbols (e.g. godoc://net/http or godoc://fmt.Println)",
		MIMEType:    "text/markdown",
	}, ResourceHandler)
}

// ResourceHandler handles the godoc:// resource requests.
func ResourceHandler(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	if !strings.HasPrefix(uri, "godoc://") {
		return nil, fmt.Errorf("invalid URI scheme")
	}
	pkgPath := strings.TrimPrefix(uri, "godoc://")

	// Try as package
	doc, err := godoclib.GetDocumentation(ctx, pkgPath, "")
	if err == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "text/markdown",
					Text:     doc,
				},
			},
		}, nil
	}

	// Try splitting for symbol (e.g. net/http/Client -> pkg: net/http, sym: Client)
	dir, file := path.Split(pkgPath)
	dir = strings.TrimSuffix(dir, "/")

	if dir != "" && file != "" {
		doc, err2 := godoclib.GetDocumentation(ctx, dir, file)
		if err2 == nil {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      uri,
						MIMEType: "text/markdown",
						Text:     doc,
					},
				},
			}, nil
		}
	}

	// Try splitting by dot (e.g. fmt.Println or net/http.Client)
	lastDot := strings.LastIndex(pkgPath, ".")
	if lastDot != -1 {
		pkg := pkgPath[:lastDot]
		sym := pkgPath[lastDot+1:]
		doc, err3 := godoclib.GetDocumentation(ctx, pkg, sym)
		if err3 == nil {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      uri,
						MIMEType: "text/markdown",
						Text:     doc,
					},
				},
			}, nil
		}
	}

	return nil, mcp.ResourceNotFoundError(uri)
}
