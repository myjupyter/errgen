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
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	log.Info("test start")
	defer log.Info("test end")

	err = fooErr()
	var e *InternalError
	if ok := errors.As(err, &e); ok {
		log.Error("error message", zap.Object("error", e))
	}

	log.Error("error message", zap.Object("error", NewEntityNotFoundError("user", 123)))
}
