// Package main is the entry point for the neko MCP server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"syscall"

	"github.com/danicat/neko/internal/backend"
	golangbe "github.com/danicat/neko/internal/backend/golang"
	"github.com/danicat/neko/internal/backend/plugin"
	pythonbe "github.com/danicat/neko/internal/backend/python"
	"github.com/danicat/neko/internal/core/config"
	"github.com/danicat/neko/internal/instructions"
	"github.com/danicat/neko/internal/server"
	"github.com/danicat/neko/internal/toolnames"
)

var (
	version = "dev"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := run(ctx, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		return err
	}

	// Check for AI credentials and disable code_review if missing
	if os.Getenv("GOOGLE_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" &&
		os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "" {
		cfg.DisableTool("code_review")
	}

	if cfg.Version {
		fmt.Println(version)
		return nil
	}

	if cfg.ListTools {
		var tools []toolnames.ToolDef
		for _, def := range toolnames.Registry {
			if cfg.IsToolEnabled(def.Name) {
				tools = append(tools, def)
			}
		}

		sort.Slice(tools, func(i, j int) bool {
			return tools[i].Name < tools[j].Name
		})

		for _, tool := range tools {
			fmt.Printf("Name: %s\nTitle: %s\nDescription: %s\n\n", tool.Name, tool.Title, tool.Description)
		}
		return nil
	}

	if cfg.Agents {
		reg := backend.NewRegistry()
		if _, err := exec.LookPath("go"); err == nil {
			reg.Register(golangbe.New())
		}
		if _, err := exec.LookPath("python3"); err == nil {
			reg.Register(pythonbe.New())
		} else if _, err := exec.LookPath("python"); err == nil {
			reg.Register(pythonbe.New())
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

		fmt.Println(instructions.Get(cfg, reg))
		return nil
	}

	srv := server.New(cfg, version)

	if cfg.ListenAddr != "" {
		return srv.ServeHTTP(ctx, cfg.ListenAddr)
	}

	return srv.Run(ctx)
}
