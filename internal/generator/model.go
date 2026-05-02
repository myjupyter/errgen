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
	Zerolog         zerologTemplateData
}

// zapTemplateData groups data used by the zap-related template sections
type zapTemplateData struct {
	Enabled      bool // -zap flag
	NeedsZapcore bool // emit "go.uber.org/zap/zapcore" import
	NeedsZap     bool // emit "go.uber.org/zap" import (only when zap.Any fallback is used)
}

// zerologTemplateData groups data used by the zerolog-related template sections
type zerologTemplateData struct {
	Enabled      bool // -zerolog flag
	NeedsZerolog bool // emit "github.com/rs/zerolog" import
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
}
