package jt808

import "encoding/binary"

// Jenis data pada pesan 0x0704.
const (
	BatchNormal    = 0 // unggahan massal biasa
	BatchBlindFill = 1 // data susulan (tersimpan di perangkat saat tidak ada koneksi)
)

// LocationBatch adalah hasil membaca isi pesan 0x0704 (unggahan lokasi massal):
// jenis data dan sekumpulan laporan lokasi.
type LocationBatch struct {
	Type      byte
	Locations []Location
}

// ParseLocationBatch membaca isi pesan 0x0704.
//
// Formatnya: jumlah data (2 byte), jenis data (1 byte), lalu untuk tiap data:
// panjang (2 byte) dan isi laporan lokasi yang formatnya sama dengan 0x0200.
//
// Kalau data terpotong di tengah, fungsi ini mengembalikan laporan yang sudah
// berhasil dibaca sebelumnya bersama error, supaya data yang utuh tidak ikut
// terbuang.
func ParseLocationBatch(body []byte) (LocationBatch, error) {
	if len(body) < 3 {
		return LocationBatch{}, ErrBodyTooShort
	}

	count := int(binary.BigEndian.Uint16(body[0:2]))
	batch := LocationBatch{Type: body[2]}
	offset := 3

	for i := 0; i < count; i++ {
		if offset+2 > len(body) {
			return batch, ErrBodyTooShort
		}
		n := int(binary.BigEndian.Uint16(body[offset : offset+2]))
		offset += 2

		if offset+n > len(body) {
			return batch, ErrBodyTooShort
		}
		loc, err := ParseLocation(body[offset : offset+n])
		if err != nil {
			return batch, err
		}
		batch.Locations = append(batch.Locations, loc)
		offset += n
	}
	return batch, nil
}