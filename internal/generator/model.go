package generator

// templateData is the top-level data passed to the template
type templateData struct {
	PackageName string
	Imports     []string
	Defs        []errDefData
	NeedsFmt    bool
	NoHooks     bool
}

// defData is per-error template data
type errDefData struct {
	TypeName        string
	VarName         string
	Fields          []fieldData
	ErrorFormat     *string
	FmtString       string
	ConstructorArgs string
	WrappedFields   []fieldData // fields with type "error", used for Unwrap
}

type fieldData struct {
	Name       string
	NameLower  string
	Type       string
	ImportPath string
}
