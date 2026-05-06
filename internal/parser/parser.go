package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/myjupyter/errgen/internal/model"
)

// Parse reads a Go source file and returns parsed file info
// with package name and annotated error definitions
func Parse(filename string) (*model.FileInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, model.NewParsingFileError(filename, err)
	}

	info := &model.FileInfo{
		PackageName: f.Name.Name,
	}

	for _, decl := range f.Decls {
		// Skip if it's not a global var declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Only process vars that are assigned errors.New(...) or fmt.Errorf(...)
			if !isErrorsNew(valueSpec) && !isFmtErrorf(valueSpec) {
				continue
			}

			// Get the comment group — prefer doc comments (above the var),
			// then inline comments, then the decl's doc (standalone var)
			commentGroup := valueSpec.Doc
			if commentGroup == nil {
				commentGroup = valueSpec.Comment
			}
			if commentGroup == nil {
				commentGroup = genDecl.Doc
			}
			if commentGroup == nil {
				continue
			}

			def, err := parseAnnotations(commentGroup.List)
			if err != nil {
				return nil, model.NewParsingAnnotationError(
					valueSpec.Names[0].Name, err,
				)
			}

			// No annotations found, skip
			if def == nil {
				continue
			}

			def.Name = valueSpec.Names[0].Name

			info.ErrDefs = append(info.ErrDefs, *def)
		}
	}

	return info, nil
}

