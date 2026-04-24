package resolver

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/myjupyter/errgen/internal/model"
)

// Resolver resolves short package names to full import paths
// by scanning the module directory tree
type Resolver struct {
	modulePath string
	moduleDir  string
}

// New creates a Resolver by finding go.mod starting from dir and walking up
func New(dir string) (*Resolver, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, model.NewResolvingError(err)
	}

	moduleDir, modulePath, err := findModule(absDir)
	if err != nil {
		return nil, err
	}

	return &Resolver{
		modulePath: modulePath,
		moduleDir:  moduleDir,
	}, nil
}

// Resolve takes a set of short package names and returns a map of
// package name to full import path. Returns an error if a package
// name has zero or multiple matches
func (r *Resolver) Resolve(pkgNames []string) (map[string]string, error) {
	if len(pkgNames) == 0 {
		return nil, nil
	}

	// Build a map of package name -> list of import paths found
	found := make(map[string][]string)
	for _, name := range pkgNames {
		found[name] = nil
	}

	err := filepath.Walk(r.moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip inaccessible dirs
		}
		if !info.IsDir() {
			return nil
		}
		// skip hidden dirs and vendor
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
			return filepath.SkipDir
		}

		dirPkgName, err := dirPackageName(path)
		if err != nil || dirPkgName == "" {
			return nil //nolint:nilerr // skip dirs without valid Go files
		}

		if _, needed := found[dirPkgName]; !needed {
			return nil
		}

		rel, err := filepath.Rel(r.moduleDir, path)
		if err != nil {
			return nil //nolint:nilerr // skip unresolvable relative paths
		}

		importPath := r.modulePath
		if rel != "." {
			importPath = r.modulePath + "/" + filepath.ToSlash(rel)
		}

		found[dirPkgName] = append(found[dirPkgName], importPath)
		return nil
	})
	if err != nil {
		return nil, model.NewResolvingError(err)
	}

	result := make(map[string]string, len(pkgNames))
	for _, name := range pkgNames {
		paths := found[name]
		switch len(paths) {
		case 0:
			return nil, model.NewPackageNotFoundError(name, r.modulePath)
		case 1:
			result[name] = paths[0]
		default:
			return nil, model.NewAmbiguousPackageError(name, strings.Join(paths, "\n  "))
		}
	}

	return result, nil
}

// findModule walks up from dir looking for go.mod and returns
// the module directory and module path
func findModule(dir string) (string, string, error) {
	for {
		modFile := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modFile) //nolint:gosec // walking module tree for go.mod
		if err == nil {
			modPath, err := parseModulePath(string(data))
			if err != nil {
				return "", "", err
			}
			return dir, modPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", model.NewGoModNotFoundError()
		}
		dir = parent
	}
}

// parseModulePath extracts the module path from go.mod content
func parseModulePath(content string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", model.NewNoModuleDirectiveError()
}

// dirPackageName returns the Go package name declared in the given directory,
// or empty string if no Go files exist
func dirPackageName(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.PackageClauseOnly)
		if err != nil {
			continue
		}
		return f.Name.Name, nil
	}

	return "", nil
}

// ImportPathForDir returns the full Go import path for the given directory
// by locating go.mod and computing the relative path
func ImportPathForDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", model.NewResolvingError(err)
	}
	moduleDir, modulePath, err := findModule(absDir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(moduleDir, absDir)
	if err != nil {
		return "", model.NewResolvingError(err)
	}
	if rel == "." {
		return modulePath, nil
	}
	return modulePath + "/" + filepath.ToSlash(rel), nil
}

// ExtractPackageNames extracts unique package names from field types
// For example, from "pkg.Type", "*pkg.Type", "[]pkg.Type" it extracts "pkg"
func ExtractPackageNames(types []string) []string {
	seen := make(map[string]bool)
	var names []string

	for _, t := range types {
		name := ExtractPkgName(t)
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return names
}

// ExtractPkgName extracts the package name from a type string
// Returns empty string for builtin types
func ExtractPkgName(typ string) string {
	// Strip pointer, slice, and map prefixes
	typ = strings.TrimLeft(typ, "*[]")
	if strings.HasPrefix(typ, "map[") {
		// Find the closing bracket
		depth := 1
		i := 4
		for i < len(typ) && depth > 0 {
			if typ[i] == '[' { //nolint:staticcheck // simple if-else is clearer here
				depth++
			} else if typ[i] == ']' {
				depth--
			}
			i++
		}
		if i < len(typ) {
			typ = strings.TrimLeft(typ[i:], "*[]")
		}
	}

	if idx := strings.Index(typ, "."); idx > 0 {
		return typ[:idx]
	}
	return ""
}
