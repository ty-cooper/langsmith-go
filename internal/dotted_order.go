package internal

import (
	"fmt"
	"strings"
	"time"
)

// GenerateDottedOrder creates a dotted order string for a run.
// Format: YYYYMMDDTHHMMSSffffffZ<uuid>
// The microsecond portion uses no decimal separator to avoid
// conflicting with the dotted order's "." child separator.
// For child runs, it's parent_dotted_order.child_dotted_order
func GenerateDottedOrder(t time.Time, runID string) string {
	utc := t.UTC()
	us := utc.Nanosecond() / 1000
	return fmt.Sprintf("%s%06dZ%s", utc.Format("20060102T150405"), us, runID)
}

// AppendDottedOrder appends a child segment to a parent dotted order.
func AppendDottedOrder(parentDottedOrder string, t time.Time, runID string) string {
	child := GenerateDottedOrder(t, runID)
	return parentDottedOrder + "." + child
}

// ParseTraceIDFromDottedOrder extracts the trace (root run) ID from a dotted order string.
// The trace ID is the UUID in the first segment.
func ParseTraceIDFromDottedOrder(dottedOrder string) string {
	// Split on "." to get segments; first segment is root.
	parts := strings.SplitN(dottedOrder, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	// Format is: YYYYMMDDTHHMMSSffffffZ<uuid>
	idx := strings.Index(parts[0], "Z")
	if idx == -1 || idx+1 >= len(parts[0]) {
		return ""
	}
	return parts[0][idx+1:]
}
