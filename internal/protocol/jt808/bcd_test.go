package jt808

import "testing"

func TestParseBCDPhone(t *testing.T) {
	got := ParseBCDPhone([]byte{0x20, 0x25, 0x05, 0x25, 0x00, 0x01})
	want := "202505250001"
	if got != want {
		t.Errorf("ParseBCDPhone = %s, want %s", got, want)
	}
}

func TestParseBCDTime(t *testing.T) {
	// Waktu dari frame nyata: 26 10 26 16 10 10
	got := ParseBCDTime([]byte{0x26, 0x10, 0x26, 0x16, 0x10, 0x10})
	want := "2026-10-26 16:10:10"
	if got != want {
		t.Errorf("ParseBCDTime = %s, want %s", got, want)
	}

	// Panjang data salah harus ditolak, bukan membuat program error
	if got := ParseBCDTime([]byte{0x26, 0x10}); got != "(invalid BCD time)" {
		t.Errorf("ParseBCDTime(data pendek) = %s, want (invalid BCD time)", got)
	}
}