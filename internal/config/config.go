package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DryRun         bool
	NoHooks        bool
	StackTrace     bool
	ShowVersion    bool
	PackageName    string
	InputFile      string
	OutputPath     string
	HookOutputPath string
	TemplatePath   string
	ManualImports  importMapFlag
}

func New() (*Config, error) {
	packageName := flag.String("p", os.Getenv("GOPACKAGE"), "package name for the generated file (default: $GOPACKAGE)")
	outputPathValue := flag.String("o", "", "output file path (default: <input>_gen.go)")
	templatePath := flag.String("t", "", "path to a custom Go template file (default: built-in template)")
	dryRun := flag.Bool("n", false, "dry run: print generated code to stdout instead of writing a file")
	showVersion := flag.Bool("v", false, "print version and exit")
	noHooks := flag.Bool("no-hooks", false, "skip hook file generation")
	stackTrace := flag.Bool("stack-trace", false, "capture stack trace in constructors via runtime.Callers")
	var manualImports importMapFlag
	flag.Var(&manualImports, "m", "manual import mapping: pkg=import/path (repeatable, for ambiguous packages)")

	flag.Parse()

	if *showVersion {
		return &Config{ShowVersion: true}, nil
	}

	inputFile := flag.Arg(0)
	if inputFile == "" {
		inputFile = os.Getenv("GOFILE")
	}
	if inputFile == "" {
		return nil, NewNoInputFileSpecifiedError()
	}

	if *packageName == "" {
		return nil, ErrNoPackageSpecified
	}

	cfg := &Config{
		PackageName:   *packageName,
		InputFile:     inputFile,
		TemplatePath:  *templatePath,
		DryRun:        *dryRun,
		NoHooks:       *noHooks,
		StackTrace:    *stackTrace,
		ShowVersion:   *showVersion,
		ManualImports: manualImports,
	}

	const genFileSuffix = "_gen"
	const hookFileSuffix = "_gen_hook"

	path := inputFile
	var pathOverriden bool
	if outputPathValue != nil && *outputPathValue != "" {
		path = *outputPathValue
		pathOverriden = true
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	cfg.OutputPath = base + genFileSuffix + ext
	if pathOverriden {
		cfg.OutputPath = path
	}
	cfg.HookOutputPath = base + hookFileSuffix + ext

	return cfg, nil
}
