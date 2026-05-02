package generator

// This file holds the type-matching tables that drive structured-logger
// codegen. Each matcher is a pure function from a Go field type to either a
// logger-specific method name or a value expression.
//
// The case clauses are Go type names — not magic strings — so goconst is
// disabled for this file via .golangci.yaml.

// zapEncoderMethod returns the zapcore.ObjectEncoder method name for a given
// Go field type. Returns "Any" for types without a direct encoder method —
// the template uses that to fall back to zap.Any(key, val).AddTo(enc).
func zapEncoderMethod(goType string) string { //nolint:funlen,gocyclo // straight-line dispatch
	switch goType {
	case "bool":
		return "AddBool"
	case "string":
		return "AddString"
	case "int":
		return "AddInt"
	case "int8":
		return "AddInt8"
	case "int16":
		return "AddInt16"
	case "int32":
		return "AddInt32"
	case "int64":
		return "AddInt64"
	case "uint":
		return "AddUint"
	case "uint8":
		return "AddUint8"
	case "uint16":
		return "AddUint16"
	case "uint32":
		return "AddUint32"
	case "uint64":
		return "AddUint64"
	case "uintptr":
		return "AddUintptr"
	case "float32":
		return "AddFloat32"
	case "float64":
		return "AddFloat64"
	case "complex64":
		return "AddComplex64"
	case "complex128":
		return "AddComplex128"
	case "time.Duration":
		return "AddDuration"
	case "time.Time":
		return "AddTime"
	case "[]byte":
		return "AddBinary"
	default:
		return "Any"
	}
}

// slogConstructorMethod returns the slog package constructor name for a given
// Go field type (e.g. "Int", "String", "Time"). Returns "Any" for types without
// a direct constructor — slog.Any handles them via reflection.
func slogConstructorMethod(goType string) string {
	switch goType {
	case "bool":
		return "Bool"
	case "string":
		return "String"
	case "int":
		return "Int"
	case "int8", "int16", "int32", "int64":
		return "Int64"
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "Uint64"
	case "float32", "float64":
		return "Float64"
	case "time.Time":
		return "Time"
	case "time.Duration":
		return "Duration"
	default:
		return "Any"
	}
}

// slogValueExpr returns the value expression to pass to the slog constructor
// for a field of the given type. Most types pass through as e.<Name>; widening
// types (int8/16/32 -> int64, uint variants -> uint64, float32 -> float64)
// require an explicit cast so the call resolves to the typed constructor.
func slogValueExpr(goType, fieldName string) string {
	base := "e." + fieldName
	switch goType {
	case "int8", "int16", "int32":
		return "int64(" + base + ")"
	case "uint", "uint8", "uint16", "uint32", "uintptr":
		return "uint64(" + base + ")"
	case "float32":
		return "float64(" + base + ")"
	default:
		return base
	}
}

// zerologEventMethod returns the zerolog.Event method name for a given Go field
// type (e.g. "Str", "Int", "Time"). Returns "Interface" for types without a
// direct method — the template uses that to fall back to event.Interface.
func zerologEventMethod(goType string) string { //nolint:funlen,gocyclo // straight-line dispatch
	switch goType {
	case "bool":
		return "Bool"
	case "string":
		return "Str"
	case "int":
		return "Int"
	case "int8":
		return "Int8"
	case "int16":
		return "Int16"
	case "int32":
		return "Int32"
	case "int64":
		return "Int64"
	case "uint":
		return "Uint"
	case "uint8":
		return "Uint8"
	case "uint16":
		return "Uint16"
	case "uint32":
		return "Uint32"
	case "uint64", "uintptr":
		return "Uint64"
	case "float32":
		return "Float32"
	case "float64":
		return "Float64"
	case "time.Time":
		return "Time"
	case "time.Duration":
		return "Dur"
	case "[]byte":
		return "Bytes"
	default:
		return "Interface"
	}
}

// zerologValueExpr returns the value expression to pass to a zerolog.Event method.
// Most types pass through as e.<Name>; uintptr requires widening to uint64 because
// zerolog has no direct Uintptr method.
func zerologValueExpr(goType, fieldName string) string {
	base := "e." + fieldName
	if goType == "uintptr" {
		return "uint64(" + base + ")"
	}
	return base
}
