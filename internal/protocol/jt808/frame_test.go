package jt808

import (
	"bytes"
	"testing"
)

func TestExtractFrames(t *testing.T) {
	f1 := []byte{0x7E, 0x01, 0x02, 0x7E}
	f2 := []byte{0x7E, 0x03, 0x04, 0x7E}

	// Satu bacaan TCP berisi: frame 1 + frame 2 + awal frame 3 (belum lengkap)
	buf := append(append([]byte{}, f1...), f2...)
	buf = append(buf, 0x7E, 0x05)

	frames := ExtractFrames(&buf)

	if len(frames) != 2 {
		t.Fatalf("jumlah frame = %d, want 2", len(frames))
	}
	if !bytes.Equal(frames[0], f1) {
		t.Errorf("frame[0] = %x, want %x", frames[0], f1)
	}
	if !bytes.Equal(frames[1], f2) {
		t.Errorf("frame[1] = %x, want %x", frames[1], f2)
	}

	// Bagian frame 3 yang belum lengkap harus tetap tersimpan di buffer
	if !bytes.Equal(buf, []byte{0x7E, 0x05}) {
		t.Errorf("sisa buffer = %x, want 7e05", buf)
	}

	// Sisa frame 3 datang di bacaan berikutnya
	buf = append(buf, 0x06, 0x7E)
	frames = ExtractFrames(&buf)

	want3 := []byte{0x7E, 0x05, 0x06, 0x7E}
	if len(frames) != 1 || !bytes.Equal(frames[0], want3) {
		t.Errorf("frames = %x, want satu frame %x", frames, want3)
	}
	if len(buf) != 0 {
		t.Errorf("buffer seharusnya kosong, isinya %x", buf)
	}
}