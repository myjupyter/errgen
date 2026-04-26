package test

import "errors"

//go:generate go run github.com/myjupyter/errgen

// @Reason string
// @Code(http.StatusInternalServerError)
// @Error("internal error: reason is %Reason")
var ErrInternal = errors.New("internal error")
