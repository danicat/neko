// Package server implements the Model Context Protocol (MCP) server for neko.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/config"
	"github.com/danicat/neko/internal/core/roots"
	"github.com/danicat/neko/internal/instructions"
	"github.com/danicat/neko/internal/lsp"
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
	describe "github.com/danicat/neko/internal/tools/lang/symbolinfo"
	"github.com/danicat/neko/internal/tools/lang/testquery"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server encapsulates the MCP server and its configuration.
type Server struct {
	mcpServer       *mcp.Server
	cfg             *config.Config
	registry        *backend.Registry
	registeredTools map[string]bool

	// Project state
	mu             sync.Mutex
	projectOpen    bool
	projectRoot    string
	activeBackends map[string]backend.LanguageBackend // keyed by Name()
}

// New creates a new Server instance with the given registry and config.
func New(cfg *config.Config, version string, reg *backend.Registry) *Server {
	s := &Server{
		cfg:             cfg,
		registry:        reg,
		registeredTools: make(map[string]bool),
		activeBackends:  make(map[string]backend.LanguageBackend),
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "neko",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: instructions.Get(cfg, reg),
		RootsListChangedHandler: func(ctx context.Context, req *mcp.RootsListChangedRequest) {
			roots.Global.Sync(ctx, req.Session)
		},
	})

	s.mcpServer = mcpServer
	return s
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.registerHandlersLocked()
}

func (s *Server) registerHandlersLocked() error {
	if !s.projectOpen {
		// Lobby Phase
		s.mcpServer.RemoveTools("close_project", "read_file", "edit_file", "list_files", "create_file", "build", "read_docs", "add_dependencies", "modernize_code", "test_mutations", "query_tests", "describe", "find_definition", "find_references", "review_code")

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "open_project",
			Title:       "Open Project",
			Description: "Opens an existing project directory.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
			Dir string `json:"dir" jsonschema:"The root directory of the project"`
		}) (*mcp.CallToolResult, any, error) {
			return s.openProjectHandler(ctx, req, args)
		})

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "create_project",
			Title:       "Create Project",
			Description: "Bootstraps a new project.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, args project.Params) (*mcp.CallToolResult, any, error) {
			return s.createProjectHandler(ctx, req, args)
		})
	} else {
		// Project Phase
		s.mcpServer.RemoveTools("open_project", "create_project")

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "close_project",
			Title:       "Close Project",
			Description: "Closes the current project.",
		}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			return s.closeProjectHandler(ctx, req, struct{}{})
		})

		// Agnostic tools
		read.Register(s.mcpServer, s)
		edit.Register(s.mcpServer, s)
		list.Register(s.mcpServer, s)
		create.Register(s.mcpServer, s)
		codereview.Register(s.mcpServer, s, s.cfg.DefaultModel)

		// Capability-based tools
		caps := make(map[backend.Capability]bool)
		for _, be := range s.activeBackends {
			for _, c := range be.Capabilities() {
				caps[c] = true
			}
		}

		if caps[backend.CapToolchain] {
			quality.Register(s.mcpServer, s)
		}
		if caps[backend.CapDocumentation] {
			docs.Register(s.mcpServer, s)
		}
		if caps[backend.CapDependencies] {
			get.Register(s.mcpServer, s)
		}
		if caps[backend.CapModernize] {
			modernize.Register(s.mcpServer, s)
		}
		if caps[backend.CapMutationTest] {
			mutation.Register(s.mcpServer, s)
		}
		if caps[backend.CapTestQuery] {
			testquery.Register(s.mcpServer, s)
		}
		if caps[backend.CapLSP] {
			describe.Register(s.mcpServer, s)
			definition.Register(s.mcpServer, s)
			references.Register(s.mcpServer, s)
		}
	}

	return nil
}

// Registry returns the backend registry for external access.
func (s *Server) Registry() *backend.Registry {
	return s.registry
}

// ResolveBackend returns the appropriate backend for a language-aware tool.
func (s *Server) ResolveBackend(language string) (backend.LanguageBackend, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	supported := s.registry.Available()

	if language != "" {
		if be, ok := s.activeBackends[language]; ok {
			return be, nil
		}
		if be := s.registry.Get(language); be != nil {
			return be, nil
		}
		return nil, fmt.Errorf("unknown language backend: %q. Supported languages: %s", language, strings.Join(supported, ", "))
	}

	if len(s.activeBackends) == 1 {
		for _, be := range s.activeBackends {
			return be, nil
		}
	}

	if len(s.activeBackends) == 0 {
		return nil, fmt.Errorf("no language backends active for this project. Supported languages: %s", strings.Join(supported, ", "))
	}

	names := make([]string, 0, len(s.activeBackends))
	for name := range s.activeBackends {
		names = append(names, name)
	}
	return nil, fmt.Errorf("multiple backends active (%s), please specify the 'language' parameter", strings.Join(names, ", "))
}

