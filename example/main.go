package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/myjupyter/errgen/example/api"
)

func fetchUser(id int) error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/users/%d", id))
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // example code

	if resp.StatusCode != http.StatusOK {
		return api.NewHTTPError(resp.StatusCode, "user not found")
	}
	return nil
}

func validateAge(age int) error {
	if age < 0 {
		return api.NewValidationError("age", fmt.Errorf("must be non-negative, got %d", age))
	}
	return nil
}

func main() {
	err := fetchUser(42)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			fmt.Printf("status: %d, message: %s\n", httpErr.StatusCode, httpErr.Message)
			// status: 404, message: user not found
		}

		if errors.Is(err, api.ErrHTTP) {
			fmt.Println("it's an HTTP error")
		}
	}

	err = validateAge(-1)
	if err != nil {
		var valErr *api.ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("field: %s, cause: %s\n", valErr.Field, valErr.WrappedError)
			// field: age, cause: must be non-negative, got -1
		}

		// errors.Is works through the Unwrap chain
		if errors.Is(err, api.ErrValidation) {
			fmt.Println("it's a validation error")
		}
	}
}
