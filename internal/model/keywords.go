package model

// goKeywords is the set of Go reserved keywords that cannot be used as field names
// https://go.dev/ref/spec#Keywords
var GoKeywords = map[string]struct{}{
	"break":       {},
	"case":        {},
	"chan":        {},
	"const":       {},
	"continue":    {},
	"default":     {},
	"defer":       {},
	"else":        {},
	"fallthrough": {},
	"for":         {},
	"func":        {},
	"go":          {},
	"goto":        {},
	"if":          {},
	"import":      {},
	"interface":   {},
	"map":         {},
	"package":     {},
	"range":       {},
	"return":      {},
	"select":      {},
	"struct":      {},
	"switch":      {},
	"type":        {},
	"var":         {},
}

var ErrorMethods = map[string]struct{}{
	"Error":  {},
	"Is":     {},
	"Unwrap": {},
	"error":  {}, // reserved: auto-emitted as the "error" key by MarshalJSON / LogValue / MarshalLogObject
}
