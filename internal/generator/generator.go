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

func New(templateText string) (*Generator, error) {
	const templateName = "errgen"
	temp, err := template.
		New(templateName).
		Parse(templateText)
	if err != nil {
		return nil, model.NewGenInvalidTemplateError(templateName, err)
	}
	return &Generator{temp: temp}, nil
}

// Generate renders Go source for the given error definitions
// srcPkg is the source package qualifier (e.g. "case3") when generating into
// a different package; empty string when output is in the same package
// srcImport is the full import path of the source package
func (g *Generator) Generate(packageName string, defs []model.ErrDef, srcPkg, srcImport string, noHooks bool) ([]byte, error) {
	var tmplDefs []errDefData

	for _, def := range defs {
		dd, err := buildDefData(def)
		if err != nil {
			return nil, model.NewGenErrDefError(def.Name, err)
		}
		// Qualify sentinel var names with source package when cross-package
		if srcPkg != "" {
			dd.VarName = srcPkg + "." + dd.VarName
		}
		tmplDefs = append(tmplDefs, dd)
	}

	// Collect unique extra imports from fields and determine which std imports are needed
	seen := make(map[string]bool)
	var imports []string
	var needsFmt, needsSlog, needsJSON, needsErrors bool
	for _, d := range tmplDefs {
		if d.ErrorFormat != nil {
			needsFmt = true
		}
		if len(d.Fields) > 0 {
			needsSlog = true
			needsJSON = true
		}
		if len(d.WrappedFields) > 0 {
			needsErrors = true
		}
		for _, f := range d.Fields {
			if f.ImportPath != "" && !seen[f.ImportPath] {
				seen[f.ImportPath] = true
				imports = append(imports, f.ImportPath)
			}
		}
	}

	// Add source package import when generating into a different package
	if srcImport != "" && !seen[srcImport] {
		imports = append(imports, srcImport)
	}

	data := templateData{
		PackageName: packageName,
		Imports:     imports,
		Defs:        tmplDefs,
		NeedsFmt:    needsFmt,
		NeedsSlog:   needsSlog,
		NeedsJSON:   needsJSON,
		NeedsErrors: needsErrors,
		NoHooks:     noHooks,
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
