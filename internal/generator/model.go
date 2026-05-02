package generator

// templateData is the top-level data passed to the template
type templateData struct {
	PackageName     string
	Imports         []string
	Defs            []errDefData
	NeedsFmt        bool
	NeedsSlog       bool
	NeedsJSON       bool
	NeedsErrors     bool // for errors.New in UnmarshalJSON
	NeedsHTTPStatus bool
	StackTrace      bool
	NoHooks         bool
	Zap             zapTemplateData
}

// zapTemplateData groups data used by the zap-related template sections
type zapTemplateData struct {
	Enabled      bool // -zap flag
	NeedsZapcore bool // emit "go.uber.org/zap/zapcore" import
	NeedsZap     bool // emit "go.uber.org/zap" import (only when zap.Any fallback is used)
}

// defData is per-error template data
type errDefData struct { //nolint:govet // readability over alignment
	TypeName        string
	VarName         string
	Fields          []fieldData
	ErrorFormat     *string
	FmtString       string
	ConstructorArgs string
	WrappedFields   []fieldData // fields with type "error", used for Unwrap
	Code            *string     // status code expression from @Code(...)
}

type fieldData struct {
	Name       string
	NameLower  string
	Type       string
	ImportPath string
	// ZapMethod is the zapcore.ObjectEncoder method name to call for this
	// field (e.g. "AddInt", "AddString"). Empty when the field type has no
	// direct encoder method and the template should fall back to zap.Any.
	// Not used for error-typed fields, which the template handles inline.
	ZapMethod string
}
