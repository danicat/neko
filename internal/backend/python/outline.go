package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const outlineScript = `
import ast, sys

def outline(filename):
    with open(filename) as f:
        tree = ast.parse(f.read(), filename)

    lines = []
    for node in ast.iter_child_nodes(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                lines.append(f"import {alias.name}")
        elif isinstance(node, ast.ImportFrom):
            names = ", ".join(a.name for a in node.names)
            lines.append(f"from {node.module} import {names}")
        elif isinstance(node, ast.FunctionDef) or isinstance(node, ast.AsyncFunctionDef):
            decorators = "".join(f"@{ast.unparse(d)}\n" for d in node.decorator_list)
            args = ast.unparse(node.args) if node.args.args else ""
            ret = f" -> {ast.unparse(node.returns)}" if node.returns else ""
            prefix = "async " if isinstance(node, ast.AsyncFunctionDef) else ""
            doc = ast.get_docstring(node)
            doc_str = f'\n    """{doc}"""' if doc else ""
            lines.append(f"{decorators}{prefix}def {node.name}({args}){ret}: ...{doc_str}")
        elif isinstance(node, ast.ClassDef):
            decorators = "".join(f"@{ast.unparse(d)}\n" for d in node.decorator_list)
            bases = ", ".join(ast.unparse(b) for b in node.bases) if node.bases else ""
            base_str = f"({bases})" if bases else ""
            doc = ast.get_docstring(node)
            doc_str = f'\n    """{doc}"""' if doc else ""
            methods = []
            for item in ast.iter_child_nodes(node):
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    m_dec = "".join(f"    @{ast.unparse(d)}\n" for d in item.decorator_list)
                    m_args = ast.unparse(item.args) if item.args.args else ""
                    m_ret = f" -> {ast.unparse(item.returns)}" if item.returns else ""
                    m_prefix = "async " if isinstance(item, ast.AsyncFunctionDef) else ""
                    methods.append(f"{m_dec}    {m_prefix}def {item.name}({m_args}){m_ret}: ...")
            method_block = "\n".join(methods) if methods else "    ..."
            lines.append(f"{decorators}class {node.name}{base_str}:{doc_str}\n{method_block}")
        elif isinstance(node, ast.Assign):
            targets = ", ".join(ast.unparse(t) for t in node.targets)
            value = ast.unparse(node.value)
            if len(value) > 80:
                value = value[:77] + "..."
            lines.append(f"{targets} = {value}")
        elif isinstance(node, ast.AnnAssign):
            target = ast.unparse(node.target)
            ann = ast.unparse(node.annotation)
            if node.value:
                value = ast.unparse(node.value)
                if len(value) > 80:
                    value = value[:77] + "..."
                lines.append(f"{target}: {ann} = {value}")
            else:
                lines.append(f"{target}: {ann}")

    return "\n\n".join(lines)

if __name__ == "__main__":
    print(outline(sys.argv[1]))
`

const parseImportsScript = `
import ast, sys, json

def parse_imports(filename):
    with open(filename) as f:
        tree = ast.parse(f.read(), filename)
    modules = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                modules.add(alias.name.split(".")[0])
        elif isinstance(node, ast.ImportFrom):
            if node.module:
                modules.add(node.module.split(".")[0])
    # Filter out stdlib using sys.stdlib_module_names (Python 3.10+)
    stdlib = getattr(sys, "stdlib_module_names", set())
    third_party = sorted(m for m in modules if m not in stdlib and not m.startswith("_"))
    print(json.dumps(third_party))

parse_imports(sys.argv[1])
`

func pythonOutline(ctx context.Context, filename string) (string, error) {
	cmd := exec.CommandContext(ctx, "python3", "-c", outlineScript, filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("outline failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func pythonParseImports(ctx context.Context, filename string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "python3", "-c", parseImportsScript, filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("import parsing failed: %s", strings.TrimSpace(string(out)))
	}
	var imports []string
	if err := json.Unmarshal(out, &imports); err != nil {
		return nil, fmt.Errorf("failed to parse import list: %v", err)
	}
	return imports, nil
}
