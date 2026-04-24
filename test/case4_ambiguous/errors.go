package case4

import "errors"

// Both netpkg/v1 and netpkg/v2 declare "package netpkg",
// and both auth/v1 and auth/v2 declare "package auth"
// Without -m, the resolver would fail with ambiguity errors
// Each -m flag pins one package name to a specific import path
//go:generate go run github.com/myjupyter/errgen -m netpkg=github.com/myjupyter/errgen/test/case4_ambiguous/netpkg/v1 -m auth=github.com/myjupyter/errgen/test/case4_ambiguous/auth/v2

var (
	// @Peer netpkg.Addr
	// @Error("connection to %Peer failed")
	ErrConnection = errors.New("connection error")

	// @Token auth.Token
	// @Error("authentication failed for token %Token")
	ErrAuth = errors.New("auth error")
)
