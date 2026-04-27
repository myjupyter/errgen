package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/myjupyter/errgen/internal/config"
	"github.com/myjupyter/errgen/internal/generator"
	"github.com/myjupyter/errgen/internal/model"
	"github.com/myjupyter/errgen/internal/parser"
	"github.com/myjupyter/errgen/internal/resolver"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "errgen: %v\n", err)
		flag.PrintDefaults()
		os.Exit(1)
	}

	if cfg.ShowVersion {
		fmt.Println("errgen " + version)
		return
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "errgen: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	fileInfo, err := parser.Parse(cfg.InputFile)
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
	if cfg.PackageName == "" {
		cfg.PackageName = fileInfo.PackageName
	}

	if err := resolveImports(fileInfo, cfg.ManualImports, cfg.InputFile); err != nil { //nolint:govet // shadow is intentional
		return err
	}

	// Detect cross-package generation
	srcPkg, srcImport, err := detectCrossPackage(cfg.InputFile, cfg.OutputPath, fileInfo.PackageName)
	if err != nil {
		return err
	}

	templateText, err := loadTemplate(cfg.TemplatePath)
	if err != nil {
		return err
	}

	// Generate main file
	genInput := generator.GenerateInput{
		PackageName: cfg.PackageName,
		Defs:        fileInfo.ErrDefs,
		SrcPkg:      srcPkg,
		SrcImport:   srcImport,
		NoHooks:     cfg.NoHooks,
		StackTrace:  cfg.StackTrace,
	}
	if err := generateFile(templateText, cfg.OutputPath, genInput, cfg.DryRun); err != nil {
		return err
	}

	// Generate hook file
	if !cfg.NoHooks {
		if err := generateHookFile(cfg.HookOutputPath, cfg.PackageName, fileInfo.ErrDefs, cfg.DryRun); err != nil {
			return err
		}
	}

	return nil
}

func resolveImports(fileInfo *model.FileInfo, manualImports map[string]string, inputFile string) error {
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
