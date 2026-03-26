package internal

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// UUID7 generates a UUID v7 (time-sortable) as defined in RFC 9562.
func UUID7() string {
	return uuid7FromMillis(time.Now().UnixMilli())
}

// UUID7FromTime generates a UUID v7 using the given timestamp.
func UUID7FromTime(t time.Time) string {
	return uuid7FromMillis(t.UnixMilli())
}

func uuid7FromMillis(ms int64) string {
	var b [16]byte

	// Timestamp (48 bits, big-endian).
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	// Random bytes for the remaining 10 bytes.
	_, _ = rand.Read(b[6:])

	// Set version to 7 (bits 48-51).
	b[6] = (b[6] & 0x0F) | 0x70

	// Set variant to 10xx (bits 64-65).
	b[8] = (b[8] & 0x3F) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16],
	)
}
