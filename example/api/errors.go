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
