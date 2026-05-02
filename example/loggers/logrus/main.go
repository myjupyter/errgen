package main

import (
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
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
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetFormatter(&logrus.JSONFormatter{})

	log.Info("test start")
	defer log.Info("test end")

	err := fooErr()
	var e *InternalError
	if ok := errors.As(err, &e); ok {
		// Generates log like:
		// {"boolField":true,"code":404,"durationField":1000000000,"error":"internal error","float64Field":3.14,"intField":2,"intSliceField":[1,2,3],"int64Field":1,"level":"error","mapType":{"key":"value"},"msg":"error message","objectSliceField":[],"stringField":"string","timeField":"2026-05-02T17:03:12+05:00","uint64Field":3,"uintField":4,"uintptrField":5} //nolint:lll
		log.WithFields(e.LogrusFields()).Error("error message")
	}

	log.WithFields(NewEntityNotFoundError("user", 123).LogrusFields()).Error("request error")
}
