package main

import (
	"errors"
	"fmt"
)

func inner(id int) error {
	return NewUserNotFoundError(id)
}

func middle(id int) error {
	return inner(id)
}

func outer(id int) error {
	if err := middle(id); err != nil {
		return NewDownstreamError(err)
	}
	return nil
}

func main() {
	err := outer(42)
	if err == nil {
		return
	}

	fmt.Println("--- default ---")
	fmt.Printf("%v\n", err)

	fmt.Println("\n--- plus-verb ---")
	fmt.Printf("%+v\n", err)

	fmt.Println("\n--- Customizing output via StackFrames() ---")
	var notFound *UserNotFoundError
	if errors.As(err, &notFound) {
		for i, f := range notFound.StackFrames() {
			fmt.Printf("  [%d] %s\n      %s:%d\n", i, f.Function, f.File, f.Line)
		}
	}
}
