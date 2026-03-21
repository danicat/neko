# Language Backends

The `internal/backend` package is the core abstraction that makes Neko a polyglot system. It defines an extensible interface for language-specific operations (Go, Python, etc.), allowing Neko to apply standard architectural principles (like the "Quality Gate") across entirely different tech stacks.

## Sub-Components
- [Interface Definition](interface.md): Details the mandatory `LanguageBackend` contract.
- [External Tool Management](tools.md): Strategy for managing external CLI tool dependencies across languages.
- [Go Implementation](go.md): How Go code is analyzed, tested, and formatted.
- [Python Implementation](python.md): Python equivalents using Ruff, Pytest, etc.
