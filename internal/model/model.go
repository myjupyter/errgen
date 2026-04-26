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
	Code        *string // gRPC status code name (e.g. "NotFound", "InvalidArgument")
}

type FileInfo struct {
	PackageName string
	ErrDefs     []ErrDef
}
