package jt808

import (
	"testing"
	"time"
)

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

// Waktu perangkat 2026-10-27 10:51:19 di UTC+7 adalah 03:51:19 UTC.
func TestBCDTimeToUTC(t *testing.T) {
	bcd := []byte{0x26, 0x10, 0x27, 0x10, 0x51, 0x19}

	got, err := BCDTimeToUTC(bcd, 420)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 27, 3, 51, 19, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("UTC+7: got %v, want %v", got, want)
	}

	// Perangkat yang diatur ke UTC: waktunya sama dengan UTC
	got, err = BCDTimeToUTC(bcd, 0)
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, 10, 27, 10, 51, 19, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("UTC+0: got %v, want %v", got, want)
	}
}

// Pukul 02:30 tanggal 27 di UTC+7 masih tanggal 26 di UTC.
func TestBCDTimeToUTCCrossesMidnight(t *testing.T) {
	bcd := []byte{0x26, 0x10, 0x27, 0x02, 0x30, 0x00}

	got, err := BCDTimeToUTC(bcd, 420)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 26, 19, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBCDTimeToUTCInvalid(t *testing.T) {
	cases := map[string][]byte{
		"panjang salah":       {0x26, 0x10},
		"bulan 13":            {0x26, 0x13, 0x01, 0x00, 0x00, 0x00},
		"31 Februari":         {0x26, 0x02, 0x31, 0x00, 0x00, 0x00},
		"jam 24":              {0x26, 0x10, 0x27, 0x24, 0x00, 0x00},
		"hari 0":              {0x26, 0x10, 0x00, 0x10, 0x00, 0x00},
		"bukan digit BCD (A)": {0x2A, 0x10, 0x27, 0x10, 0x00, 0x00},
	}

	for name, bcd := range cases {
		if _, err := BCDTimeToUTC(bcd, 420); err != ErrInvalidTime {
			t.Errorf("%s: err = %v, want ErrInvalidTime", name, err)
		}
	}
}