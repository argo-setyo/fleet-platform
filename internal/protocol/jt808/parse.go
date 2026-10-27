package jt808

import "encoding/binary"

// Frame adalah hasil membaca satu frame JT808:
// header (ID pesan, properti, nomor terminal, serial), isi pesan, dan checksum.
type Frame struct {
	MessageID     uint16
	MsgBodyProps  uint16
	PhoneBCD      []byte
	PhoneNumber   string
	Serial        uint16
	MsgBody       []byte
	Checksum      byte
	ChecksumValid bool
	Raw           []byte
}

// Checksum menghitung checksum JT808, yaitu XOR dari semua byte.
func Checksum(data []byte) byte {
	cs := byte(0)
	for _, b := range data {
		cs ^= b
	}
	return cs
}

// ParseFrame membaca satu frame utuh (lengkap dengan penanda 7E di awal dan
// akhir). Fungsi ini hanya membaca dan tidak mencetak apa pun. Kalau frame
// terlalu pendek, hasilnya Frame kosong dengan ChecksumValid bernilai false.
//
// Header yang dibaca adalah JT808-2013 (12 byte, nomor terminal BCD 6 byte).
func ParseFrame(frame []byte) Frame {
	result := Frame{Raw: frame}
	if len(frame) < 3 {
		return result
	}

	// Buang penanda 7E di kedua ujung, lalu kembalikan byte yang disamarkan
	rawPayload := frame[1 : len(frame)-1]
	data := Unescape(rawPayload)

	// 12 byte header + 1 byte checksum
	if len(data) < 13 {
		return result
	}

	result.MessageID = binary.BigEndian.Uint16(data[0:2])
	result.MsgBodyProps = binary.BigEndian.Uint16(data[2:4])
	result.PhoneBCD = data[4:10]
	result.PhoneNumber = ParseBCDPhone(data[4:10])
	result.Serial = binary.BigEndian.Uint16(data[10:12])

	bodyLen := int(result.MsgBodyProps & 0x03FF)
	if bodyLen > len(data)-13 {
		bodyLen = len(data) - 13
	}
	result.MsgBody = data[12 : 12+bodyLen]
	result.Checksum = data[len(data)-1]
	result.ChecksumValid = Checksum(data[:len(data)-1]) == result.Checksum
	return result
}