// Package backend defines the LanguageBackend interface and the backend registry.
package backend

import (
	"context"
)

// BuildOpts holds options for the build pipeline.
type BuildOpts struct {
	Packages string
	RunTests bool
	RunLint  bool
	AutoFix  bool
}

// BuildReport contains the results of a build pipeline run.
type BuildReport struct {
	Output  string
	IsError bool
}

// InitOpts holds options for project initialization.
type InitOpts struct {
	Path         string
	ModulePath   string
	Dependencies []string
}

// Capability represents a specific feature supported by a language backend.
type Capability string

const (
	CapToolchain     Capability = "toolchain"
	CapDocumentation Capability = "documentation"
	CapDependencies  Capability = "dependencies"
	CapModernize     Capability = "modernize"
	CapMutationTest  Capability = "mutation_test"
	CapTestQuery     Capability = "test_query"
	CapLSP           Capability = "lsp"
)

// LanguageBackend defines the interface every language must implement.
type LanguageBackend interface {
	// Capabilities returns the set of features supported by this backend.
	Capabilities() []Capability

	// File understanding
	Outline(ctx context.Context, filename string) (string, error)
	ImportDocs(ctx context.Context, imports []string) ([]string, error)
	ParseImports(ctx context.Context, filename string) ([]string, error)

	// File safety
	Validate(ctx context.Context, filename string) error
	Format(ctx context.Context, filename string) error

	// Toolchain
	BuildPipeline(ctx context.Context, dir string, opts BuildOpts) (*BuildReport, error)
	FetchDocs(ctx context.Context, dir string, pkg string, symbol string) (string, error)
	AddDependency(ctx context.Context, dir string, packages []string) (string, error)

	InitProject(ctx context.Context, opts InitOpts) error
	Modernize(ctx context.Context, dir string, fix bool) (string, error)

	// Testing
	MutationTest(ctx context.Context, dir string) (string, error)
	BuildTestDB(ctx context.Context, dir string, pkg string) error
	QueryTestDB(ctx context.Context, dir string, query string) (string, error)

	// LSP
	LSPCommand() (command string, args []string, ok bool)
	InitializationOptions() map[string]interface{}

	// Metadata
	LanguageID() string
	Name() string
	FileExtensions() []string
	SkipDirs() []string
	ProjectMarkers() []string
	Tier() int
}
