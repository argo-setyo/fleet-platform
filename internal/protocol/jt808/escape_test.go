package jt808

import (
	"bytes"
	"testing"
)

func TestEscapeAndUnescape(t *testing.T) {
	original := []byte{0x01, 0x7E, 0x02, 0x7D, 0x03}
	want := []byte{0x01, 0x7D, 0x02, 0x02, 0x7D, 0x01, 0x03}

	escaped := Escape(original)
	if !bytes.Equal(escaped, want) {
		t.Errorf("Escape = %x, want %x", escaped, want)
	}

	back := Unescape(escaped)
	if !bytes.Equal(back, original) {
		t.Errorf("Unescape = %x, want %x", back, original)
	}
}