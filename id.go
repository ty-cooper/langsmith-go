package langsmith

import "github.com/tcoooper/langsmith-go/internal"

// NewID generates a new time-sortable UUID v7.
func NewID() string {
	return internal.UUID7()
}
