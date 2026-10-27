package main

import (
	"math"
	"testing"
)

func TestDecodeLatLon(t *testing.T) {
	// status 0x000C0007: bit 2 = 1 (selatan), bit 3 = 0 (timur)
	lat, lon := decodeLatLon(0x000C0007, 0x00777c09, 0x0694cfcc)

	if math.Abs(lat-(-7.830537)) > 1e-9 {
		t.Errorf("lat = %f, want -7.830537", lat)
	}
	if math.Abs(lon-110.415820) > 1e-9 {
		t.Errorf("lon = %f, want 110.415820", lon)
	}
}