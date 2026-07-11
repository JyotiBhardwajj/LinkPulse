// Package errors defines structured domain errors and HTTP mapping helpers.
package errors

import (
	"errors"
	"net/http"
)

var (
	// ErrNotFound is returned when a requested resource is missing.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when a resource violates unique constraints.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput is returned when parameter validation fails.
	ErrInvalidInput = errors.New("invalid input data")

	// ErrUnauthorized is returned when credentials are invalid or missing.
	ErrUnauthorized = errors.New("unauthorized access")

	// ErrInvalidCredentials is returned on login failures to prevent user enumeration.
	ErrInvalidCredentials = errors.New("Invalid email or password.")

	// ErrInternal is returned when an unexpected system error occurs.
	ErrInternal = errors.New("internal server error")

	// ErrGone is returned when a resource is expired or permanently unavailable.
	ErrGone = errors.New("resource is gone")
)

// MapToHTTPStatus resolves domain errors to standard HTTP status codes.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrGone):
		return http.StatusGone
	default:
		return http.StatusInternalServerError
	}
}
