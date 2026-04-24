package case6

import "errors"

//go:generate go run github.com/myjupyter/errgen -no-hooks

var (
	// @Code int
	// @Message string
	// @Error("error %Code: %Message")
	ErrApplication = errors.New("application error")

	// @WrappedError error
	// @Error("internal: %WrappedError")
	ErrInternal = errors.New("internal error")
)
