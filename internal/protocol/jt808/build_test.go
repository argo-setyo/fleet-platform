package jt808

import (
	"bytes"
	"testing"
)

var testPhone = []byte{0x20, 0x25, 0x05, 0x25, 0x00, 0x01}

// Hasilnya harus sama persis dengan balasan 0x8001 yang tercatat di log
// saat perangkat asli melakukan otentikasi (0x0102, serial 2).
func TestBuildGeneralResponse(t *testing.T) {
	req := Frame{PhoneBCD: testPhone, MessageID: 0x0102, Serial: 2}
	got := BuildGeneralResponse(req, 1, 0x00)
	want := mustHex(t, "7e8001000520250525000100010002010200a07e")

	if !bytes.Equal(got, want) {
		t.Errorf("BuildGeneralResponse = %x, want %x", got, want)
	}
}

// Hasilnya harus sama persis dengan balasan 0x8100 yang tercatat di log
// saat perangkat asli melakukan registrasi (0x0100, serial 1).
func TestBuildRegisterResponse(t *testing.T) {
	req := Frame{PhoneBCD: testPhone, MessageID: 0x0100, Serial: 1}
	got := BuildRegisterResponse(req, 1)
	want := mustHex(t, "7e8100000520250525000100010001004f4ba47e")

	if !bytes.Equal(got, want) {
		t.Errorf("BuildRegisterResponse = %x, want %x", got, want)
	}
}

// Nomor serial pesan balasan harus mengikuti nilai yang diberikan, dan
// terpisah dari serial pesan yang dibalas (yang ada di dalam isi pesan).
func TestBuildResponseUsesOutgoingSerial(t *testing.T) {
	req := Frame{PhoneBCD: testPhone, MessageID: 0x0200, Serial: 7}
	frame := BuildGeneralResponse(req, 258, 0x00)

	p := ParseFrame(frame)
	if !p.ChecksumValid {
		t.Error("checksum seharusnya valid")
	}
	if p.Serial != 258 {
		t.Errorf("serial pesan keluar = %d, want 258", p.Serial)
	}
	// Dua byte pertama isi pesan = serial pesan yang dibalas (7)
	if len(p.MsgBody) < 2 || p.MsgBody[0] != 0x00 || p.MsgBody[1] != 0x07 {
		t.Errorf("MsgBody = %x, want diawali 0007", p.MsgBody)
	}
}

// Byte 7E di dalam isi pesan harus disamarkan, dan kalau frame dibaca lagi
// isinya harus kembali seperti semula.
func TestBuildMessageEscapesBody(t *testing.T) {
	body := []byte{0x7E}
	frame := BuildMessage(0x8001, testPhone, 1, body)

	if n := bytes.Count(frame, []byte{0x7E}); n != 2 {
		t.Errorf("jumlah byte 7E di frame = %d, want 2 (hanya penanda awal dan akhir)", n)
	}

	p := ParseFrame(frame)
	if !p.ChecksumValid {
		t.Error("checksum seharusnya valid")
	}
	if !bytes.Equal(p.MsgBody, body) {
		t.Errorf("MsgBody = %x, want %x", p.MsgBody, body)
	}
}