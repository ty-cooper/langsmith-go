package langsmith

import "fmt"

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
