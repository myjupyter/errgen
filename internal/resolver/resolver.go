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

	// Walk the module directory tree
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
			return filepath.SkipDir
		}

		if _, needed := found[dirPkgName]; !needed {
			return nil
		}

		rel, err := filepath.Rel(r.moduleDir, path)
		if err != nil {
			return filepath.SkipDir
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

	// Try to find built-in packages match
	for pkgName := range found {
		path, ok := findBuiltin(pkgName)
		if ok {
			found[pkgName] = append(found[pkgName], path)
		}
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
func findModule(dir string) (dirName string, modulePath string, err error) {
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
			switch typ[i] {
			case '[':
				depth++
			case ']':
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

// findBuiltin maps a short Go package name (e.g. "http", "json", "time") to
// its full stdlib import path (e.g. "net/http", "encoding/json", "time").
// Ambiguous short names whose basename appears in more than one stdlib path
// — `template` (text/template vs html/template), `rand` (math/rand vs
// crypto/rand), `pprof` (net/http/pprof vs runtime/pprof), `scanner`
// (go/scanner vs text/scanner) — are deliberately omitted; users should
// disambiguate with `-m`.
func findBuiltin(pkgName string) (string, bool) { //nolint:funlen,gocyclo // straight-line dispatch
	switch pkgName {
	// archive
	case "tar":
		return "archive/tar", true
	case "zip":
		return "archive/zip", true
	// top-level / single-segment
	case "bufio":
		return "bufio", true
	case "bytes":
		return "bytes", true
	case "cmp":
		return "cmp", true
	case "context":
		return "context", true
	case "crypto":
		return "crypto", true
	case "embed":
		return "embed", true
	case "encoding":
		return "encoding", true
	case "errors":
		return "errors", true
	case "expvar":
		return "expvar", true
	case "flag":
		return "flag", true
	case "fmt":
		return "fmt", true
	case "hash":
		return "hash", true
	case "html":
		return "html", true
	case "image":
		return "image", true
	case "io":
		return "io", true
	case "log":
		return "log", true
	case "maps":
		return "maps", true
	case "math":
		return "math", true
	case "mime":
		return "mime", true
	case "net":
		return "net", true
	case "os":
		return "os", true
	case "path":
		return "path", true
	case "plugin":
		return "plugin", true
	case "reflect":
		return "reflect", true
	case "regexp":
		return "regexp", true
	case "runtime":
		return "runtime", true
	case "slices":
		return "slices", true
	case "sort":
		return "sort", true
	case "strconv":
		return "strconv", true
	case "strings":
		return "strings", true
	case "sync":
		return "sync", true
	case "syscall":
		return "syscall", true
	case "testing":
		return "testing", true
	case "time":
		return "time", true
	case "unicode":
		return "unicode", true
	case "unsafe":
		return "unsafe", true
	// compress
	case "bzip2":
		return "compress/bzip2", true
	case "flate":
		return "compress/flate", true
	case "gzip":
		return "compress/gzip", true
	case "lzw":
		return "compress/lzw", true
	case "zlib":
		return "compress/zlib", true
	// container
	case "heap":
		return "container/heap", true
	case "list":
		return "container/list", true
	case "ring":
		return "container/ring", true
	// crypto/*
	case "aes":
		return "crypto/aes", true
	case "cipher":
		return "crypto/cipher", true
	case "des":
		return "crypto/des", true
	case "dsa":
		return "crypto/dsa", true
	case "ecdh":
		return "crypto/ecdh", true
	case "ecdsa":
		return "crypto/ecdsa", true
	case "ed25519":
		return "crypto/ed25519", true
	case "elliptic":
		return "crypto/elliptic", true
	case "hmac":
		return "crypto/hmac", true
	case "md5":
		return "crypto/md5", true
	case "rc4":
		return "crypto/rc4", true
	case "rsa":
		return "crypto/rsa", true
	case "sha1":
		return "crypto/sha1", true
	case "sha256":
		return "crypto/sha256", true
	case "sha512":
		return "crypto/sha512", true
	case "subtle":
		return "crypto/subtle", true
	case "tls":
		return "crypto/tls", true
	case "x509":
		return "crypto/x509", true
	case "pkix":
		return "crypto/x509/pkix", true
	// database/sql
	case "sql":
		return "database/sql", true
	case "driver":
		return "database/sql/driver", true
	// debug
	case "buildinfo":
		return "debug/buildinfo", true
	case "dwarf":
		return "debug/dwarf", true
	case "elf":
		return "debug/elf", true
	case "gosym":
		return "debug/gosym", true
	case "macho":
		return "debug/macho", true
	case "pe":
		return "debug/pe", true
	case "plan9obj":
		return "debug/plan9obj", true
	// encoding/*
	case "ascii85":
		return "encoding/ascii85", true
	case "asn1":
		return "encoding/asn1", true
	case "base32":
		return "encoding/base32", true
	case "base64":
		return "encoding/base64", true
	case "binary":
		return "encoding/binary", true
	case "csv":
		return "encoding/csv", true
	case "gob":
		return "encoding/gob", true
	case "hex":
		return "encoding/hex", true
	case "json":
		return "encoding/json", true
	case "pem":
		return "encoding/pem", true
	case "xml":
		return "encoding/xml", true
	// go/*
	case "ast":
		return "go/ast", true
	case "build":
		return "go/build", true
	case "constraint":
		return "go/build/constraint", true
	case "constant":
		return "go/constant", true
	case "doc":
		return "go/doc", true
	case "comment":
		return "go/doc/comment", true
	case "format":
		return "go/format", true
	case "importer":
		return "go/importer", true
	case "parser":
		return "go/parser", true
	case "printer":
		return "go/printer", true
	case "token":
		return "go/token", true
	case "types":
		return "go/types", true
	// hash
	case "adler32":
		return "hash/adler32", true
	case "crc32":
		return "hash/crc32", true
	case "crc64":
		return "hash/crc64", true
	case "fnv":
		return "hash/fnv", true
	case "maphash":
		return "hash/maphash", true
	// image
	case "color":
		return "image/color", true
	case "palette":
		return "image/color/palette", true
	case "draw":
		return "image/draw", true
	case "gif":
		return "image/gif", true
	case "jpeg":
		return "image/jpeg", true
	case "png":
		return "image/png", true
	case "suffixarray":
		return "index/suffixarray", true
	// io
	case "fs":
		return "io/fs", true
	case "ioutil":
		return "io/ioutil", true
	// log
	case "slog":
		return "log/slog", true
	case "syslog":
		return "log/syslog", true
	// math/*
	case "big":
		return "math/big", true
	case "bits":
		return "math/bits", true
	case "cmplx":
		return "math/cmplx", true
	// mime
	case "multipart":
		return "mime/multipart", true
	case "quotedprintable":
		return "mime/quotedprintable", true
	// net/*
	case "http":
		return "net/http", true
	case "cgi":
		return "net/http/cgi", true
	case "cookiejar":
		return "net/http/cookiejar", true
	case "fcgi":
		return "net/http/fcgi", true
	case "httptest":
		return "net/http/httptest", true
	case "httptrace":
		return "net/http/httptrace", true
	case "httputil":
		return "net/http/httputil", true
	case "mail":
		return "net/mail", true
	case "netip":
		return "net/netip", true
	case "rpc":
		return "net/rpc", true
	case "jsonrpc":
		return "net/rpc/jsonrpc", true
	case "smtp":
		return "net/smtp", true
	case "textproto":
		return "net/textproto", true
	case "url":
		return "net/url", true
	// os
	case "exec":
		return "os/exec", true
	case "signal":
		return "os/signal", true
	case "user":
		return "os/user", true
	// path
	case "filepath":
		return "path/filepath", true
	// regexp
	case "syntax":
		return "regexp/syntax", true
	// runtime
	case "cgo":
		return "runtime/cgo", true
	case "coverage":
		return "runtime/coverage", true
	case "metrics":
		return "runtime/metrics", true
	case "race":
		return "runtime/race", true
	case "trace":
		return "runtime/trace", true
	case "debug":
		return "runtime/debug", true
	// sync
	case "atomic":
		return "sync/atomic", true
	// testing
	case "fstest":
		return "testing/fstest", true
	case "iotest":
		return "testing/iotest", true
	case "quick":
		return "testing/quick", true
	case "slogtest":
		return "testing/slogtest", true
	// text
	case "tabwriter":
		return "text/tabwriter", true
	case "parse":
		return "text/template/parse", true
	// time
	case "tzdata":
		return "time/tzdata", true
	// unicode
	case "utf16":
		return "unicode/utf16", true
	case "utf8":
		return "unicode/utf8", true
	default:
		return "", false
	}
}
