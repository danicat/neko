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

	// Header with detected languages
	langs := reg.Available()
	if len(langs) > 0 {
		sb.WriteString("# Smart Tooling Guide (")
		sb.WriteString(strings.Join(langs, ", "))
		sb.WriteString(")\n\n")
	} else {
		sb.WriteString("# Smart Tooling Guide\n\n")
	}

	// Navigation
	sb.WriteString("### 🔍 Navigation: Save Tokens & Context\n")
	if isEnabled("smart_read") {
		sb.WriteString(toolnames.Registry["smart_read"].Instruction + "\n")
	}
	if isEnabled("list_files") {
		sb.WriteString(toolnames.Registry["list_files"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Editing
	sb.WriteString("### ✏️ Editing: Ensure Safety\n")
	if isEnabled("smart_edit") {
		sb.WriteString(toolnames.Registry["smart_edit"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Modernization & Upgrades
	sb.WriteString("### 🚀 Modernization & Upgrades\n")
	if isEnabled("modernize_code") {
		sb.WriteString(toolnames.Registry["modernize_code"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// Utilities
	sb.WriteString("### 🛠️ Utilities\n")
	if isEnabled("smart_build") {
		sb.WriteString(toolnames.Registry["smart_build"].Instruction + "\n")
	}
	if isEnabled("read_docs") {
		sb.WriteString(toolnames.Registry["read_docs"].Instruction + "\n")
	}
	if isEnabled("add_dependency") {
		sb.WriteString(toolnames.Registry["add_dependency"].Instruction + "\n")
	}
	if isEnabled("project_init") {
		sb.WriteString(toolnames.Registry["project_init"].Instruction + "\n")
	}
	if isEnabled("code_review") {
		sb.WriteString(toolnames.Registry["code_review"].Instruction + "\n")
	}
	sb.WriteString("\n")

	// LSP
	sb.WriteString("### 🔎 Code Intelligence (LSP)\n")
	if isEnabled("symbol_info") {
		sb.WriteString(toolnames.Registry["symbol_info"].Instruction + "\n")
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
	if isEnabled("mutation_test") {
		sb.WriteString(toolnames.Registry["mutation_test"].Instruction + "\n")
	}
	if isEnabled("test_query") {
		sb.WriteString(toolnames.Registry["test_query"].Instruction + "\n")
	}

	return sb.String()
}
