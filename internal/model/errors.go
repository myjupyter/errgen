package model

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
	ErrParsing    = errors.New("parsing error")
	ErrGeneration = errors.New("generation error")
	ErrResolving  = errors.New("resolving error")
)

// Parsing error declarations
var (
	// @ErrVarName string
	// @WrappedError error
	// @Is(ErrParsing)
	// @Error("error var '%ErrVarName': %WrappedError")
	ErrParsingAnnotation = errors.New("parsing annotation error")
	// @Filename string
	// @WrappedError error
	// @Is(ErrParsing)
	// @Error("file '%Filename': %WrappedError")
	ErrParsingFile = errors.New("parsing file error")
	// @InvalidAnnotationText string
	// @Error("invalid error annotation '%InvalidAnnotationText': expected: @Error(\"...\")")
	ErrParsingInvalidErrorAnnotation = errors.New("invalid error annotation")
	// @InvalidAnnotationText string
	// @Message string
	// @Error("invalid var annotation '%InvalidAnnotationText': %Message")
	ErrParsingInvalidVarAnnotation = errors.New("invalid var annotation")
	// @InvalidAnnotationText string
	// @Error("invalid code annotation '%InvalidAnnotationText': expected an int literal or a qualified constant like http.StatusNotFound")
	ErrParsingInvalidCodeAnnotation = errors.New("invalid code annotation")
	// @InvalidAnnotationText string
	// @Is(ErrParsing)
	// @Error("invalid is annotation '%InvalidAnnotationText': expected an error identifier like ErrNotFound or pkg.ErrNotFound")
	ErrParsingInvalidIsAnnotation = errors.New("invalid is annotation")
)

// Generation error declarations
var (
	// @TemplateName string
	// @WrappedError error
	// @Is(ErrGeneration)
	// @Error("'%TemplateName' is invalid: %WrappedError")
	ErrGenInvalidTemplate = errors.New("invalid template")
	// @WrappedError error
	// @Is(ErrGeneration)
	// @Error("template execution: %WrappedError")
	ErrGenTemplateExec = errors.New("template execution error")
	// @WrappedError error
	// @Is(ErrGeneration)
	// @Error("generated code formatting: %WrappedError")
	ErrGenCodeFormatting = errors.New("code formatting error")
	// @UnknownField string
	// @Is(ErrGeneration)
	// @Error("no annotation found for '%UnknownField'")
	ErrGenUnknownField = errors.New("unknown field")
	// @ErrVarName string
	// @WrappedError error
	// @Is(ErrGeneration)
	// @Error("err var generation: '%ErrVarName' build error: %WrappedError")
	ErrGenErrDef = errors.New("error definition generation error")
)

// Resolver error declarations
var (
	// @PackageName string
	// @ModulePath string
	// @Is(ErrResolving)
	// @Error("resolver: package '%PackageName' not found in module %ModulePath")
	ErrPackageNotFound = errors.New("package not found")
	// @PackageName string
	// @Locations string
	// @Is(ErrResolving)
	// @Error("resolver: package name '%PackageName' is ambiguous, found in:\n  %Locations\nuse -m flag to specify the import path")
	ErrAmbiguousPackage = errors.New("ambiguous package")
	// @Is(ErrResolving)
	// @Error("resolver: go.mod not found")
	ErrGoModNotFound = errors.New("go.mod not found")
	// @Is(ErrResolving)
	// @Error("resolver: no module directive in go.mod")
	ErrNoModuleDirective = errors.New("no module directive in go.mod")
)
