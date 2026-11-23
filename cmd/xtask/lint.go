package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	nolintJustifiedRe = regexp.MustCompile(`^//nolint(?::[A-Za-z0-9_,-]+)?\s+//\s+\S.{4,}$`)
	todoCommentRe     = regexp.MustCompile(`\b(?:TODO|FIXME)\b`)
)

const golangciVersion = "v2.15.0"

func runLint(root string) error {
	if err := runExec(root, "go", "vet", "./..."); err != nil {
		return err
	}
	if err := runGolangciLint(root); err != nil {
		return err
	}
	if err := checkGoLineLimits(root); err != nil {
		return err
	}
	fset := token.NewFileSet()
	return filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		return inspectGoFile(fset, path, file)
	})
}

func runGolangciLint(root string) error {
	bin, err := exec.LookPath("golangci-lint")
	if err != nil {
		bin = goBinDir("golangci-lint")
		if _, statErr := os.Stat(bin); statErr != nil {
			return fmt.Errorf("golangci-lint %s not found; install the pinned binary:\n  curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b \"$(go env GOPATH)/bin\" %s", golangciVersion, golangciVersion)
		}
	}
	return runExec(root, bin, "run", "--config", ".golangci.yml", "./...")
}

func goBinDir(name string) string {
	out, err := exec.Command("go", "env", "GOBIN").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return filepath.Join(strings.TrimSpace(string(out)), name)
	}
	out, err = exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return name
	}
	return filepath.Join(strings.TrimSpace(string(out)), "bin", name)
}

func checkGoLineLimits(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if count := len(strings.Split(string(raw), "\n")); count > 500 {
			return fmt.Errorf("file %s exceeds 500 lines (%d)", path, count)
		}
		return nil
	})
}

func inspectGoFile(fset *token.FileSet, path string, file *ast.File) error {
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == `"unsafe"` {
			return fmt.Errorf("forbidden unsafe package import in %s", path)
		}
		if imp.Name != nil && imp.Name.Name == "." {
			return fmt.Errorf("forbidden dot import in %s", path)
		}
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			if strings.HasPrefix(text, "//nolint") && !nolintJustifiedRe.MatchString(text) {
				pos := fset.Position(c.Pos())
				return fmt.Errorf("bare //nolint without rule and '// reason' at %s:%d", path, pos.Line)
			}
			if todoCommentRe.MatchString(text) {
				pos := fset.Position(c.Pos())
				return fmt.Errorf("unresolved TODO or FIXME comment at %s:%d", path, pos.Line)
			}
		}
	}

	for _, decl := range file.Decls {
		if err := inspectTopDecl(fset, path, decl); err != nil {
			return err
		}
	}

	var lintErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if lintErr != nil {
			return false
		}
		if iface, ok := n.(*ast.InterfaceType); ok {
			if iface.Methods == nil || len(iface.Methods.List) == 0 {
				pos := fset.Position(iface.Pos())
				lintErr = fmt.Errorf("use 'any' instead of empty interface{} at %s:%d", path, pos.Line)
				return false
			}
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
			lintErr = fmt.Errorf("forbidden panic() in %s", path)
			return false
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok {
				name := sel.Sel.Name
				if (pkg.Name == "fmt" && strings.HasPrefix(name, "Print")) ||
					(pkg.Name == "os" && name == "Exit") ||
					(pkg.Name == "log" && (strings.HasPrefix(name, "Fatal") || strings.HasPrefix(name, "Panic"))) ||
					(pkg.Name == "time" && name == "Sleep") ||
					(pkg.Name == "context" && (name == "Background" || name == "TODO")) {
					pos := fset.Position(call.Pos())
					lintErr = fmt.Errorf("forbidden %s.%s at %s:%d", pkg.Name, name, path, pos.Line)
					return false
				}
			}
		}
		return true
	})

	skipAssert := map[ast.Node]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if lintErr != nil {
			return false
		}
		switch v := n.(type) {
		case *ast.TypeSwitchStmt:
			if v.Assign != nil {
				skipAssert[v.Assign] = true
			}
		case *ast.AssignStmt:
			if len(v.Lhs) >= 2 {
				for _, rhs := range v.Rhs {
					skipAssert[rhs] = true
				}
			}
		case *ast.StarExpr:
			if _, ok := v.X.(*ast.InterfaceType); ok {
				pos := fset.Position(v.Pos())
				lintErr = fmt.Errorf("pointer-to-interface is forbidden at %s:%d", path, pos.Line)
				return false
			}
		case *ast.StructType:
			for _, field := range v.Fields.List {
				if len(field.Names) != 0 {
					continue
				}
				sel, ok := field.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				pkg, ok := sel.X.(*ast.Ident)
				if ok && pkg.Name == "sync" && (sel.Sel.Name == "Mutex" || sel.Sel.Name == "RWMutex") {
					pos := fset.Position(field.Pos())
					lintErr = fmt.Errorf("embedded sync.%s is forbidden at %s:%d", sel.Sel.Name, path, pos.Line)
					return false
				}
			}
		case *ast.TypeAssertExpr:
			if skipAssert[v] || v.Type == nil {
				return true
			}
			pos := fset.Position(v.Pos())
			lintErr = fmt.Errorf("unchecked type assertion requires comma-ok at %s:%d", path, pos.Line)
			return false
		}
		return true
	})
	return lintErr
}

func inspectTopDecl(fset *token.FileSet, path string, decl ast.Decl) error {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && d.Name.Name == "init" {
			return fmt.Errorf("forbidden init() function in %s", path)
		}
		if ast.IsExported(d.Name.Name) {
			if d.Doc == nil {
				pos := fset.Position(d.Pos())
				return fmt.Errorf("exported func %s missing doc comment at %s:%d", d.Name.Name, path, pos.Line)
			}
			if d.Type.Params != nil {
				for _, field := range d.Type.Params.List {
					if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "bool" {
						pos := fset.Position(field.Pos())
						return fmt.Errorf("boolean parameter forbidden in exported func %s at %s:%d", d.Name.Name, path, pos.Line)
					}
				}
			}
		}
	case *ast.GenDecl:
		if d.Tok == token.VAR {
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if name.Name == "_" && vs.Type != nil && len(vs.Values) == 1 {
						continue
					}
					if !strings.HasPrefix(name.Name, "Err") {
						pos := fset.Position(name.Pos())
						return fmt.Errorf("mutable package var %q forbidden (only Err* sentinels allowed) at %s:%d", name.Name, path, pos.Line)
					}
					if ast.IsExported(name.Name) && d.Doc == nil && vs.Doc == nil {
						pos := fset.Position(name.Pos())
						return fmt.Errorf("exported sentinel %s missing doc comment at %s:%d", name.Name, path, pos.Line)
					}
				}
			}
		}
		if d.Tok == token.TYPE || d.Tok == token.CONST {
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) && d.Doc == nil && s.Doc == nil {
						pos := fset.Position(s.Pos())
						return fmt.Errorf("exported type %s missing doc comment at %s:%d", s.Name.Name, path, pos.Line)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if ast.IsExported(name.Name) && d.Doc == nil && s.Doc == nil {
							pos := fset.Position(name.Pos())
							return fmt.Errorf("exported const %s missing doc comment at %s:%d", name.Name, path, pos.Line)
						}
					}
				}
			}
		}
	}
	return nil
}
