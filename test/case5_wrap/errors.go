package case5

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
	// @Inner error
	// @Error("wrap single: %Inner")
	ErrWrapSingle = errors.New("wrap single")

	// @First error
	// @Second error
	// @Error("wrap two: %First, %Second")
	ErrWrapTwo = errors.New("wrap two")

	// @Cause error
	// @Underlying error
	// @Root error
	// @Error("wrap chain: %Cause -> %Underlying -> %Root")
	ErrWrapChain = errors.New("wrap chain")

	// @Err error
	ErrWrapNoFormat = errors.New("wrap no format")
)
