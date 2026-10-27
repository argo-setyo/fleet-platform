package jt808

import (
	"math"
	"testing"
)

// Frame 0x0704 asli dari perangkat (serial 7): satu catatan lokasi susulan
// bertanggal 2026-10-26 14:33:54.
const frame0704 = "7e070400632025052500010007000101005e00000001000c000700777c090694cfcc005c00000000261026143354010400000001030200001404000000041504000000001604000000001702000618030000001904000000002504000000002a02000030010031011c520100ad0200f3a77e"

func TestParseLocationBatchRealFrame(t *testing.T) {
	frame := ParseFrame(mustHex(t, frame0704))
	if frame.MessageID != 0x0704 {
		t.Fatalf("MessageID = 0x%04X, want 0x0704", frame.MessageID)
	}
	if !frame.ChecksumValid {
		t.Fatal("checksum seharusnya valid")
	}

	batch, err := ParseLocationBatch(frame.MsgBody)
	if err != nil {
		t.Fatal(err)
	}

	if batch.Type != BatchBlindFill {
		t.Errorf("Type = %d, want %d", batch.Type, BatchBlindFill)
	}
	if len(batch.Locations) != 1 {
		t.Fatalf("jumlah lokasi = %d, want 1", len(batch.Locations))
	}

	loc := batch.Locations[0]
	if loc.Time != "2026-10-26 14:33:54" {
		t.Errorf("Time = %s, want 2026-10-26 14:33:54", loc.Time)
	}
	if math.Abs(loc.Latitude-(-7.830537)) > 1e-9 {
		t.Errorf("Latitude = %f, want -7.830537", loc.Latitude)
	}
	if math.Abs(loc.Longitude-110.415820) > 1e-9 {
		t.Errorf("Longitude = %f, want 110.415820", loc.Longitude)
	}
}

// Menyusun isi 0x0704 berisi n laporan lokasi yang sama (94 byte per laporan).
func buildBatchBody(t *testing.T, batchType byte, n int, declaredCount int) []byte {
	t.Helper()
	locBody := ParseFrame(mustHex(t, frame0200)).MsgBody // 94 byte

	body := []byte{byte(declaredCount >> 8), byte(declaredCount), batchType}
	for i := 0; i < n; i++ {
		body = append(body, byte(len(locBody)>>8), byte(len(locBody)))
		body = append(body, locBody...)
	}
	return body
}

func TestParseLocationBatchTwoItems(t *testing.T) {
	batch, err := ParseLocationBatch(buildBatchBody(t, BatchNormal, 2, 2))
	if err != nil {
		t.Fatal(err)
	}
	if batch.Type != BatchNormal {
		t.Errorf("Type = %d, want %d", batch.Type, BatchNormal)
	}
	if len(batch.Locations) != 2 {
		t.Errorf("jumlah lokasi = %d, want 2", len(batch.Locations))
	}
}

// Jumlah yang diumumkan 2, tetapi data hanya berisi 1: yang utuh tetap
// dikembalikan, disertai error.
func TestParseLocationBatchTruncated(t *testing.T) {
	batch, err := ParseLocationBatch(buildBatchBody(t, BatchNormal, 1, 2))
	if err != ErrBodyTooShort {
		t.Errorf("err = %v, want ErrBodyTooShort", err)
	}
	if len(batch.Locations) != 1 {
		t.Errorf("jumlah lokasi = %d, want 1 (yang utuh tetap dikembalikan)", len(batch.Locations))
	}
}

func TestParseLocationBatchTooShort(t *testing.T) {
	_, err := ParseLocationBatch([]byte{0x00, 0x01})
	if err != ErrBodyTooShort {
		t.Errorf("err = %v, want ErrBodyTooShort", err)
	}
}