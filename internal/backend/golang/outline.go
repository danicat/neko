package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

// goOutline returns a structural outline of a Go file.
func goOutline(filename string) (string, error) {
	outline, _, _, err := GetOutline(filename)
	if err != nil {
		return "", err
	}
	return outline, nil
}

// GetOutline loads a file and returns its outline, list of imports, and build errors.
func GetOutline(file string) (string, []string, []error, error) {
	fset := token.NewFileSet()
	//nolint:gosec // G304
	content, err := os.ReadFile(file)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	targetFile, err := parser.ParseFile(fset, file, content, parser.ParseComments)
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	if targetFile == nil {
		return "", nil, errs, fmt.Errorf("failed to parse file: %w", err)
	}

	var imports []string
	for _, imp := range targetFile.Imports {
		if imp.Path != nil {
			imports = append(imports, imp.Path.Value)
		}
	}

	outline := outlinize(targetFile)

	var buf bytes.Buffer
	config := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
	if err := config.Fprint(&buf, fset, outline); err != nil {
		return "", nil, errs, fmt.Errorf("failed to format outline: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return string(formatted), imports, errs, nil
}

func outlinize(f *ast.File) *ast.File {
	res := *f
	res.Decls = make([]ast.Decl, len(f.Decls))

	allowedComments := make(map[*ast.CommentGroup]bool)
	if f.Doc != nil {
		allowedComments[f.Doc] = true
	}
	for _, cg := range f.Comments {
		if cg.End() < f.Package {
			allowedComments[cg] = true
		}
	}

	for i, decl := range f.Decls {
		switch fn := decl.(type) {
		case *ast.FuncDecl:
			newFn := *fn
			newFn.Body = nil
			res.Decls[i] = &newFn
			if fn.Doc != nil {
				allowedComments[fn.Doc] = true
			}
		case *ast.GenDecl:
			res.Decls[i] = decl
			if fn.Doc != nil {
				allowedComments[fn.Doc] = true
			}
			for _, spec := range fn.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Doc != nil {
						allowedComments[s.Doc] = true
					}
					// Field docstrings in structs and method docstrings in interfaces
					switch t := s.Type.(type) {
					case *ast.StructType:
						for _, field := range t.Fields.List {
							if field.Doc != nil {
								allowedComments[field.Doc] = true
							}
						}
					case *ast.InterfaceType:
						for _, method := range t.Methods.List {
							if method.Doc != nil {
								allowedComments[method.Doc] = true
							}
						}
					}
				case *ast.ValueSpec:
					if s.Doc != nil {
						allowedComments[s.Doc] = true
					}
				}
			}
		default:
			res.Decls[i] = decl
		}
	}

	var newComments []*ast.CommentGroup
	for _, cg := range f.Comments {
		if allowedComments[cg] {
			newComments = append(newComments, cg)
		}
	}
	res.Comments = newComments

	return &res
}
