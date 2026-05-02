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
		log.WithFields(e.LogrusFields()).Error("error message")
	}

	log.WithFields(NewEntityNotFoundError("user", 123).LogrusFields()).Error("request failed")
}
