// Package server implements the Model Context Protocol (MCP) server for neko.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/backend/plugin"
	"github.com/danicat/neko/internal/backend/python"
	"github.com/danicat/neko/internal/core/config"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/instructions"
	"github.com/danicat/neko/internal/lsp"
	"github.com/danicat/neko/internal/prompts"
	godocres "github.com/danicat/neko/internal/resources/godoc"
	"github.com/danicat/neko/internal/tools/file/create"
	"github.com/danicat/neko/internal/tools/file/edit"
	"github.com/danicat/neko/internal/tools/file/list"
	"github.com/danicat/neko/internal/tools/file/read"
	"github.com/danicat/neko/internal/tools/lang/codereview"
	"github.com/danicat/neko/internal/tools/lang/definition"
	"github.com/danicat/neko/internal/tools/lang/docs"
	"github.com/danicat/neko/internal/tools/lang/get"
	"github.com/danicat/neko/internal/tools/lang/modernize"
	"github.com/danicat/neko/internal/tools/lang/mutation"
	"github.com/danicat/neko/internal/tools/lang/project"
	"github.com/danicat/neko/internal/tools/lang/quality"
	"github.com/danicat/neko/internal/tools/lang/references"
	"github.com/danicat/neko/internal/tools/lang/symbolinfo"
	"github.com/danicat/neko/internal/tools/lang/testquery"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server encapsulates the MCP server and its configuration.
type Server struct {
	mcpServer       *mcp.Server
	cfg             *config.Config
	registry        *backend.Registry
	registeredTools map[string]bool
}

// New creates a new Server instance with auto-detected language backends.
func New(cfg *config.Config, version string) *Server {
	reg := backend.NewRegistry()

	// Auto-detect available backends
	if hasGo() {
		reg.Register(golang.New())
	}
	if hasPython() {
		reg.Register(python.New())
	}

	// Load language plugins
	if cfg.PluginDir != "" {
		plugins, err := plugin.LoadPlugins(cfg.PluginDir)
		if err == nil {
			for _, p := range plugins {
				reg.Register(p)
			}
		}
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "neko",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: instructions.Get(cfg, reg),
		RootsListChangedHandler: func(ctx context.Context, req *mcp.RootsListChangedRequest) {
			roots.Global.Sync(ctx, req.Session)
		},
	})

	return &Server{
		mcpServer:       s,
		cfg:             cfg,
		registry:        reg,
		registeredTools: make(map[string]bool),
	}
}

// Run starts the MCP server using Stdio.
func (s *Server) Run(ctx context.Context) error {
	defer lsp.DefaultManager.CloseAll()
	if err := s.RegisterHandlers(); err != nil {
		return err
	}
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// ServeHTTP starts the server over HTTP using StreamableHTTP.
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	defer lsp.DefaultManager.CloseAll()
	if err := s.RegisterHandlers(); err != nil {
		return err
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return s.mcpServer
	}, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.0.0.1") {
				if !strings.Contains(origin, "localhost") && !strings.Contains(origin, "127.0.0.1") {
					http.Error(w, "Forbidden: Invalid Origin", http.StatusForbidden)
					return
				}
			}
		}
		mcpHandler.ServeHTTP(w, r)
	})

	log.Printf("MCP HTTP Server starting on %s", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

// RegisterHandlers wires all tools, resources, and prompts.
func (s *Server) RegisterHandlers() error {
	type toolDef struct {
		name     string
		register func(*mcp.Server, *backend.Registry)
	}

	availableTools := []toolDef{
		{name: "smart_read", register: read.Register},
		{name: "smart_edit", register: edit.Register},
		{name: "file_create", register: create.Register},
		{name: "list_files", register: list.Register},
		{name: "smart_build", register: quality.Register},
		{name: "read_docs", register: docs.Register},
		{name: "add_dependency", register: get.Register},
		{name: "project_init", register: project.Register},
		{name: "modernize_code", register: modernize.Register},
		{name: "mutation_test", register: mutation.Register},
		{name: "test_query", register: testquery.Register},
		{name: "symbol_info", register: symbolinfo.Register},
		{name: "find_definition", register: definition.Register},
		{name: "find_references", register: references.Register},
	}

	validTools := make(map[string]bool)

	for _, t := range availableTools {
		validTools[t.name] = true
		if s.cfg.IsToolEnabled(t.name) {
			if !s.registeredTools[t.name] {
				t.register(s.mcpServer, s.registry)
				s.registeredTools[t.name] = true
			}
		}
	}

	// Register code_review tool (requires AI credentials, may self-disable)
	validTools["code_review"] = true
	if s.cfg.IsToolEnabled("code_review") && !s.registeredTools["code_review"] {
		codereview.Register(s.mcpServer, s.cfg.DefaultModel)
		s.registeredTools["code_review"] = true
	}

	// Validate disabled tools
	for name := range s.cfg.DisabledTools {
		if !validTools[name] {
			return fmt.Errorf("unknown tool disabled: %s", name)
		}
	}

	// Register prompts based on available backends
	if s.registry.Get("go") != nil {
		if !s.registeredTools["prompt_go_import_this"] {
			s.mcpServer.AddPrompt(prompts.GoImportThis(), prompts.GoImportThisHandler)
			s.registeredTools["prompt_go_import_this"] = true
		}
		if !s.registeredTools["prompt_go_code_review"] {
			s.mcpServer.AddPrompt(prompts.GoCodeReview(), prompts.GoCodeReviewHandler)
			s.registeredTools["prompt_go_code_review"] = true
		}
	}

	if s.registry.Get("python") != nil {
		if !s.registeredTools["prompt_python_import_this"] {
			s.mcpServer.AddPrompt(prompts.PythonImportThis(), prompts.PythonImportThisHandler)
			s.registeredTools["prompt_python_import_this"] = true
		}
		if !s.registeredTools["prompt_python_code_review"] {
			s.mcpServer.AddPrompt(prompts.PythonCodeReview(), prompts.PythonCodeReviewHandler)
			s.registeredTools["prompt_python_code_review"] = true
		}
	}

	// Register godoc resources if Go backend is available
	if s.registry.Get("go") != nil {
		if !s.registeredTools["resource_godoc"] {
			godocres.Register(s.mcpServer)
			s.registeredTools["resource_godoc"] = true
		}
	}

	return nil
}

// Registry returns the backend registry for external access.
func (s *Server) Registry() *backend.Registry {
	return s.registry
}

func hasGo() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

func hasPython() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}
