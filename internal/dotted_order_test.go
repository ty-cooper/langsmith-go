package internal

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateDottedOrder(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	runID := "abc-123"
	result := GenerateDottedOrder(ts, runID)

	expected := "20240615T103000000000Zabc-123"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestAppendDottedOrder(t *testing.T) {
	parentTS := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	parentOrder := GenerateDottedOrder(parentTS, "parent-id")

	childTS := time.Date(2024, 6, 15, 10, 30, 1, 0, time.UTC)
	result := AppendDottedOrder(parentOrder, childTS, "child-id")

	parts := strings.Split(result, ".")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d: %s", len(parts), result)
	}
	if !strings.Contains(parts[0], "parent-id") {
		t.Error("first part should contain parent-id")
	}
	if !strings.Contains(parts[1], "child-id") {
		t.Error("second part should contain child-id")
	}
}

func TestParseTraceIDFromDottedOrder(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	order := GenerateDottedOrder(ts, "my-trace-id")

	traceID := ParseTraceIDFromDottedOrder(order)
	if traceID != "my-trace-id" {
		t.Errorf("expected my-trace-id, got %s", traceID)
	}

	// Test with child dotted order.
	childOrder := AppendDottedOrder(order, ts, "child-id")
	traceID = ParseTraceIDFromDottedOrder(childOrder)
	if traceID != "my-trace-id" {
		t.Errorf("expected my-trace-id from child order, got %s", traceID)
	}

	// Test edge cases.
	if ParseTraceIDFromDottedOrder("") != "" {
		t.Error("expected empty for empty input")
	}
	if ParseTraceIDFromDottedOrder("no-z-char") != "" {
		t.Error("expected empty for input without Z")
	}
}
