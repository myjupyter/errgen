package main

import (
	"errors"
	"os"
	"time"

	"github.com/rs/zerolog"
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
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

	log.Info().Msg("test start")
	defer log.Info().Msg("test end")

	err := fooErr()
	var e *InternalError
	if ok := errors.As(err, &e); ok {
		// Generates log like:
		// {"level":"error","error":{"error":"internal error","stringField":"string","int64Field":1,"intField":2,"uint64Field":3,"uintField":4,"uintptrField":5,"float64Field":3.14,"boolField":true,"timeField":"2026-05-02T17:03:12+05:00","durationField":1000,"intSliceField":[1,2,3],"objectSliceField":[],"mapType":{"key":"value"}},"time":"2026-05-02T17:03:12+05:00","message":"error message"} //nolint:lll
		log.Error().Object("error", e).Msg("error message")
	}

	log.Error().Object("error", NewEntityNotFoundError("user", 123)).Msg("error message")
}
