package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/myjupyter/errgen/internal/generator"
	"github.com/myjupyter/errgen/internal/model"
	"github.com/myjupyter/errgen/internal/parser"
	"github.com/myjupyter/errgen/internal/resolver"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

// importMapFlag implements flag.Value for repeatable -m pkg=import/path flags
type importMapFlag map[string]string

func (m *importMapFlag) String() string { return fmt.Sprintf("%v", map[string]string(*m)) }

func (m *importMapFlag) Set(val string) error {
	parts := strings.SplitN(val, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("expected pkg=import/path, got %q", val)
	}
	if *m == nil {
		*m = make(importMapFlag)
	}
	(*m)[parts[0]] = parts[1]
	return nil
}

const (
	genFileSuffix  = "_gen"
	hookFileSuffix = "_gen_hook"
)

type config struct { //nolint:govet // readability over alignment
	packageName    string
	inputFile      string
	outputPath     string
	hookOutputPath string
	templatePath   string
	dryRun         bool
	noHooks        bool
	stackTrace     bool
	zap            bool
	zerolog        bool
	otel           bool
	manualImports  importMapFlag
}

func main() {
	cfg, exit := parseConfig()
	if exit {
		return
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "errgen: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig() (*config, bool) {
	packageName := flag.String("p", "", "package name for the generated file (default: $GOPACKAGE)")
	outputPathValue := flag.String("o", "", "output file path (default: <input>_gen.go)")
	templatePath := flag.String("t", "", "path to a custom Go template file (default: built-in template)")
	dryRun := flag.Bool("n", false, "dry run: print generated code to stdout instead of writing a file")
	showVersion := flag.Bool("v", false, "print version and exit")
	noHooks := flag.Bool("no-hooks", false, "skip hook file generation")
	stackTrace := flag.Bool("stack-trace", false, "capture stack trace in constructors via runtime.Callers")
	zap := flag.Bool("zap", false, "generate zapcore.ObjectMarshaler implementation for use with go.uber.org/zap")
	zerolog := flag.Bool("zerolog", false, "generate zerolog.LogObjectMarshaler implementation for use with github.com/rs/zerolog")
	otel := flag.Bool("otel", false, "generate Attributes() []attribute.KeyValue method for use with go.opentelemetry.io/otel")

	var manualImports importMapFlag
	flag.Var(&manualImports, "m", "manual import mapping: pkg=import/path (repeatable, for ambiguous packages)")

	flag.Parse()

	if *showVersion {
		fmt.Println("errgen " + version)
		return nil, true
	}

	inputFile := flag.Arg(0)
	if inputFile == "" {
		inputFile = os.Getenv("GOFILE")
	}
	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "usage: errgen [flags] [source-file]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg := &config{
		packageName:   *packageName,
		inputFile:     inputFile,
		templatePath:  *templatePath,
		dryRun:        *dryRun,
		noHooks:       *noHooks,
		stackTrace:    *stackTrace,
		zap:           *zap,
		zerolog:       *zerolog,
		otel:          *otel,
		manualImports: manualImports,
	}

	// Compute output paths
	if outputPathValue == nil || *outputPathValue == "" {
		ext := filepath.Ext(inputFile)
		base := strings.TrimSuffix(inputFile, ext)
		cfg.outputPath = base + genFileSuffix + ext
		cfg.hookOutputPath = base + hookFileSuffix + ext
	} else {
		cfg.outputPath = *outputPathValue
		dir := filepath.Dir(*outputPathValue)
		name := filepath.Base(*outputPathValue)
		ext := filepath.Ext(*outputPathValue)
		cfg.hookOutputPath = filepath.Join(dir, strings.TrimSuffix(name, ext)+hookFileSuffix+ext)
	}

	// Package name: flag or $GOPACKAGE (source file fallback applied after parse)
	if cfg.packageName == "" {
		cfg.packageName = os.Getenv("GOPACKAGE")
	}

	return cfg, false
}

