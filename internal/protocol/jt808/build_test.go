package jt808

import (
	"bytes"
	"testing"
)

var testPhone = []byte{0x20, 0x25, 0x05, 0x25, 0x00, 0x01}

// Hasilnya harus sama persis dengan balasan 0x8001 yang tercatat di log
// saat perangkat asli melakukan otentikasi (0x0102, serial 2).
func TestBuildGeneralResponse(t *testing.T) {
	got := BuildGeneralResponse(testPhone, 0x0102, 2, 0x00)
	want := mustHex(t, "7e8001000520250525000100010002010200a07e")

	if !bytes.Equal(got, want) {
		t.Errorf("BuildGeneralResponse = %x, want %x", got, want)
	}
}

// Hasilnya harus sama persis dengan balasan 0x8100 yang tercatat di log
// saat perangkat asli melakukan registrasi (0x0100, serial 1).
func TestBuildRegisterResponse(t *testing.T) {
	got := BuildRegisterResponse(testPhone, 1)
	want := mustHex(t, "7e8100000520250525000100010001004f4ba47e")

	if !bytes.Equal(got, want) {
		t.Errorf("BuildRegisterResponse = %x, want %x", got, want)
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