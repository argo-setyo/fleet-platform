package jt808

import (
	"encoding/hex"
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

func TestParseFrameHeader(t *testing.T) {
	p := ParseFrame(mustHex(t, frame0200))

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

func TestParseFrameBadChecksum(t *testing.T) {
	b := mustHex(t, frame0200)
	b[16] = 0x00 // ubah satu byte di isi pesan, checksum dibiarkan

	p := ParseFrame(b)
	if p.ChecksumValid {
		t.Error("checksum seharusnya TIDAK valid setelah isi pesan diubah")
	}
}

func TestParseFrameTooShort(t *testing.T) {
	// Data terlalu pendek tidak boleh membuat program error
	for _, short := range [][]byte{
		{0x7E, 0x7E},
		{0x7E, 0x01, 0x02, 0x7E},
	} {
		p := ParseFrame(short)
		if p.MessageID != 0 || len(p.MsgBody) != 0 || p.ChecksumValid {
			t.Errorf("ParseFrame(%x) seharusnya menghasilkan Frame kosong", short)
		}
	}
}

func TestChecksum(t *testing.T) {
	// 0x01 XOR 0x02 XOR 0x04 = 0x07
	if got := Checksum([]byte{0x01, 0x02, 0x04}); got != 0x07 {
		t.Errorf("Checksum = 0x%02X, want 0x07", got)
	}
}