package internal

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// UUID7 generates a UUID v7 (time-sortable) as defined in RFC 9562.
// The first 48 bits are the Unix timestamp in milliseconds,
// followed by random bits with the version and variant set appropriately.
func UUID7() string {
	now := time.Now().UnixMilli()

	var b [16]byte

	// Timestamp (48 bits, big-endian)
	b[0] = byte(now >> 40)
	b[1] = byte(now >> 32)
	b[2] = byte(now >> 24)
	b[3] = byte(now >> 16)
	b[4] = byte(now >> 8)
	b[5] = byte(now)

	// Random bytes for the rest
	randBytes := make([]byte, 10)
	_, _ = rand.Read(randBytes)
	copy(b[6:], randBytes)

	// Set version to 7 (bits 48-51)
	b[6] = (b[6] & 0x0F) | 0x70

	// Set variant to 10xx (bits 64-65)
	b[8] = (b[8] & 0x3F) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16],
	)
}

// UUID7FromTime generates a UUID v7 using the given timestamp.
func UUID7FromTime(t time.Time) string {
	ms := t.UnixMilli()

	var b [16]byte

	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	randBytes := make([]byte, 10)
	_, _ = rand.Read(randBytes)
	copy(b[6:], randBytes)

	b[6] = (b[6] & 0x0F) | 0x70
	b[8] = (b[8] & 0x3F) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16],
	)
}
