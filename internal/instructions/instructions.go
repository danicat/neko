// Package instructions generates dynamic system instructions for the AI agent.
package instructions

import (
	"strings"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/config"
	"github.com/danicat/neko/internal/toolnames"
)

// Get returns the agent instructions for the server based on enabled tools and available backends.
func Get(cfg *config.Config, reg *backend.Registry) string {
	var sb strings.Builder

	isEnabled := func(tool string) bool {
		return cfg.IsToolEnabled(tool)
	}

	// Header
	langs := reg.Available()
	if len(langs) > 0 {
		sb.WriteString("# Neko Project Guide (")
		sb.WriteString(strings.Join(langs, ", "))
		sb.WriteString(")\n\n")
	} else {
		sb.WriteString("# Neko Project Guide\n\n")
	}

	sb.WriteString("Neko operates in two phases: **Lobby** and **Project Open**.\n")
	sb.WriteString("- In the **Lobby**, you can only `open_project` or `create_project`.\n")
	sb.WriteString("- Once a project is open, you gain access to navigation, editing, and engineering tools.\n\n")

	// Project Lifecycle
	sb.WriteString("### 📁 Project Lifecycle\n")
	sb.WriteString(toolnames.Registry["open_project"].Instruction + "\n")
	sb.WriteString(toolnames.Registry["create_project"].Instruction + "\n")
	sb.WriteString(toolnames.Registry["close_project"].Instruction + "\n\n")

	// Navigation
	sb.WriteString("### 🔍 Navigation: Save Tokens & Context\n")
	if isEnabled("read_file") {
		sb.WriteString(toolnames.Registry["read_file"].Instruction + "\n")
	}
	if isEnabled("list_files") {
		sb.WriteString(toolnames.Registry["list_files"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Editing
	sb.WriteString("### ✏️ Editing: Ensure Safety\n")
	if isEnabled("edit_file") {
		sb.WriteString(toolnames.Registry["edit_file"].Instruction + "\n")
	}
	if isEnabled("create_file") {
		sb.WriteString(toolnames.Registry["create_file"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Utilities
	sb.WriteString("### 🛠️ Utilities\n")
	if isEnabled("build") {
		sb.WriteString(toolnames.Registry["build"].Instruction + "\n")
	}
	if isEnabled("read_docs") {
		sb.WriteString(toolnames.Registry["read_docs"].Instruction + "\n")
	}
	if isEnabled("add_dependencies") {
		sb.WriteString(toolnames.Registry["add_dependencies"].Instruction + "\n")
	}
	if isEnabled("review_code") {
		sb.WriteString(toolnames.Registry["review_code"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// LSP
	sb.WriteString("### 🔎 Code Intelligence (LSP)\n")
	if isEnabled("describe") {
		sb.WriteString(toolnames.Registry["describe"].Instruction + "\n")
	}
	if isEnabled("find_definition") {
		sb.WriteString(toolnames.Registry["find_definition"].Instruction + "\n")
	}
	if isEnabled("find_references") {
		sb.WriteString(toolnames.Registry["find_references"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Testing
	sb.WriteString("### 🧪 Testing\n")
	if isEnabled("test_mutations") {
		sb.WriteString(toolnames.Registry["test_mutations"].Instruction + "\n")
	}
	if isEnabled("query_tests") {
		sb.WriteString(toolnames.Registry["query_tests"].Instruction + "\n")
	}

	return sb.String()
}
