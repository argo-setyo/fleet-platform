package main

import (
	"encoding/hex"
	"math"
	"testing"
)

// Frame asli dari perangkat: laporan lokasi 0x0200
const frame0200 = "7e0200005e202505250001000e00000001000c000700777c090694cfcc005c00000000261026143414010400000001030200001404000000041504000000001604000000001702000618030000001904000000002504000000002a02000030010031011c520100ad0200f28a7e"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseHeader0200(t *testing.T) {
	p := parseJTT808Frame(mustHex(t, frame0200))

	if p.MessageID != 0x0200 {
		t.Errorf("MessageID = 0x%04X, want 0x0200", p.MessageID)
	}
	if p.PhoneNumber != "202505250001" {
		t.Errorf("Phone = %s, want 202505250001", p.PhoneNumber)
	}
	if p.Serial != 14 {
		t.Errorf("Serial = %d, want 14", p.Serial)
	}
	if len(p.MsgBody) != 94 {
		t.Errorf("len(MsgBody) = %d, want 94", len(p.MsgBody))
	}
	if !p.ChecksumValid {
		t.Error("checksum seharusnya valid")
	}
}

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