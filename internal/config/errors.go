package config

import "errors"

//go:generate go run github.com/myjupyter/errgen -no-hooks

var (
	// @Error("usage: errgen [flags] [source-file]")
	ErrNoInputFileSpecified = errors.New("no input file specified")

	ErrNoPackageSpecified = errors.New("no package name specified")
)
