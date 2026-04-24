package model

type Field struct {
	Name       string
	Type       string
	ImportPath string // full import path, empty for builtin types
}

type ErrDef struct { //nolint:govet // readability over alignment
	Name        string
	Fields      []*Field
	ErrorFormat *string
}

type FileInfo struct {
	PackageName string
	ErrDefs     []ErrDef
}
