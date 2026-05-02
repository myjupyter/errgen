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
)

type Generator struct {
	temp *template.Template
}

// templateFuncs are exposed to the template so it can match Go field types
// to the right slog/zap constructor instead of using a generic slog.Any /
// zap.Any for everything.
var templateFuncs = template.FuncMap{
	"zapMethod":    zapEncoderMethod,
	"slogMethod":   slogConstructorMethod,
	"slogValueOf":  slogValueExpr,
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
}

// zapEncoderMethod returns the zapcore.ObjectEncoder method name for a given
// Go field type. Returns "Any" for types without a direct encoder method —
// the template uses that to fall back to zap.Any(key, val).AddTo(enc).
func zapEncoderMethod(goType string) string {
	switch goType {
	case "bool":
		return "AddBool"
	case "string":
		return "AddString"
	case "int":
		return "AddInt"
	case "int8":
		return "AddInt8"
	case "int16":
		return "AddInt16"
	case "int32":
		return "AddInt32"
	case "int64":
		return "AddInt64"
	case "uint":
		return "AddUint"
	case "uint8":
		return "AddUint8"
	case "uint16":
		return "AddUint16"
	case "uint32":
		return "AddUint32"
	case "uint64":
		return "AddUint64"
	case "uintptr":
		return "AddUintptr"
	case "float32":
		return "AddFloat32"
	case "float64":
		return "AddFloat64"
	case "complex64":
		return "AddComplex64"
	case "complex128":
		return "AddComplex128"
	case "time.Duration":
		return "AddDuration"
	case "time.Time":
		return "AddTime"
	case "[]byte":
		return "AddBinary"
	default:
		return "Any"
	}
}

// slogConstructorMethod returns the slog package constructor name for a given
// Go field type (e.g. "Int", "String", "Time"). Returns "Any" for types without
// a direct constructor — slog.Any handles them via reflection.
func slogConstructorMethod(goType string) string {
	switch goType {
	case "bool":
		return "Bool"
	case "string":
		return "String"
	case "int":
		return "Int"
	case "int8", "int16", "int32", "int64":
		return "Int64"
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "Uint64"
	case "float32", "float64":
		return "Float64"
	case "time.Time":
		return "Time"
	case "time.Duration":
		return "Duration"
	default:
		return "Any"
	}
}

// slogValueExpr returns the value expression to pass to the slog constructor
// for a field of the given type. Most types pass through as e.<Name>; widening
// types (int8/16/32 -> int64, uint variants -> uint64, float32 -> float64)
// require an explicit cast so the call resolves to the typed constructor.
func slogValueExpr(goType, fieldName string) string {
	base := "e." + fieldName
	switch goType {
	case "int8", "int16", "int32":
		return "int64(" + base + ")"
	case "uint", "uint8", "uint16", "uint32", "uintptr":
		return "uint64(" + base + ")"
	case "float32":
		return "float64(" + base + ")"
	default:
		return base
	}
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

	// Collect unique extra imports from fields and determine which std imports are needed
	seen := make(map[string]bool)
	var imports []string
	var needsFmt, needsSlog, needsJSON, needsErrors, needsHTTPStatus bool
	zapData := zapTemplateData{Enabled: in.Zap}
	for _, d := range tmplDefs {
		if d.ErrorFormat != nil {
			needsFmt = true
		}
		if len(d.Fields) > 0 {
			needsSlog = true
			needsJSON = true
			if in.Zap {
				zapData.NeedsZapcore = true
			}
		}
		if len(d.WrappedFields) > 0 {
			needsErrors = true
		}
		if d.Code != nil {
			needsHTTPStatus = true
		}
		for _, f := range d.Fields {
			if f.ImportPath != "" && !seen[f.ImportPath] {
				seen[f.ImportPath] = true
				imports = append(imports, f.ImportPath)
			}
			if in.Zap && f.Type != "error" && zapEncoderMethod(f.Type) == "Any" {
				zapData.NeedsZap = true
			}
		}
	}

	// Add source package import when generating into a different package
	if in.SrcImport != "" && !seen[in.SrcImport] {
		imports = append(imports, in.SrcImport)
	}

	data := templateData{
		PackageName:     in.PackageName,
		Imports:         imports,
		Defs:            tmplDefs,
		NeedsFmt:        needsFmt,
		NeedsSlog:       needsSlog,
		NeedsJSON:       needsJSON,
		NeedsErrors:     needsErrors,
		NeedsHTTPStatus: needsHTTPStatus,
		StackTrace:      in.StackTrace,
		NoHooks:         in.NoHooks,
		Zap:             zapData,
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
