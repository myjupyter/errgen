package main

import (
	"errors"
	"log/slog"
	"time"
)

func fooErr() error {
	return NewInternalError(
		"string",
		1,
		2,
		3,
		4,
		5,
		3.14,
		true,
		time.Now(),
		time.Second,
		[]int{1, 2, 3},
		[]Object{},
		map[string]string{
			"key": "value",
		},
	)
}

func main() {
	slog.Info("test start")
	defer slog.Info("test end")

	err := fooErr()
	var e *InternalError
	if ok := errors.As(err, &e); ok {
		slog.Error("error message", "error", e)
	}
	err = NewEntityNotFoundError("user", 123)
	slog.Error("request failed", "error", err)
}
