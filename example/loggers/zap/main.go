package main

import (
	"errors"
	"time"

	"go.uber.org/zap"
)

func fooErr() error {
	return NewInternalError(
		"string",
		1,
		2,
		3,
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
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	log.Info("test start")
	defer log.Info("test end")

	err = fooErr()
	var e *InternalError
	if ok := errors.As(err, &e); ok {
		// Generates log like:
		// {"level":"error","ts":1777716527.866704,"caller":"zap/main.go:42","msg":"error message","error":{"stringField":"string","int64Field":1,"intField":2,"uint64Field":3,"float64Field":3.14,"boolField":true,"timeField":1777716527.866702,"durationField":1,"intSliceField":[1,2,3],"objectSliceField":[],"mapType":{"key":"value"}}} //nolint:lll
		log.Error("error message", zap.Object("error fields", e), zap.String("error message", e.Error()))
	}
}
