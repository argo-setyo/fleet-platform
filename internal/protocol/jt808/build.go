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

// BuildGeneralResponse menyusun balasan umum platform (0x8001) untuk pesan
// req yang baru diterima: "pesan dengan serial <req.Serial> dan ID
// <req.MessageID> sudah kuterima, hasilnya <result>".
//
// outSerial adalah nomor serial pesan balasan ini sendiri. Nomor itu
// bertambah di tiap pesan yang dikirim server, dan dihitung oleh Session.
//
// result: 0 = berhasil, 1 = gagal, 2 = pesan salah, 3 = tidak didukung.
func BuildGeneralResponse(req Frame, outSerial uint16, result byte) []byte {
	body := []byte{
		byte(req.Serial >> 8), byte(req.Serial),
		byte(req.MessageID >> 8), byte(req.MessageID),
		result,
	}
	return BuildMessage(0x8001, req.PhoneBCD, outSerial, body)
}

// BuildRegisterResponse menyusun balasan registrasi terminal (0x8100) untuk
// pesan registrasi req, dengan hasil berhasil dan kode otentikasi "OK".
// outSerial adalah nomor serial pesan balasan ini sendiri.
func BuildRegisterResponse(req Frame, outSerial uint16) []byte {
	authCode := []byte("OK")
	body := append([]byte{
		byte(req.Serial >> 8), byte(req.Serial),
		0x00, // hasil: 0 = berhasil
	}, authCode...)
	return BuildMessage(0x8100, req.PhoneBCD, outSerial, body)
}