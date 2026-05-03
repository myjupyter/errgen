package main

import "errors"

//go:generate go run github.com/myjupyter/errgen -stack-trace -no-hooks

// @UserID int
// @Error("user %UserID not found")
var ErrUserNotFound = errors.New("user not found")

// @WrappedError error
// @Error("downstream call failed: %WrappedError")
var ErrDownstream = errors.New("downstream error")
