package case3

import "errors"

// Generated code goes into the "generated" subdirectory with a different package name
// The -p flag sets the output package, and -o places the file in generated/
// The generated code will import this package to reference the Err sentinels
//go:generate go run github.com/myjupyter/errgen -p generated -o generated/errors_gen.go

var (
	// @StatusCode int
	// @Error("HTTP error: status %StatusCode")
	ErrHTTP = errors.New("http error")

	// @Endpoint string
	// @Timeout int
	// @Error("request to '%Endpoint' timed out after %Timeout ms")
	ErrTimeout = errors.New("timeout error")
)
