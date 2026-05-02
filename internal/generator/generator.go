package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"regexp"
	"strings"
	"text/template"

	"github.com/myjupyter/errgen/internal/model"
)

var (
	//go:embed errgen.go.tmpl
	DefaultErrgenTemplate string
	//go:embed errgen.hook.tmpl
	DefaultErrgenHookTemplate string
	//go:embed errgen.zap.tmpl
	defaultErrgenZapTemplate string
	//go:embed errgen.zerolog.tmpl
	defaultErrgenZerologTemplate string
	//go:embed errgen.otel.tmpl
	defaultErrgenOTelTemplate string
)

type Generator struct {
	temp *template.Template
}

// templateFuncs are exposed to the template so each logger section can match
// Go field types to its typed constructor (slog.Int, enc.AddInt, event.Int, …)
// instead of falling through to a generic Any / Interface for everything.
// The mapping tables themselves live in field_matchers.go.
var templateFuncs = template.FuncMap{
	"zapMethod":      zapEncoderMethod,
	"slogMethod":     slogConstructorMethod,
	"slogValueOf":    slogValueExpr,
	"zerologMethod":  zerologEventMethod,
	"zerologValueOf": zerologValueExpr,
	"otelAttribute":  otelAttributeFunc,
	"otelValueOf":    otelValueExpr,
}

func New(templateText string) (*Generator, error) {
	const templateName = "errgen"
	temp, err := template.
		New(templateName).
		Funcs(templateFuncs).
		Parse(templateText)
	if err != nil {
		return nil, model.NewGenInvalidTemplateError(templateName, err)
	}

	extraTemplates := map[string]string{
		"errgen.zap":     defaultErrgenZapTemplate,
		"errgen.zerolog": defaultErrgenZerologTemplate,
		"errgen.otel":    defaultErrgenOTelTemplate,
	}
	for name, text := range extraTemplates {
		if _, err := temp.Parse(text); err != nil {
			return nil, model.NewGenInvalidTemplateError(name, err)
		}
	}

	return &Generator{temp: temp}, nil
}

// GenerateInput holds all parameters for code generation
type GenerateInput struct { //nolint:govet // readability over alignment
	PackageName string
	Defs        []model.ErrDef
	SrcPkg      string // source package qualifier for cross-package generation
	SrcImport   string // full import path of the source package
	NoHooks     bool
	StackTrace  bool
	Zap         bool // emit zapcore.ObjectMarshaler implementation
	Zerolog     bool // emit zerolog.LogObjectMarshaler implementation
	OTel        bool // emit Attributes() []attribute.KeyValue method
}

// Generate renders Go source for the given error definitions
func (g *Generator) Generate(in GenerateInput) ([]byte, error) {
	var tmplDefs []errDefData
	for _, def := range in.Defs {
		dd, err := buildDefData(def)
		if err != nil {
			return nil, model.NewGenErrDefError(def.Name, err)
		}
		// Qualify sentinel var names with source package when cross-package
		if in.SrcPkg != "" {
			dd.VarName = in.SrcPkg + "." + dd.VarName
		}
		tmplDefs = append(tmplDefs, dd)
	}

	flags := aggregateImportFlags(tmplDefs, in)

	data := templateData{
		PackageName:     in.PackageName,
		Imports:         flags.imports,
		Defs:            tmplDefs,
		NeedsFmt:        flags.needsFmt,
		NeedsSlog:       flags.needsSlog,
		NeedsJSON:       flags.needsJSON,
		NeedsErrors:     flags.needsErrors,
		NeedsHTTPStatus: flags.needsHTTPStatus,
		StackTrace:      in.StackTrace,
		NoHooks:         in.NoHooks,
		Zap:             flags.zap,
		Zerolog:         flags.zerolog,
		OTel:            flags.otel,
	}

	var buf bytes.Buffer
	if err := g.temp.Execute(&buf, data); err != nil {
		return nil, model.NewGenTemplateExecError(err)
	}

	// gofmt the output
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return raw output to help debug template issues
		return buf.Bytes(), model.NewGenCodeFormattingError(err)
	}

	return formatted, nil
}

// importFlags collects every per-file boolean and import path that the main
// template needs, computed once per Generate call by aggregateImportFlags.
type importFlags struct { //nolint:govet // readability over alignment
	imports         []string
	needsFmt        bool
	needsSlog       bool
	needsJSON       bool
	needsErrors     bool
	needsHTTPStatus bool
	zap             zapTemplateData
	zerolog         zerologTemplateData
	otel            otelTemplateData
}