func run(cfg *config) error {
	fileInfo, err := parser.Parse(cfg.inputFile)
	if err != nil {
		return err
	}

	if len(fileInfo.ErrDefs) == 0 {
		return nil
	}

	// Warn about variable names that don't start with "Err"
	for _, def := range fileInfo.ErrDefs {
		if !strings.HasPrefix(def.Name, "Err") {
			fmt.Fprintf(os.Stderr, "warning: %q does not start with \"Err\" — generated type name will be %q\n",
				def.Name, strings.TrimPrefix(def.Name, "Err")+"Error")
		}
		if strings.HasSuffix(def.Name, "Error") {
			fmt.Fprintf(os.Stderr, "warning: %q ends with \"Error\" — generated type name will be %q\n",
				def.Name, strings.TrimPrefix(def.Name, "Err")+"Error")
		}
	}

	// Package name priority: flag > $GOPACKAGE > parsed from source file
	if cfg.packageName == "" {
		cfg.packageName = fileInfo.PackageName
	}

	if err := resolveImports(fileInfo, cfg.manualImports, cfg.inputFile); err != nil { //nolint:govet // shadow is intentional
		return err
	}

	// Detect cross-package generation
	srcPkg, srcImport, err := detectCrossPackage(cfg.inputFile, cfg.outputPath, fileInfo.PackageName)
	if err != nil {
		return err
	}

	templateText, err := loadTemplate(cfg.templatePath)
	if err != nil {
		return err
	}

	// Generate main file
	genInput := generator.GenerateInput{
		PackageName: cfg.packageName,
		Defs:        fileInfo.ErrDefs,
		SrcPkg:      srcPkg,
		SrcImport:   srcImport,
		NoHooks:     cfg.noHooks,
		StackTrace:  cfg.stackTrace,
		Zap:         cfg.zap,
		Zerolog:     cfg.zerolog,
		OTel:        cfg.otel,
	}
	if err := generateFile(templateText, cfg.outputPath, genInput, cfg.dryRun); err != nil {
		return err
	}

	// Generate hook file
	if !cfg.noHooks {
		if err := generateHookFile(cfg.hookOutputPath, cfg.packageName, fileInfo.ErrDefs, cfg.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func resolveImports(fileInfo *model.FileInfo, manualImports importMapFlag, inputFile string) error {
	var allTypes []string
	for _, def := range fileInfo.ErrDefs {
		for _, f := range def.Fields {
			allTypes = append(allTypes, f.Type)
		}
	}

	pkgNames := resolver.ExtractPackageNames(allTypes)
	if len(pkgNames) == 0 {
		return nil
	}

	// Separate manually mapped packages from those needing resolution
	resolved := make(map[string]string)
	var unresolvedPkgs []string
	for _, name := range pkgNames {
		if importPath, ok := manualImports[name]; ok {
			resolved[name] = importPath
		} else {
			unresolvedPkgs = append(unresolvedPkgs, name)
		}
	}

	if len(unresolvedPkgs) > 0 {
		res, err := resolver.New(filepath.Dir(inputFile))
		if err != nil {
			return err
		}
		autoResolved, err := res.Resolve(unresolvedPkgs)
		if err != nil {
			return err
		}
		for k, v := range autoResolved {
			resolved[k] = v
		}
	}

	// Assign import paths back to fields
	for i := range fileInfo.ErrDefs {
		for j, f := range fileInfo.ErrDefs[i].Fields {
			pkgName := resolver.ExtractPkgName(f.Type)
			if pkgName != "" {
				fileInfo.ErrDefs[i].Fields[j].ImportPath = resolved[pkgName]
			}
		}
	}

	return nil
}

func detectCrossPackage(inputFile, outputPath, srcPkgName string) (string, string, error) {
	absInputDir, err := filepath.Abs(filepath.Dir(inputFile))
	if err != nil {
		return "", "", fmt.Errorf("abs path: %w", err)
	}
	absOutputDir, err := filepath.Abs(filepath.Dir(outputPath))
	if err != nil {
		return "", "", fmt.Errorf("abs path: %w", err)
	}
	if absOutputDir == absInputDir {
		return "", "", nil
	}

	srcImport, err := resolver.ImportPathForDir(filepath.Dir(inputFile))
	if err != nil {
		return "", "", err
	}
	return srcPkgName, srcImport, nil
}

func loadTemplate(templatePath string) (string, error) {
	if templatePath == "" {
		return generator.DefaultErrgenTemplate, nil
	}
	data, err := os.ReadFile(templatePath) //nolint:gosec // path from CLI flag
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}
	return string(data), nil
}

func generateFile(templateText, outputPath string, in generator.GenerateInput, dryRun bool) error {
	gen, err := generator.New(templateText)
	if err != nil {
		return err
	}

	out, err := gen.Generate(in)
	if err != nil {
		return err
	}

	if dryRun {
		os.Stdout.Write(out) //nolint:errcheck,gosec // stdout write errors are not actionable
		return nil
	}

	return os.WriteFile(outputPath, out, 0644) //nolint:gosec // 0644 is correct for generated Go source
}

func generateHookFile(hookPath, packageName string, defs []model.ErrDef, dryRun bool) error {
	// If hook file exists, append stubs only for new error types
	_, statErr := os.Stat(hookPath)
	if statErr == nil {
		return appendNewHooks(hookPath, defs)
	}
	if !os.IsNotExist(statErr) {
		return statErr
	}

	// Hook file doesn't exist - generate full file
	gen, err := generator.New(generator.DefaultErrgenHookTemplate)
	if err != nil {
		return err
	}

	out, err := gen.Generate(generator.GenerateInput{
		PackageName: packageName,
		Defs:        defs,
	})
	if err != nil {
		return err
	}

	if dryRun {
		os.Stdout.Write(out) //nolint:errcheck,gosec // stdout write errors are not actionable
		return nil
	}

	return os.WriteFile(hookPath, out, 0644) //nolint:gosec // 0644 is correct for generated Go source
}

func appendNewHooks(hookPath string, defs []model.ErrDef) error {
	existing, err := parser.ParseHookTypes(hookPath)
	if err != nil {
		return fmt.Errorf("parsing hook file: %w", err)
	}

	var newStubs []string
	for _, def := range defs {
		typeName := strings.TrimPrefix(def.Name, "Err") + "Error"
		if !existing[typeName] {
			newStubs = append(newStubs, fmt.Sprintf(
				"\n// onCreate is a hook for user custom logic\n// the code inside must not panic\nfunc (e *%s) onCreate() {\n\t// put custom logic here\n}\n",
				typeName,
			))
		}
	}

	if len(newStubs) == 0 {
		return nil
	}

	f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0644) //nolint:gosec // path from CLI args, 0644 is correct for Go source
	if err != nil {
		return err
	}

	for _, stub := range newStubs {
		if _, err := f.WriteString(stub); err != nil {
			return errors.Join(err, f.Close())
		}
	}
	return f.Close()
}
