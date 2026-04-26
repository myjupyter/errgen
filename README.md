# errgen

Go code generator for rich error types. Stdlib only, zero dependencies.

- Annotate `errors.New` sentinels with fields and a format string
- Generates struct, constructor, `Error()`, `Is()`, `Unwrap()`, `LogValue()`, `MarshalJSON()`, `UnmarshalJSON()`
- Errors as data containers - carry context, extract with `errors.As`
- Customizable via Go templates and `onCreate` hooks
- Doesn't create extra dependencies in your code

## Table of Contents

- [Install](#install)
- [Usage](#usage)
  - [Flags](#flags)
  - [Cross-package generation](#cross-package-generation)
  - [Ambiguous imports (`-m`)](#ambiguous-imports--m)
- [Annotation DSL](#annotation-dsl)
  - [`@Name Type` - field declaration](#name-type----field-declaration)
  - [`@Error("format string")` - error message](#errorformat-string----error-message)
  - [Error-typed fields and `Unwrap`](#error-typed-fields-and-unwrap)
  - [Structured logging (`slog.LogValuer`)](#structured-logging-sloglogvaluer)
  - [JSON serialization](#json-serialization)
  - [Generated type naming](#generated-type-naming)
  - [Hook file (`_gen_hook.go`)](#hook-file-_gen_hookgo)
- [Example](#example)

## Install

```sh
go install github.com/myjupyter/errgen@latest
```

## Usage

Add a `go:generate` directive to your source file:

```go
//go:generate go run github.com/myjupyter/errgen
```

Then run:

```sh
go generate ./...
```

### Flags

```
-p string    package name for the generated file (default: $GOPACKAGE)
-o string    output file path (default: <input>_gen.go)
-t string    path to a custom Go template file (default: built-in template)
-n bool      dry run: print generated code to stdout instead of writing a file
-m value     manual import mapping: pkg=import/path (repeatable, for ambiguous packages)
-no-hooks    skip hook file generation
```

### Cross-package generation

Use `-p` and `-o` to generate into a different package. The generated code automatically imports the source package to reference the sentinel variables:

```go
//go:generate go run github.com/myjupyter/errgen -p generated -o generated/errors_gen.go
```

### Ambiguous imports (`-m`)

When two packages in your module share the same Go package name, the resolver can't pick one automatically. Use `-m` to specify which import path to use:

```go
//go:generate go run github.com/myjupyter/errgen -m netpkg=github.com/myapp/netpkg/v2
```

Multiple `-m` flags can be combined:

```go
//go:generate go run github.com/myjupyter/errgen -m netpkg=github.com/myapp/netpkg/v1 -m auth=github.com/myapp/auth/v2
```

## Annotation DSL

Annotations are placed in comments directly above a `var` declaration assigned via `errors.New()` or `fmt.Errorf()`.

### `@Name Type` - field declaration

Declares a field on the generated error struct. `Name` must be a valid Go identifier (not a Go keyword or error method). `Type` can be any valid Go type.

```go
// @Code int
// @Message string
ErrExample = errors.New("example")
```

This generates an `ExampleError` struct with `Code int` and `Message string` fields, a `NewExampleError(code int, message string)` constructor, and the full error interface.

Supported type forms:

| Type              | Example                          |
|-------------------|----------------------------------|
| Builtin           | `int`, `string`, `bool`          |
| Pointer           | `*pkg.Type`                      |
| Slice             | `[]pkg.Type`                     |
| Map               | `map[pkg.Key]pkg.Value`            |
| Qualified         | `pkg.Type`                       |
| Error             | `error` (triggers special Unwrap) |

Anonymous structs (`struct{ ... }`) and generic types (`Type[T]`) are not supported - define a named type instead.

For qualified types (e.g. `pkg.Type`), `errgen` automatically scans your module to find the matching import path. If multiple packages share the same name, use the `-m` flag to disambiguate (see [Ambiguous imports](#ambiguous-imports--m) above).

### `@Error("format string")` - error message

Defines the return value of `Error()`. Reference declared fields with `%FieldName` - each expands to `%v` in a `fmt.Sprintf` call.

```go
// @Code int
// @Domain string
// @Error("invalid argument: code=%Code, domain=%Domain")
ErrInvalid = errors.New("invalid")
```

Generated:

```go
func (e *InvalidError) Error() string {
    return fmt.Sprintf("invalid argument: code=%v, domain=%v", e.Code, e.Domain)
}
```

If `@Error` is omitted, `Error()` returns the sentinel's message. All `%FieldName` references must match declared fields - unknown references produce a parse error.

### Error-typed fields and `Unwrap`

When fields have type `error`, they are all included in `Unwrap() []error` alongside the sentinel. This supports the full `errors.Is`/`errors.As` chain:

```go
// @Cause error
// @Underlying error
// @Error("wrap chain: %Cause -> %Underlying")
ErrWrapChain = errors.New("wrap chain")
```

Generated:

```go
func (e *WrapChainError) Unwrap() []error {
    return []error{e.Cause, e.Underlying, ErrWrapChain}
}
```

Without error-typed fields, `Unwrap()` returns just the sentinel as a single `error`.

### Structured logging (`slog.LogValuer`)

Every error type with fields implements `slog.LogValuer` (stdlib since Go 1.21). This means structured loggers automatically extract typed fields instead of just calling `Error()`:

```go
slog.Error("request failed", "err", api.NewHTTPError(404, "user not found"))
```

Output:

```json
{"level":"ERROR","msg":"request failed","err":{"statusCode":404,"message":"user not found"}}
```

Works with any `slog`-compatible logger (zerolog, zap via bridge, etc).

### JSON serialization

Every error type with fields implements `json.Marshaler` and `json.Unmarshaler`. The JSON output includes an `"error"` key with the formatted message plus all typed fields:

```go
err := api.NewHTTPError(404, "user not found")
data, _ := json.Marshal(err)
```

```json
{"error":"HTTP 404: user not found","statusCode":404,"message":"user not found"}
```

Error-typed fields are serialized as strings. On unmarshal, they are reconstructed via `errors.New`.

### Generated type naming

The type name is derived from the variable name by stripping the `Err` prefix and appending `Error`:

| Variable          | Generated type      |
|-------------------|---------------------|
| `ErrHTTP`         | `HTTPError`         |
| `ErrNotFound`     | `NotFoundError`     |
| `ErrValidation`   | `ValidationError`   |

### Hook file (`_gen_hook.go`)

On the first run, `errgen` creates a hook file alongside the generated code. This file contains an `onCreate()` method stub for each error type:

```go
func (e *HTTPError) onCreate() {
    // called by the constructor - add your logic here
}
```

The constructor calls `e.onCreate()` after populating all fields. For example, you can log, record metrics, or capture a stack trace:

```go
func (e *HTTPError) onCreate() {
    // capture stack trace
    buf := make([]byte, 4096)
    n := runtime.Stack(buf, false)

    // log with context
    logger.Error("HTTP error created",
        zap.Int("status_code", e.StatusCode),
        zap.String("stack", string(buf[:n])),
    )

    // record metric
    httpErrors.WithLabelValues(fmt.Sprintf("%d", e.StatusCode)).Inc()
}
```

Your custom logic is safe across regenerations:

- Existing `onCreate()` methods are never overwritten or removed
- When you add a new error type, its stub is automatically appended to the hook file
- When you remove an error type, its hook stays in the file (delete it manually if needed)

## Example

Annotate your sentinel errors ([example/api/errors.go](example/api/errors.go)):

```go
package api

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
    // @StatusCode int
    // @Message string
    // @Error("HTTP %StatusCode: %Message")
    ErrHTTP = errors.New("http error")

    // @Field string
    // @WrappedError error
    // @Error("validation failed on '%Field': %WrappedError")
    ErrValidation = errors.New("validation error")
)
```

Run `go generate ./...`. This produces two files:

- **[`errors_gen.go`](example/api/errors_gen.go)** - the generated structs, constructors, and error interface methods. Regenerated every time.
- **[`errors_gen_hook.go`](example/api/errors_gen_hook.go)** - `onCreate()` stubs for custom logic. Created once, never overwritten.

Then use the generated types in your code ([example/main.go](example/main.go)):

```go
// Create errors with typed fields
err := api.NewHTTPError(http.StatusNotFound, "user not found")

...

// Extract data with errors.As
var httpErr *api.HTTPError
if errors.As(err, &httpErr) {
    fmt.Printf("status: %d, message: %s\n", httpErr.StatusCode, httpErr.Message)
}

// Match sentinels with errors.Is
if errors.Is(err, api.ErrHTTP) {
    fmt.Println("it's an HTTP error")
}
```

See the full runnable example in the [example/](example/) directory.

## License

[MIT](LICENSE)
