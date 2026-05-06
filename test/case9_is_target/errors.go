package case9

import "errors"

//go:generate go run github.com/myjupyter/errgen -no-hooks

var (
	// @Name string
	// @Is(errcodes.ErrInvalidArgument)
	// @Error("username '%Name' is invalid")
	ErrInvalidUserName = errors.New("invalid username")
	// @Is(errcodes.ErrInvalidArgument)
	ErrInvalidSettings = errors.New("invalid settings")

	// @ID int
	// @Is(errcodes.ErrNotFound)
	// @Error("user with ID '%ID' not found")
	ErrNotFoundUser = errors.New("user not found")
	// @Is(errcodes.ErrNotFound)
	ErrNotFoundSettings = errors.New("settings not found")

	// @Cause error
	// @Is(errcodes.ErrInternal)
	// @Error("user service internal error: %Cause")
	ErrUserServiceInternal = errors.New("user service internal error")
)