// aggregateImportFlags walks every error def once and returns the union of
// imports and the std/std-logger flags the main template uses to gate its
// import block and per-section codegen.
func aggregateImportFlags(defs []errDefData, in GenerateInput) importFlags {
	seen := make(map[string]bool)
	flags := importFlags{
		zap:     zapTemplateData{Enabled: in.Zap},
		zerolog: zerologTemplateData{Enabled: in.Zerolog},
		otel:    otelTemplateData{Enabled: in.OTel},
	}
	for _, d := range defs {
		if d.ErrorFormat != nil {
			flags.needsFmt = true
		}
		if len(d.Fields) > 0 {
			flags.needsSlog = true
			flags.needsJSON = true
			if in.Zap {
				flags.zap.NeedsZapcore = true
			}
			if in.Zerolog {
				flags.zerolog.NeedsZerolog = true
			}
			if in.OTel {
				flags.otel.NeedsAttribute = true
			}
		}
		if len(d.WrappedFields) > 0 {
			flags.needsErrors = true
		}
		if d.Code != nil {
			flags.needsHTTPStatus = true
		}
		for _, f := range d.Fields {
			if f.ImportPath != "" && !seen[f.ImportPath] {
				seen[f.ImportPath] = true
				flags.imports = append(flags.imports, f.ImportPath)
			}
			if in.Zap && f.Type != "error" && zapEncoderMethod(f.Type) == "Any" { //nolint:goconst // sentinel returned by zapEncoderMethod
				flags.zap.NeedsZap = true
			}
			if in.OTel && f.Type != "error" && otelAttributeFunc(f.Type) == "Sprintf" {
				// Sprintf fallback writes attribute.String(k, fmt.Sprintf("%v", v))
				flags.needsFmt = true
			}
		}
	}
	// Source-package import only emitted when generating cross-package
	if in.SrcImport != "" && !seen[in.SrcImport] {
		flags.imports = append(flags.imports, in.SrcImport)
	}
	return flags
}

func buildErrorTypeName(errVarName string) string {
	return strings.TrimPrefix(errVarName, "Err") + "Error"
}

func buildFieldsData(fields []*model.Field) []fieldData {
	var data []fieldData
	for _, f := range fields {
		// if lowercase name is a Go keyword, append "_"
		varNameLowered := strings.ToLower(f.Name[:1]) + f.Name[1:]
		if _, ok := model.GoKeywords[varNameLowered]; ok {
			varNameLowered = varNameLowered + "_"
		}
		data = append(data, fieldData{
			Name:       f.Name,
			NameLower:  varNameLowered,
			Type:       f.Type,
			ImportPath: f.ImportPath,
		})
	}
	return data
}

func buildConstructorArgs(fields []fieldData) string {
	args := make([]string, 0, len(fields))
	for _, f := range fields {
		args = append(args, f.NameLower+" "+f.Type)
	}
	return strings.Join(args, ", ")
}

func buildDefData(def model.ErrDef) (errDefData, error) {
	// build field data
	fieldsData := buildFieldsData(def.Fields)
	// build constructor args: "code int, message string"
	constructorArgs := buildConstructorArgs(fieldsData)
	// Build fmt.Sprintf string from @Error format
	var errorFormat *string
	if def.ErrorFormat != nil && *def.ErrorFormat != "" {
		errFmt, err := buildFmtString(*def.ErrorFormat, fieldsData)
		if err != nil {
			return errDefData{}, err
		}

		errorFormat = &errFmt
	}

	// collect all error-typed fields for Unwrap
	var wrappedFields []fieldData
	for _, f := range fieldsData {
		if f.Type == "error" {
			wrappedFields = append(wrappedFields, f)
		}
	}

	// build type name
	typeName := buildErrorTypeName(def.Name)

	return errDefData{
		TypeName:        typeName,
		VarName:         def.Name,
		Fields:          fieldsData,
		ErrorFormat:     errorFormat,
		ConstructorArgs: constructorArgs,
		WrappedFields:   wrappedFields,
		Code:            def.Code,
	}, nil
}

// buildFmtString converts @Error format string with %Field references
// into a fmt.Sprintf call string
//
// Example:
//
//	"code: %Code, message: %Message" with fields [Code int, Message string]
//	→ `"code: %v, message: %v", e.Code, e.Message`
func buildFmtString(format string, fields []fieldData) (string, error) {
	// Build a lookup of declared field names
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[f.Name] = true
	}

	// Replace %FieldName with %v and collect the field refs in order
	// Use word-boundary matching to avoid partial replacements
	// (e.g. %Err must not match inside %Error)
	var refs []string
	result := format
	for _, f := range fields {
		re := regexp.MustCompile(`%` + regexp.QuoteMeta(f.Name) + `\b`)
		if re.MatchString(result) {
			result = re.ReplaceAllString(result, "%v")
			refs = append(refs, "e."+f.Name)
		}
	}

	// checks for any remaining %Word that don't match declared fields
	err := validateNoUnknownRefs(result, fieldSet)
	if err != nil {
		return "", err
	}

	if len(refs) == 0 {
		return fmt.Sprintf(`"%s"`, result), nil
	}
	return fmt.Sprintf(`"%s", %s`, result, strings.Join(refs, ", ")), nil
}

// validateNoUnknownRefs checks that no %Word remains that wasn't replaced
func validateNoUnknownRefs(result string, fieldSet map[string]bool) error {
	// Walk the result string looking for % followed by an uppercase letter
	for i := 0; i < len(result)-1; i++ {
		if result[i] == '%' && result[i+1] >= 'A' && result[i+1] <= 'Z' {
			// Extract the word
			j := i + 1
			for j < len(result) && (result[j] >= 'A' && result[j] <= 'Z' || result[j] >= 'a' && result[j] <= 'z') {
				j++
			}
			name := result[i+1 : j]
			if !fieldSet[name] {
				return model.NewGenUnknownFieldError(name)
			}
		}
	}
	return nil
}
