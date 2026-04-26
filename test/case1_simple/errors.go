package case1

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
	// @Code int
	// @Message string
	// @Error("error code %Code: %Message")
	ErrApplication = errors.New("application error")

	// @Reason string
	ErrNotFound = errors.New("not found")

	// @WrappedError error
	// @Error("internal failure: %WrappedError")
	ErrInternal = errors.New("internal error")

	// @Timeout int
	// @Endpoint string
	// @Error("request to '%Endpoint' timed out after %Timeout ms")
	ErrTimeout = errors.New("timeout error")

	// @Error("HTTP error: invalid status code")
	ErrValidation = errors.New("validation error")

	// @Code int
	// @Domain string
	// @Error("invalid argument: code=%Code, domain=%Domain")
	InvalidErr = errors.New("invalid") //nolint:staticcheck // Example test code

	ErrIgnored = errors.New("ignored")
)