// establishProject handles the shared state transition when a project is opened or created.
func (s *Server) establishProject(ctx context.Context, absRoot string, backends []backend.LanguageBackend) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.projectOpen = true
	s.projectRoot = absRoot
	s.activeBackends = make(map[string]backend.LanguageBackend)
	for _, be := range backends {
		s.activeBackends[be.Name()] = be
	}

	// Eager LSP initialization
	for _, be := range backends {
		s.startLSP(ctx, be, absRoot)
	}

	// Update tools list
	s.registerHandlersLocked()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project established at %s\n", absRoot))
	if len(backends) > 0 {
		sb.WriteString("Active languages detected:\n")
		for _, be := range backends {
			sb.WriteString(fmt.Sprintf("- %s\n", be.Name()))
		}
	} else {
		sb.WriteString("No specific language backends detected. General file tools are enabled.")
	}
	return sb.String()
}

func (s *Server) startLSP(ctx context.Context, be backend.LanguageBackend, absRoot string) {
	if cmd, args, ok := be.LSPCommand(); ok {
		opts := be.InitializationOptions()
		_, err := lsp.DefaultManager.ClientFor(ctx, be.Name(), absRoot, cmd, args, be.LanguageID(), opts)
		if err != nil {
			log.Printf("Warning: failed to start LSP for %s: %v", be.LanguageID(), err)
		}
	}
}

// ForFile is the new routing mechanism that also handles dynamic backend activation.
func (s *Server) ForFile(ctx context.Context, path string) backend.LanguageBackend {
	be := s.registry.ForFile(path)
	if be == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.projectOpen {
		return be
	}

	_, active := s.activeBackends[be.Name()]
	if active {
		return be
	}

	// Dynamic activation ("On-Touch")
	log.Printf("Dynamically activating backend: %s", be.Name())
	s.activeBackends[be.Name()] = be
	root := s.projectRoot

	s.startLSP(ctx, be, root)
	s.registerHandlersLocked() // Re-register to potentially surface new tools

	return be
}

// openProjectHandler establishes a project context.
func (s *Server) openProjectHandler(ctx context.Context, _ *mcp.CallToolRequest, args struct {
	Dir string `json:"dir" jsonschema:"The root directory of the project"`
}) (*mcp.CallToolResult, any, error) {
	absRoot, err := roots.Global.Validate(args.Dir)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, nil, nil
	}

	backends := s.registry.DetectBackends(absRoot)
	msg := s.establishProject(ctx, absRoot, backends)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, nil, nil
}

// createProjectHandler bootstraps and opens a project.
func (s *Server) createProjectHandler(ctx context.Context, req *mcp.CallToolRequest, args project.Params) (*mcp.CallToolResult, any, error) {
	// 1. Initialize on disk
	res, _, err := project.InitHandler(ctx, args, s.registry)
	if err != nil || res.IsError {
		return res, nil, err
	}

	// 2. Open the newly created project
	absRoot, _ := roots.Global.Validate(args.Dir)
	backends := s.registry.DetectBackends(absRoot)

	// If detection failed (e.g. no marker yet), manually add the requested language backend
	if len(backends) == 0 && args.Language != "" {
		if be := s.registry.Get(args.Language); be != nil {
			backends = append(backends, be)
		}
	}

	msg := s.establishProject(ctx, absRoot, backends)

	// Combine messages
	initMsg := res.Content[0].(*mcp.TextContent).Text
	combinedMsg := initMsg + "\n\n" + msg

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: combinedMsg}},
	}, nil, nil
}

// closeProjectHandler cleans up the project context.
func (s *Server) closeProjectHandler(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	lsp.DefaultManager.CloseAll()

	s.mu.Lock()
	s.projectOpen = false
	s.projectRoot = ""
	s.activeBackends = make(map[string]backend.LanguageBackend)
	s.registerHandlersLocked()
	s.mu.Unlock()

	return &mcp.CallToolResult{

		Content: []mcp.Content{&mcp.TextContent{Text: "Project closed. Returned to lobby."}},
	}, nil, nil
}
