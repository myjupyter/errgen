package errcodes

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("not found")
	ErrInternal        = errors.New("internal error")
)
