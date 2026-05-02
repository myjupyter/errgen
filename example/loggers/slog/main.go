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
		// Generates log like:
		// 2026/05/02 14:00:46 ERROR error message error.stringField=string error.int64Field=1 error.intField=2 error.uint64Field=3 error.float64Field=3.14 error.boolField=true error.timeField=2026-05-02T14:00:46.524+05:00 error.durationField=1s error.intSliceField="[1 2 3]" error.objectSliceField=[] error.mapType=map[key:value] //nolint: lll
		slog.Error("error message", "error", e)
	}
	err = NewEntityNotFoundError("user", 123)
	slog.Error("request failed", "error", err)
}
