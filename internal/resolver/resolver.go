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

// findBulitin searches in bultin packages and returns bultin package path
func findBuiltin(pkgName string) (string, bool) { //nolint:funlen,gocyclo // Bultin-in packages list
	switch pkgName {
	case "archive/tar":
		return "archive/tar", true
	case "archive/zip":
		return "archive/zip", true
	case "bufio":
		return "bufio", true
	case "bytes":
		return "bytes", true
	case "cmp":
		return "cmp", true
	case "compress/bzip2":
		return "compress/bzip2", true
	case "compress/flate":
		return "compress/flate", true
	case "compress/gzip":
		return "compress/gzip", true
	case "compress/lzw":
		return "compress/lzw", true
	case "compress/zlib":
		return "compress/zlib", true
	case "container/heap":
		return "container/heap", true
	case "container/list":
		return "container/list", true
	case "container/ring":
		return "container/ring", true
	case "context":
		return "context", true
	case "crypto":
		return "crypto", true
	case "crypto/aes":
		return "crypto/aes", true
	case "crypto/cipher":
		return "crypto/cipher", true
	case "crypto/des":
		return "crypto/des", true
	case "crypto/dsa":
		return "crypto/dsa", true
	case "crypto/ecdh":
		return "crypto/ecdh", true
	case "crypto/ecdsa":
		return "crypto/ecdsa", true
	case "crypto/ed25519":
		return "crypto/ed25519", true
	case "crypto/elliptic":
		return "crypto/elliptic", true
	case "crypto/hmac":
		return "crypto/hmac", true
	case "crypto/internal/alias":
		return "crypto/internal/alias", true
	case "crypto/internal/bigmod":
		return "crypto/internal/bigmod", true
	case "crypto/internal/boring":
		return "crypto/internal/boring", true
	case "crypto/internal/boring/bbig":
		return "crypto/internal/boring/bbig", true
	case "crypto/internal/boring/bcache":
		return "crypto/internal/boring/bcache", true
	case "crypto/internal/boring/sig":
		return "crypto/internal/boring/sig", true
	case "crypto/internal/edwards25519":
		return "crypto/internal/edwards25519", true
	case "crypto/internal/edwards25519/field":
		return "crypto/internal/edwards25519/field", true
	case "crypto/internal/nistec":
		return "crypto/internal/nistec", true
	case "crypto/internal/nistec/fiat":
		return "crypto/internal/nistec/fiat", true
	case "crypto/internal/randutil":
		return "crypto/internal/randutil", true
	case "crypto/md5":
		return "crypto/md5", true
	case "crypto/rand":
		return "crypto/rand", true
	case "crypto/rc4":
		return "crypto/rc4", true
	case "crypto/rsa":
		return "crypto/rsa", true
	case "crypto/sha1":
		return "crypto/sha1", true
	case "crypto/sha256":
		return "crypto/sha256", true
	case "crypto/sha512":
		return "crypto/sha512", true
	case "crypto/subtle":
		return "crypto/subtle", true
	case "crypto/tls":
		return "crypto/tls", true
	case "crypto/x509":
		return "crypto/x509", true
	case "crypto/x509/pkix":
		return "crypto/x509/pkix", true
	case "database/sql":
		return "database/sql", true
	case "database/sql/driver":
		return "database/sql/driver", true
	case "debug/buildinfo":
		return "debug/buildinfo", true
	case "debug/dwarf":
		return "debug/dwarf", true
	case "debug/elf":
		return "debug/elf", true
	case "debug/gosym":
		return "debug/gosym", true
	case "debug/macho":
		return "debug/macho", true
	case "debug/pe":
		return "debug/pe", true
	case "debug/plan9obj":
		return "debug/plan9obj", true
	case "embed":
		return "embed", true
	case "embed/internal/embedtest":
		return "embed/internal/embedtest", true
	case "encoding":
		return "encoding", true
	case "encoding/ascii85":
		return "encoding/ascii85", true
	case "encoding/asn1":
		return "encoding/asn1", true
	case "encoding/base32":
		return "encoding/base32", true
	case "encoding/base64":
		return "encoding/base64", true
	case "encoding/binary":
		return "encoding/binary", true
	case "encoding/csv":
		return "encoding/csv", true
	case "encoding/gob":
		return "encoding/gob", true
	case "encoding/hex":
		return "encoding/hex", true
	case "encoding/json":
		return "encoding/json", true
	case "encoding/pem":
		return "encoding/pem", true
	case "encoding/xml":
		return "encoding/xml", true
	case "errors":
		return "errors", true
	case "expvar":
		return "expvar", true
	case "flag":
		return "flag", true
	case "fmt":
		return "fmt", true
	case "go/ast":
		return "go/ast", true
	case "go/build":
		return "go/build", true
	case "go/build/constraint":
		return "go/build/constraint", true
	case "go/constant":
		return "go/constant", true
	case "go/doc":
		return "go/doc", true
	case "go/doc/comment":
		return "go/doc/comment", true
	case "go/format":
		return "go/format", true
	case "go/importer":
		return "go/importer", true
	case "go/internal/gccgoimporter":
		return "go/internal/gccgoimporter", true
	case "go/internal/gcimporter":
		return "go/internal/gcimporter", true
	case "go/internal/srcimporter":
		return "go/internal/srcimporter", true
	case "go/internal/typeparams":
		return "go/internal/typeparams", true
	case "go/parser":
		return "go/parser", true
	case "go/printer":
		return "go/printer", true
	case "go/scanner":
		return "go/scanner", true
	case "go/token":
		return "go/token", true
	case "go/types":
		return "go/types", true
	case "go/version":
		return "go/version", true
	case "hash":
		return "hash", true
	case "hash/adler32":
		return "hash/adler32", true
	case "hash/crc32":
		return "hash/crc32", true
	case "hash/crc64":
		return "hash/crc64", true
	case "hash/fnv":
		return "hash/fnv", true
	case "hash/maphash":
		return "hash/maphash", true
	case "html":
		return "html", true
	case "html/template":
		return "html/template", true
	case "image":
		return "image", true
	case "image/color":
		return "image/color", true
	case "image/color/palette":
		return "image/color/palette", true
	case "image/draw":
		return "image/draw", true
	case "image/gif":
		return "image/gif", true
	case "image/internal/imageutil":
		return "image/internal/imageutil", true
	case "image/jpeg":
		return "image/jpeg", true
	case "image/png":
		return "image/png", true
	case "index/suffixarray":
		return "index/suffixarray", true
	case "internal/abi":
		return "internal/abi", true
	case "internal/bisect":
		return "internal/bisect", true
	case "internal/buildcfg":
		return "internal/buildcfg", true
	case "internal/bytealg":
		return "internal/bytealg", true
	case "internal/cfg":
		return "internal/cfg", true
	case "internal/chacha8rand":
		return "internal/chacha8rand", true
	case "internal/coverage":
		return "internal/coverage", true
	case "internal/coverage/calloc":
		return "internal/coverage/calloc", true
	case "internal/coverage/cformat":
		return "internal/coverage/cformat", true
	case "internal/coverage/cmerge":
		return "internal/coverage/cmerge", true
	case "internal/coverage/decodecounter":
		return "internal/coverage/decodecounter", true
	case "internal/coverage/decodemeta":
		return "internal/coverage/decodemeta", true
	case "internal/coverage/encodecounter":
		return "internal/coverage/encodecounter", true
	case "internal/coverage/encodemeta":
		return "internal/coverage/encodemeta", true
	case "internal/coverage/pods":
		return "internal/coverage/pods", true
	case "internal/coverage/rtcov":
		return "internal/coverage/rtcov", true
	case "internal/coverage/slicereader":
		return "internal/coverage/slicereader", true
	case "internal/coverage/slicewriter":
		return "internal/coverage/slicewriter", true
	case "internal/coverage/stringtab":
		return "internal/coverage/stringtab", true
	case "internal/coverage/test":
		return "internal/coverage/test", true
	case "internal/coverage/uleb128":
		return "internal/coverage/uleb128", true
	case "internal/cpu":
		return "internal/cpu", true
	case "internal/dag":
		return "internal/dag", true
	case "internal/diff":
		return "internal/diff", true
	case "internal/fmtsort":
		return "internal/fmtsort", true
	case "internal/fuzz":
		return "internal/fuzz", true
	case "internal/goarch":
		return "internal/goarch", true
	case "internal/godebug":
		return "internal/godebug", true
	case "internal/godebugs":
		return "internal/godebugs", true
	case "internal/goexperiment":
		return "internal/goexperiment", true
	case "internal/goos":
		return "internal/goos", true
	case "internal/goroot":
		return "internal/goroot", true
	case "internal/gover":
		return "internal/gover", true
	case "internal/goversion":
		return "internal/goversion", true
	case "internal/intern":
		return "internal/intern", true
	case "internal/itoa":
		return "internal/itoa", true
	case "internal/lazyregexp":
		return "internal/lazyregexp", true
	case "internal/lazytemplate":
		return "internal/lazytemplate", true
	case "internal/nettrace":
		return "internal/nettrace", true
	case "internal/obscuretestdata":
		return "internal/obscuretestdata", true
	case "internal/oserror":
		return "internal/oserror", true
	case "internal/pkgbits":
		return "internal/pkgbits", true
	case "internal/platform":
		return "internal/platform", true
	case "internal/poll":
		return "internal/poll", true
	case "internal/profile":
		return "internal/profile", true
	case "internal/race":
		return "internal/race", true
	case "internal/reflectlite":
		return "internal/reflectlite", true
	case "internal/safefilepath":
		return "internal/safefilepath", true
	case "internal/saferio":
		return "internal/saferio", true
	case "internal/singleflight":
		return "internal/singleflight", true
	case "internal/syscall/execenv":
		return "internal/syscall/execenv", true
	case "internal/syscall/unix":
		return "internal/syscall/unix", true
	case "internal/sysinfo":
		return "internal/sysinfo", true
	case "internal/testenv":
		return "internal/testenv", true
	case "internal/testlog":
		return "internal/testlog", true
	case "internal/testpty":
		return "internal/testpty", true
	case "internal/trace":
		return "internal/trace", true
	case "internal/trace/traceviewer":
		return "internal/trace/traceviewer", true
	case "internal/trace/traceviewer/format":
		return "internal/trace/traceviewer/format", true
	case "internal/trace/v2":
		return "internal/trace/v2", true
	case "internal/trace/v2/event":
		return "internal/trace/v2/event", true
	case "internal/trace/v2/event/go122":
		return "internal/trace/v2/event/go122", true
	case "internal/trace/v2/internal/testgen/go122":
		return "internal/trace/v2/internal/testgen/go122", true
	case "internal/trace/v2/raw":
		return "internal/trace/v2/raw", true
	case "internal/trace/v2/testtrace":
		return "internal/trace/v2/testtrace", true
	case "internal/trace/v2/version":
		return "internal/trace/v2/version", true
	case "internal/txtar":
		return "internal/txtar", true
	case "internal/types/errors":
		return "internal/types/errors", true
	case "internal/unsafeheader":
		return "internal/unsafeheader", true
	case "internal/xcoff":
		return "internal/xcoff", true
	case "internal/zstd":
		return "internal/zstd", true
	case "io":
		return "io", true
	case "io/fs":
		return "io/fs", true
	case "io/ioutil":
		return "io/ioutil", true
	case "log":
		return "log", true
	case "log/internal":
		return "log/internal", true
	case "log/slog":
		return "log/slog", true
	case "log/slog/internal":
		return "log/slog/internal", true
	case "log/slog/internal/benchmarks":
		return "log/slog/internal/benchmarks", true
	case "log/slog/internal/buffer":
		return "log/slog/internal/buffer", true
	case "log/slog/internal/slogtest":
		return "log/slog/internal/slogtest", true
	case "log/syslog":
		return "log/syslog", true
	case "maps":
		return "maps", true
	case "math":
		return "math", true
	case "math/big":
		return "math/big", true
	case "math/bits":
		return "math/bits", true
	case "math/cmplx":
		return "math/cmplx", true
	case "math/rand":
		return "math/rand", true
	case "math/rand/v2":
		return "math/rand/v2", true
	case "mime":
		return "mime", true
	case "mime/multipart":
		return "mime/multipart", true
	case "mime/quotedprintable":
		return "mime/quotedprintable", true
	case "net":
		return "net", true
	case "net/http":
		return "net/http", true
	case "net/http/cgi":
		return "net/http/cgi", true
	case "net/http/cookiejar":
		return "net/http/cookiejar", true
	case "net/http/fcgi":
		return "net/http/fcgi", true
	case "net/http/httptest":
		return "net/http/httptest", true
	case "net/http/httptrace":
		return "net/http/httptrace", true
	case "net/http/httputil":
		return "net/http/httputil", true
	case "net/http/internal":
		return "net/http/internal", true
	case "net/http/internal/ascii":
		return "net/http/internal/ascii", true
	case "net/http/internal/testcert":
		return "net/http/internal/testcert", true
	case "net/http/pprof":
		return "net/http/pprof", true
	case "net/internal/socktest":
		return "net/internal/socktest", true
	case "net/mail":
		return "net/mail", true
	case "net/netip":
		return "net/netip", true
	case "net/rpc":
		return "net/rpc", true
	case "net/rpc/jsonrpc":
		return "net/rpc/jsonrpc", true
	case "net/smtp":
		return "net/smtp", true
	case "net/textproto":
		return "net/textproto", true
	case "net/url":
		return "net/url", true
	case "os":
		return "os", true
	case "os/exec":
		return "os/exec", true
	case "os/exec/internal/fdtest":
		return "os/exec/internal/fdtest", true
	case "os/signal":
		return "os/signal", true
	case "os/user":
		return "os/user", true
	case "path":
		return "path", true
	case "path/filepath":
		return "path/filepath", true
	case "plugin":
		return "plugin", true
	case "reflect":
		return "reflect", true
	case "reflect/internal/example1":
		return "reflect/internal/example1", true
	case "reflect/internal/example2":
		return "reflect/internal/example2", true
	case "regexp":
		return "regexp", true
	case "regexp/syntax":
		return "regexp/syntax", true
	case "runtime":
		return "runtime", true
	case "runtime/cgo":
		return "runtime/cgo", true
	case "runtime/coverage":
		return "runtime/coverage", true
	case "runtime/debug":
		return "runtime/debug", true
	case "runtime/internal/atomic":
		return "runtime/internal/atomic", true
	case "runtime/internal/math":
		return "runtime/internal/math", true
	case "runtime/internal/startlinetest":
		return "runtime/internal/startlinetest", true
	case "runtime/internal/sys":
		return "runtime/internal/sys", true
	case "runtime/internal/syscall":
		return "runtime/internal/syscall", true
	case "runtime/internal/wasitest":
		return "runtime/internal/wasitest", true
	case "runtime/metrics":
		return "runtime/metrics", true
	case "runtime/pprof":
		return "runtime/pprof", true
	case "runtime/race":
		return "runtime/race", true
	case "runtime/race/internal/amd64v1":
		return "runtime/race/internal/amd64v1", true
	case "runtime/trace":
		return "runtime/trace", true
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
	case "sync/atomic":
		return "sync/atomic", true
	case "syscall":
		return "syscall", true
	case "testing":
		return "testing", true
	case "testing/fstest":
		return "testing/fstest", true
	case "testing/internal/testdeps":
		return "testing/internal/testdeps", true
	case "testing/iotest":
		return "testing/iotest", true
	case "testing/quick":
		return "testing/quick", true
	case "testing/slogtest":
		return "testing/slogtest", true
	case "text/scanner":
		return "text/scanner", true
	case "text/tabwriter":
		return "text/tabwriter", true
	case "text/template":
		return "text/template", true
	case "text/template/parse":
		return "text/template/parse", true
	case "time":
		return "time", true
	case "time/tzdata":
		return "time/tzdata", true
	case "unicode":
		return "unicode", true
	case "unicode/utf16":
		return "unicode/utf16", true
	case "unicode/utf8":
		return "unicode/utf8", true
	case "unsafe":
		return "unsafe", true
	default:
		return "", false
	}
}
