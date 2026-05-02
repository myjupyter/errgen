package generator

// templateData is the top-level data passed to the template
type templateData struct {
	PackageName string
	Imports     []string
	Defs        []errDefData
	NeedsFmt    bool
	NeedsSlog   bool
	NeedsJSON   bool
	NeedsErrors bool // for errors.New in UnmarshalJSON
	StackTrace  bool
	NoHooks     bool
	Zap         zapTemplateData
	Zerolog     zerologTemplateData
	OTel        otelTemplateData
	Logrus      logrusTemplateData
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

// otelTemplateData groups data used by the OpenTelemetry-related template sections
type otelTemplateData struct {
	Enabled        bool // -otel flag
	NeedsAttribute bool // emit "go.opentelemetry.io/otel/attribute" import
}

// logrusTemplateData groups data used by the logrus-related template sections
type logrusTemplateData struct {
	Enabled     bool // -logrus flag
	NeedsLogrus bool // emit "github.com/sirupsen/logrus" import
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
	Code            *codeData   // nil when no @Code annotation is present
}

// codeData is the per-error template view of a parsed @Code(...) annotation.
type codeData struct {
	Expr       string // raw expression as written: "404", "http.StatusNotFound", ...
	ImportPath string // resolved import path of the package referenced by Expr
}

type fieldData struct {
	Name       string
	NameLower  string
	Type       string
	ImportPath string
}
