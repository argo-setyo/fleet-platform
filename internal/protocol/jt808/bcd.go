package jt808

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTime dikembalikan kalau waktu BCD bukan tanggal dan jam yang sah.
var ErrInvalidTime = errors.New("waktu BCD tidak valid")

// bcdToInt mengubah satu byte BCD menjadi angka desimal.
// BCD menyimpan satu digit desimal di tiap setengah byte,
// jadi byte 0x26 berarti angka 26.
func bcdToInt(b byte) int {
	return int((b>>4)*10 + (b & 0x0F))
}

// isBCD memeriksa bahwa kedua setengah byte berisi digit 0-9.
func isBCD(b byte) bool {
	return b>>4 <= 9 && b&0x0F <= 9
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

// BCDTimeToUTC mengubah waktu 6 byte BCD (YYMMDDhhmmss) menjadi waktu UTC.
//
// Waktu di dalam pesan ditulis menurut zona waktu yang diatur di perangkat.
// offsetMinutes adalah selisih zona itu terhadap UTC, dalam menit
// (UTC+7 = 420, UTC+8 = 480). Kalau isinya bukan tanggal dan jam yang sah,
// fungsi ini mengembalikan ErrInvalidTime.
func BCDTimeToUTC(bcd []byte, offsetMinutes int) (time.Time, error) {
	if len(bcd) != 6 {
		return time.Time{}, ErrInvalidTime
	}
	for _, b := range bcd {
		if !isBCD(b) {
			return time.Time{}, ErrInvalidTime
		}
	}

	year := 2000 + bcdToInt(bcd[0])
	month := bcdToInt(bcd[1])
	day := bcdToInt(bcd[2])
	hour := bcdToInt(bcd[3])
	minute := bcdToInt(bcd[4])
	second := bcdToInt(bcd[5])

	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, ErrInvalidTime
	}

	zone := time.FixedZone("device", offsetMinutes*60)
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, zone)

	// time.Date menggeser tanggal yang mustahil (misalnya 31 Februari)
	// ke bulan berikutnya tanpa memberi tahu. Tolak yang seperti itu.
	if t.Day() != day {
		return time.Time{}, ErrInvalidTime
	}
	return t.UTC(), nil
}