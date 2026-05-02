package case7

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
	// @Message string
	// @Error("not found: %Message")
	// @Code(http.StatusNotFound)
	ErrNotFound = errors.New("not found")

	// @Field string
	// @Error("bad request: invalid field '%Field'")
	// @Code(http.StatusBadRequest)
	ErrBadRequest = errors.New("bad request")

	// @Message string
	// @Error("internal: %Message")
	// @Code(500)
	ErrInternal = errors.New("internal error")

	// no @Code — should not get StatusCode method
	// @Reason string
	ErrGeneric = errors.New("generic error")

	// @Message string
	// @Code(codes.NotAuthCode)
	// @Error("not authorized: %Message")
	ErrNotAuth = errors.New("not authorized")
)
