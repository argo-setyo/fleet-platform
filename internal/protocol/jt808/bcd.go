package jt808

import "fmt"

// bcdToInt mengubah satu byte BCD menjadi angka desimal.
// BCD menyimpan satu digit desimal di tiap setengah byte,
// jadi byte 0x26 berarti angka 26.
func bcdToInt(b byte) int {
	return int((b>>4)*10 + (b & 0x0F))
}

// ParseBCDPhone mengubah nomor terminal (BCD) menjadi teks digit.
// Contoh: 20 25 05 25 00 01 menjadi "202505250001".
func ParseBCDPhone(bcd []byte) string {
	var phone string
	for _, b := range bcd {
		hi := (b & 0xF0) >> 4
		lo := b & 0x0F
		phone += fmt.Sprintf("%d%d", hi, lo)
	}
	return phone
}

// ParseBCDTime mengubah waktu 6 byte BCD (YYMMDDhhmmss) menjadi teks
// "20YY-MM-DD hh:mm:ss". Zona waktu mengikuti pengaturan perangkat.
func ParseBCDTime(bcd []byte) string {
	if len(bcd) != 6 {
		return "(invalid BCD time)"
	}
	return fmt.Sprintf("20%02d-%02d-%02d %02d:%02d:%02d",
		bcdToInt(bcd[0]),
		bcdToInt(bcd[1]),
		bcdToInt(bcd[2]),
		bcdToInt(bcd[3]),
		bcdToInt(bcd[4]),
		bcdToInt(bcd[5]))
}