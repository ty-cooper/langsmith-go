package langsmith

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a resource is not found by name or filter
// (as opposed to a 404 from the API, which produces an APIError).
var ErrNotFound = errors.New("langsmith: not found")

// APIError represents an error response from the LangSmith API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("langsmith API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("langsmith API error (status %d): %s", e.StatusCode, e.Body)
}

// IsRetryable returns true if the error is potentially retryable (5xx or 429).
func (e *APIError) IsRetryable() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

// LangSmithError is a general SDK error.
type LangSmithError struct {
	Message string
	Err     error
}

func (e *LangSmithError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("langsmith: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("langsmith: %s", e.Message)
}

func (e *LangSmithError) Unwrap() error {
	return e.Err
}

// AsAPIError extracts an *APIError from err using errors.As.
// Returns nil if err does not contain an APIError.
func AsAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

// IsNotFound returns true if err represents a not-found condition,
// either via ErrNotFound or an APIError with status 404.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	if apiErr := AsAPIError(err); apiErr != nil {
		return apiErr.StatusCode == 404
	}
	return false
}
