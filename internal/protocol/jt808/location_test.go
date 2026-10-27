package jt808

import (
	"math"
	"testing"
)

func TestDecodeLatLon(t *testing.T) {
	// status 0x000C0007: bit 2 = 1 (selatan), bit 3 = 0 (timur)
	lat, lon := DecodeLatLon(0x000C0007, 0x00777c09, 0x0694cfcc)

	if math.Abs(lat-(-7.830537)) > 1e-9 {
		t.Errorf("lat = %f, want -7.830537", lat)
	}
	if math.Abs(lon-110.415820) > 1e-9 {
		t.Errorf("lon = %f, want 110.415820", lon)
	}
}

func TestParseLocation(t *testing.T) {
	frame := ParseFrame(mustHex(t, frame0200))
	loc, err := ParseLocation(frame.MsgBody)
	if err != nil {
		t.Fatal(err)
	}

	if loc.Alarm != 0x00000001 {
		t.Errorf("Alarm = 0x%08X, want 0x00000001", loc.Alarm)
	}
	if loc.Status != 0x000C0007 {
		t.Errorf("Status = 0x%08X, want 0x000C0007", loc.Status)
	}
	if math.Abs(loc.Latitude-(-7.830537)) > 1e-9 {
		t.Errorf("Latitude = %f, want -7.830537", loc.Latitude)
	}
	if math.Abs(loc.Longitude-110.415820) > 1e-9 {
		t.Errorf("Longitude = %f, want 110.415820", loc.Longitude)
	}
	if loc.Elevation != 92 {
		t.Errorf("Elevation = %d, want 92", loc.Elevation)
	}
	if loc.Speed != 0 || loc.Direction != 0 {
		t.Errorf("Speed/Direction = %d/%d, want 0/0", loc.Speed, loc.Direction)
	}
	if loc.Time != "2026-10-26 14:34:14" {
		t.Errorf("Time = %s, want 2026-10-26 14:34:14", loc.Time)
	}

	// Frame ini membawa 14 item tambahan
	if len(loc.Extras) != 14 {
		t.Fatalf("jumlah item tambahan = %d, want 14", len(loc.Extras))
	}

	// Item 0x31 = jumlah satelit, nilainya 0x1c (28)
	found := false
	for _, item := range loc.Extras {
		if item.ID == 0x31 {
			found = true
			if len(item.Data) != 1 || item.Data[0] != 0x1c {
				t.Errorf("item 0x31 = %x, want 1c", item.Data)
			}
		}
	}
	if !found {
		t.Error("item 0x31 tidak ditemukan")
	}
}

func TestParseLocationTooShort(t *testing.T) {
	_, err := ParseLocation(make([]byte, 10))
	if err != ErrBodyTooShort {
		t.Errorf("err = %v, want ErrBodyTooShort", err)
	}
}