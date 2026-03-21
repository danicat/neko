// Package rag implements the semantic search engine using chromem-go.
package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danicat/neko/internal/lsp"
	"github.com/philippgille/chromem-go"
	"google.golang.org/genai"
)

// Engine manages the local vector database.
type Engine struct {
	db          *chromem.DB
	collection  *chromem.Collection
	mu          sync.Mutex
	projectRoot string
}

// NewEngine creates a new RAG engine.
func NewEngine(ctx context.Context, projectRoot string) (*Engine, error) {
	dbDir := filepath.Join(projectRoot, ".neko", "embeddings.db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("AI credentials missing: set GOOGLE_API_KEY or GEMINI_API_KEY")
	}

	db := chromem.NewDB()

	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, err
		}

		contents := []*genai.Content{
			genai.NewContentFromText(text, genai.RoleUser),
		}
		// Using gemini-embedding-001 as specified
		res, err := client.Models.EmbedContent(ctx, "gemini-embedding-001", contents, nil)
		if err != nil {
			return nil, err
		}
		if len(res.Embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings returned")
		}
		return res.Embeddings[0].Values, nil
	}

	collection, err := db.CreateCollection("code", nil, embeddingFunc)
	if err != nil {
		return nil, err
	}

	return &Engine{
		db:          db,
		collection:  collection,
		projectRoot: projectRoot,
	}, nil
}

// IngestFile chunks and embeds a single file.
func (e *Engine) IngestFile(ctx context.Context, path string, content string, symbols []lsp.DocumentSymbol, imports []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Header context
	header := fmt.Sprintf("File: %s\nImports: %s\n", path, strings.Join(imports, ", "))

	// 3. Chunking
	var docs []chromem.Document
	if len(symbols) == 0 {
		// Fallback to line-based chunking
		lines := strings.Split(content, "\n")
		for i := 0; i < len(lines); i += 50 {
			end := min(i+50, len(lines))
			chunk := strings.Join(lines[i:end], "\n")
			docs = append(docs, chromem.Document{
				ID:      fmt.Sprintf("%s:%d", path, i),
				Content: header + chunk,
				Metadata: map[string]string{
					"path": path,
					"line": fmt.Sprintf("%d", i+1),
				},
			})
		}
	} else {
		// Symbol-aware chunking
		for _, s := range symbols {
			if s.Kind == 12 || s.Kind == 6 || s.Kind == 5 { // Function, Method, Class
				lines := strings.Split(content, "\n")
				if s.Range.Start.Line < len(lines) && s.Range.End.Line < len(lines) {
					chunk := strings.Join(lines[s.Range.Start.Line:s.Range.End.Line+1], "\n")
					docs = append(docs, chromem.Document{
						ID:      fmt.Sprintf("%s:%s", path, s.Name),
						Content: header + chunk,
						Metadata: map[string]string{
							"path": path,
							"line": fmt.Sprintf("%d", s.Range.Start.Line+1),
							"name": s.Name,
						},
					})
				}
			}
		}
	}

	if len(docs) > 0 {
		return e.collection.AddDocuments(ctx, docs, 0)
	}
	return nil
}

// SearchResult represents a single semantic search result.
type SearchResult struct {
	Content    string
	Metadata   map[string]string
	Similarity float32
}

// Search finds similar code snippets.
func (e *Engine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	results, err := e.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{
			Content:    r.Content,
			Metadata:   r.Metadata,
			Similarity: r.Similarity,
		}
	}
	return out, nil
}
