package jt808

// BuildMessage menyusun frame JT808 lengkap yang siap dikirim ke perangkat:
// header, isi pesan, checksum, escape, lalu penanda 7E di awal dan akhir.
func BuildMessage(msgID uint16, phone []byte, serial uint16, body []byte) []byte {
	msgProps := uint16(len(body)) & 0x03FF
	packet := []byte{
		byte(msgID >> 8), byte(msgID),
		byte(msgProps >> 8), byte(msgProps),
	}
	packet = append(packet, phone...)
	packet = append(packet, byte(serial>>8), byte(serial))
	packet = append(packet, body...)

	checksum := Checksum(packet)
	escapedPayload := Escape(append(packet, checksum))

	frame := []byte{FrameDelimiter}
	frame = append(frame, escapedPayload...)
	frame = append(frame, FrameDelimiter)
	return frame
}

// BuildGeneralResponse menyusun balasan umum platform (0x8001):
// "pesan nomor <serial> dengan ID <msgID> sudah kuterima, hasilnya <result>".
// result: 0 = berhasil, 1 = gagal, 2 = pesan salah, 3 = tidak didukung.
//
// TODO (Langkah 16): nomor serial pesan keluar masih tetap 1.
// Nanti diganti dengan penghitung yang bertambah di tiap pesan.
func BuildGeneralResponse(phone []byte, msgID, serial uint16, result byte) []byte {
	body := []byte{
		byte(serial >> 8), byte(serial),
		byte(msgID >> 8), byte(msgID),
		result,
	}
	return BuildMessage(0x8001, phone, 1, body)
}

// BuildRegisterResponse menyusun balasan registrasi terminal (0x8100)
// dengan hasil berhasil dan kode otentikasi "OK".
//
// TODO (Langkah 16): nomor serial pesan keluar masih tetap 1.
func BuildRegisterResponse(phone []byte, serial uint16) []byte {
	authCode := []byte("OK")
	body := append([]byte{
		byte(serial >> 8), byte(serial),
		0x00, // hasil: 0 = berhasil
	}, authCode...)
	return BuildMessage(0x8100, phone, 1, body)
}