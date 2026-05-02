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
		log.Error().Object("error", e).Msg("error message")
	}

	log.Error().Object("error", NewEntityNotFoundError("user", 123)).Msg("error message")
}
