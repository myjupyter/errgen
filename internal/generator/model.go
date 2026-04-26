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