// isErrorsNew returns true if the var is assigned errors.New(...)
func isErrorsNew(spec *ast.ValueSpec) bool {
	if len(spec.Values) != 1 {
		return false
	}
	call, ok := spec.Values[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "errors" && sel.Sel.Name == "New"
}

func isFmtErrorf(spec *ast.ValueSpec) bool {
	if len(spec.Values) != 1 {
		return false
	}
	call, ok := spec.Values[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "fmt" && sel.Sel.Name == "Errorf"
}

// parseAnnotations processes comment lines and extracts @-annotations
// Returns nil if no annotations are found or an error
func parseAnnotations(comments []*ast.Comment) (*model.ErrDef, error) {
	var def model.ErrDef
	hasAnnotation := false

	for _, c := range comments {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		// Skip if it's not an annotation
		if !strings.HasPrefix(text, "@") {
			continue
		}
		hasAnnotation = true

		if err := parseAnnotation(text, &def); err != nil {
			return nil, err
		}
	}

	if !hasAnnotation {
		return nil, nil
	}
	return &def, nil
}

// parseAnnotation parses a single annotation line like:
//
//	@Code(500)
//	@Is(ErrInternal)
//	@Message string
//	@Error("code: %Code, message: %Message")
func parseAnnotation(text string, def *model.ErrDef) error {
	const (
		errorAnnotationPrefix = "@Error("
		codeAnnotationPrefix  = "@Code("
		isAnnotationPrefix    = "@Is("
	)
	switch {
	case strings.HasPrefix(text, errorAnnotationPrefix):
		return collectErrorAnnotation(text, def)
	case strings.HasPrefix(text, codeAnnotationPrefix):
		return collectCodeAnnotation(text, def)
	case strings.HasPrefix(text, isAnnotationPrefix):
		return collectIsAnnotation(text, def)
	default:
		return collectVarAnnotation(text, def)
	}
}

// codeExprRegex accepts either:
//   - a signed int literal (decimal or hex): 404, -1, 0x1F4
//   - a Go identifier or qualified identifier: MyConst, http.StatusNotFound
var codeExprRegex = regexp.MustCompile(
	`^(-?(0[xX][0-9a-fA-F]+|\d+)|[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?)$`,
)

// idExprRegex accepts a Go identifier or qualified identifier:
// MyErr, pkg.ErrNotFound. It rejects int literals because @Is targets
// must be values of type error.
var idExprRegex = regexp.MustCompile(
	`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`,
)

// regexp for parseVarAnnotation
var (
	varNameRegex  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	typeNameRegex = regexp.MustCompile(`^(\*|\[\]|map\[.*\]\s*)*[\w\.]+$`)
)

func collectCodeAnnotation(text string, def *model.ErrDef) error {
	expr, err := parseCodeAnnotation(text)
	if err != nil {
		return err
	}
	def.Code = &model.CodeDef{Expr: expr}
	return nil
}

// parseCodeAnnotation extracts the code expression from @Code(...)
func parseCodeAnnotation(text string) (string, error) {
	inner := strings.TrimPrefix(text, "@Code(")
	expr := strings.TrimSuffix(strings.TrimSpace(inner), ")")
	expr = strings.TrimSpace(expr)
	if expr == "" || !codeExprRegex.MatchString(expr) {
		return "", model.NewParsingInvalidCodeAnnotationError(text)
	}
	return expr, nil
}

func collectErrorAnnotation(text string, def *model.ErrDef) error {
	format, err := parseErrorAnnotation(text)
	if err != nil {
		return err
	}
	def.ErrorFormat = &format
	return nil
}

// parseErrorAnnotation extracts the format string from @Error("...")
func parseErrorAnnotation(text string) (string, error) {
	// Expect: @Error("some string")
	inner := strings.TrimPrefix(text, "@Error(")
	inner = strings.TrimSuffix(strings.TrimSpace(inner), ")")
	if !strings.HasPrefix(inner, `"`) || !strings.HasSuffix(inner, `"`) {
		return "", model.NewParsingInvalidErrorAnnotationError(text)
	}
	return strings.Trim(inner, `"`), nil
}

func collectIsAnnotation(text string, def *model.ErrDef) error {
	expr, err := parseIsAnnotation(text)
	if err != nil {
		return err
	}
	def.IsTargets = append(def.IsTargets, model.IsDef{Expr: expr})
	return nil
}

// parseIsAnnotation extracts the error identifier from @Is(...). The inner
// expression must be a Go identifier or qualified identifier referring to a
// value of type error (e.g. ErrNotFound, pkg.ErrNotFound). Int literals are
// rejected because they could not be compared to an error.
func parseIsAnnotation(text string) (string, error) {
	inner := strings.TrimPrefix(text, "@Is(")
	expr := strings.TrimSuffix(strings.TrimSpace(inner), ")")
	expr = strings.TrimSpace(expr)
	if expr == "" || !idExprRegex.MatchString(expr) {
		return "", model.NewParsingInvalidIsAnnotationError(text)
	}
	return expr, nil
}

func collectVarAnnotation(text string, def *model.ErrDef) error {
	field, err := parseVarAnnotation(text)
	if err != nil {
		return err
	}
	def.Fields = append(def.Fields, field)
	return nil
}

// parseVarAnnotation extracts the field name and type from @Name Type
func parseVarAnnotation(text string) (*model.Field, error) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "@")

	varDecl := make([]string, 0, 2)
	for _, c := range strings.Split(text, " ") {
		if c == "" {
			continue
		}

		varDecl = append(varDecl, c)
	}

	if len(varDecl) != 2 {
		return nil, model.NewParsingInvalidVarAnnotationError(
			text, "expected @<Name> <Type>",
		)
	}

	if !varNameRegex.MatchString(varDecl[0]) {
		return nil, model.NewParsingInvalidVarAnnotationError(
			text, fmt.Sprintf("field name '%s' is invalid", varDecl[0]),
		)
	}

	if _, ok := model.GoKeywords[varDecl[0]]; ok {
		return nil, model.NewParsingInvalidVarAnnotationError(
			text, fmt.Sprintf("field name '%s' is a Go keyword", varDecl[0]),
		)
	}

	if _, ok := model.ErrorMethods[varDecl[0]]; ok {
		return nil, model.NewParsingInvalidVarAnnotationError(
			text, fmt.Sprintf("field name '%s' is a Go error method", varDecl[0]),
		)
	}

	if !typeNameRegex.MatchString(varDecl[1]) {
		return nil, model.NewParsingInvalidVarAnnotationError(
			text, fmt.Sprintf("type name '%s' is invalid", varDecl[1]),
		)
	}

	return &model.Field{
		Name: varDecl[0],
		Type: varDecl[1],
	}, nil
}

// ParseHookTypes parses a hook file and returns the set of type names
// that already have an onCreate() method
func ParseHookTypes(filename string) (map[string]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	types := make(map[string]bool)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "onCreate" || fn.Recv == nil {
			continue
		}
		for _, field := range fn.Recv.List {
			if star, ok := field.Type.(*ast.StarExpr); ok {
				if ident, ok := star.X.(*ast.Ident); ok {
					types[ident.Name] = true
				}
			}
		}
	}
	return types, nil
}
