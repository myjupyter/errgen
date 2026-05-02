package model

type Field struct {
	Name       string
	Type       string
	ImportPath string // full import path, empty for builtin types
}

// CodeDef is the parsed @Code(...) annotation: the raw expression as written
// (e.g. "404", "http.StatusNotFound", "codes.NotAuthCode") and the resolved
// import path of the package it references (empty for literals or identifiers
// in the same package).
type CodeDef struct {
	Expr       string
	ImportPath string
}

type ErrDef struct { //nolint:govet // readability over alignment
	Name        string
	Fields      []*Field
	ErrorFormat *string
	Code        *CodeDef // nil when no @Code annotation is present
}

type FileInfo struct {
	PackageName string
	ErrDefs     []ErrDef
}
