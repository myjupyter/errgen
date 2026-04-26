package case8

import "errors"

//go:generate go run github.com/myjupyter/errgen -stack-trace

var (
	// @Code int
	// @Message string
	// @Error("error %Code: %Message")
	ErrApplication = errors.New("application error")

	// @WrappedError error
	// @Error("internal: %WrappedError")
	ErrInternal = errors.New("internal error")

	// no fields
	// @Error("forbidden")
	ErrForbidden = errors.New("forbidden")
)
