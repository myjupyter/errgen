package test

import "errors"

//go:generate go run github.com/myjupyter/errgen

var ErrInternal = errors.New("internal error")

// @Domain string
// @Tags []string
// @Cause error
// @Is(ErrInternal)
// @Error("internal: service unavailable: [domain=\"%Domain\", tags=%Tags]; cause: %Cause")
var ErrServiceUnavailable = errors.New("service unavailable")
