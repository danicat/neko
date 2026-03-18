# Language Backend Core Functions: High-Level Logic

This document describes the high-level logic (pseudo-code) for the core functions of the `LanguageBackend` interface, specifically how they are implemented in the generic `PluginBackend`.

## Common Helper: `runCommand(name, vars)`

Most functions rely on this helper to execute external tools defined in the JSON configuration.

```python
def runCommand(command_name, variables):
    # 1. Lookup command configuration in the plugin JSON
    config = plugin.commands[command_name]
    if not config:
        return error("Command not configured")

    # 2. Substitute placeholders in arguments
    # {{filename}}, {{dir}}, {{package}}, {{symbol}}, {{packages}}
    final_args = []
    for arg in config.args:
        final_args.append(replace_placeholders(arg, variables))

    # 3. Execute external process
    output, exit_code = execute(config.command, final_args)

    # 4. Return output or error based on exit code
    if exit_code != 0:
        return output, error("Command failed")
    return output, nil
```

---

## 1. File Understanding

### `Outline(filename)`
Returns a structural map of the file (e.g., function signatures).
```python
def Outline(filename):
    return runCommand("outline", {"filename": filename})
```

### `ParseImports(filename)`
Extracts a list of dependencies from a file.
```python
def ParseImports(filename):
    output = runCommand("parseImports", {"filename": filename})
    # Try to parse output as JSON list, fallback to line-by-line
    return parse_list(output)
```

---

## 2. File Safety (used by `smart_edit`)

### `Validate(filename)`
Checks for syntax errors.
```python
def Validate(filename):
    if "validate" not in plugin.commands:
        return nil # Optional: assume valid if no tool configured
    return runCommand("validate", {"filename": filename})
```

### `Format(filename)`
Applies idiomatic formatting.
```python
def Format(filename):
    if "format" not in plugin.commands:
        return nil # Optional: skip if no tool configured
    return runCommand("format", {"filename": filename})
```

---

## 3. Toolchain

### `BuildPipeline(dir, opts)`
Runs build, lint, and tests.
```python
def BuildPipeline(dir, opts):
    # Variables: {{dir}}, {{packages}}
    # Note: Plugins currently run a single "build" command. 
    # Complex pipelines should be wrapped in a script.
    output, err = runCommand("build", {"dir": dir, "packages": opts.packages})
    return BuildReport(output, is_error=(err != nil))
```

### `FetchDocs(package, symbol)`
Retrieves documentation for a specific symbol.
```python
def FetchDocs(package, symbol):
    return runCommand("fetchDocs", {"package": package, "symbol": symbol})
```

### `AddDependency(dir, packages)`
Installs new packages.
```python
def AddDependency(dir, packages):
    return runCommand("addDependency", {"dir": dir, "packages": join(packages)})
```

### `InitProject(opts)`
Bootstraps a new project.
```python
def InitProject(opts):
    return runCommand("init", {"path": opts.path, "modulePath": opts.module_path})
```

---

## 4. Testing & Quality

### `MutationTest(dir)`
Runs mutation testing to verify test suite quality.
```python
def MutationTest(dir):
    return runCommand("mutationTest", {"dir": dir})
```

### `BuildTestDB(dir, package)`
Collects test results into a local database for SQL querying.
```python
def BuildTestDB(dir, package):
    return runCommand("buildTestDB", {"dir": dir, "package": package})
```

### `QueryTestDB(dir, query)`
Executes SQL queries against the collected test data.
```python
def QueryTestDB(dir, query):
    return runCommand("queryTestDB", {"dir": dir, "query": query})
```

---

## 5. LSP (Language Server Protocol)

### `LSPCommand()`
Provides the command to start the language server.
```python
def LSPCommand():
    if plugin.lsp_config:
        return plugin.lsp_config.command, plugin.lsp_config.args, True
    return "", [], False
```
