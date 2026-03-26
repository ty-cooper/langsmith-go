package internal

import (
	"strings"
	"testing"
	"time"
)

func TestUUID7Format(t *testing.T) {
	id := UUID7()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %s", len(parts), id)
	}
	if len(id) != 36 {
		t.Fatalf("expected length 36, got %d: %s", len(id), id)
	}

	// Check version nibble is 7.
	if id[14] != '7' {
		t.Errorf("expected version nibble '7', got '%c' in %s", id[14], id)
	}

	// Check variant bits (position 19 should be 8, 9, a, or b).
	c := id[19]
	if c != '8' && c != '9' && c != 'a' && c != 'b' {
		t.Errorf("expected variant nibble in [89ab], got '%c' in %s", c, id)
	}
}

func TestUUID7Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := UUID7()
		if seen[id] {
			t.Fatalf("duplicate UUID7 generated: %s", id)
		}
		seen[id] = true
	}
}

func TestUUID7FromTime(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	id := UUID7FromTime(fixedTime)
	if len(id) != 36 {
		t.Fatalf("expected length 36, got %d: %s", len(id), id)
	}
	if id[14] != '7' {
		t.Errorf("expected version nibble '7', got '%c'", id[14])
	}
}

func TestUUID7Ordering(t *testing.T) {
	// UUIDs generated later should sort after earlier ones.
	id1 := UUID7()
	time.Sleep(2 * time.Millisecond)
	id2 := UUID7()
	if id1 >= id2 {
		t.Errorf("expected id1 < id2 (time-sortable), got %s >= %s", id1, id2)
	}
}
