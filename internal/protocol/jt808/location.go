package jt808

import (
	"encoding/binary"
	"errors"
)

// ErrBodyTooShort dikembalikan kalau isi pesan lebih pendek dari yang
// seharusnya, sehingga tidak bisa dibaca dengan aman.
var ErrBodyTooShort = errors.New("isi pesan terlalu pendek")

// ExtraItem adalah satu item tambahan di akhir laporan lokasi.
// Formatnya: ID (1 byte), panjang (1 byte), lalu isi.
type ExtraItem struct {
	ID   byte
	Data []byte
}

// Location adalah hasil membaca isi pesan laporan lokasi (0x0200).
type Location struct {
	Alarm     uint32  // alarm flag, tiap bit = satu jenis alarm (bit 0 = SOS)
	Status    uint32  // status flag (bit 0 ACC, bit 1 posisi valid, bit 2 selatan, bit 3 barat)
	Latitude  float64 // derajat desimal, negatif = selatan
	Longitude float64 // derajat desimal, negatif = barat
	Elevation uint16  // meter
	Speed     uint16  // satuan 0,1 km/jam
	Direction uint16  // derajat
	Time      string  // waktu perangkat apa adanya, mengikuti zona waktu di perangkat
	TimeBCD   []byte  // waktu mentah (6 byte BCD), dipakai untuk konversi ke UTC
	Extras    []ExtraItem
	ExtrasRaw []byte // byte mentah semua item tambahan
}

// DecodeLatLon mengubah nilai mentah JT808 menjadi derajat desimal bertanda,
// memakai bit 2 dan bit 3 dari status flag.
func DecodeLatLon(status, latRaw, lonRaw uint32) (lat, lon float64) {
	lat = float64(latRaw) / 1e6
	lon = float64(lonRaw) / 1e6
	if (status>>2)&1 == 1 { // bit 2: 0 = utara, 1 = selatan
		lat = -lat
	}
	if (status>>3)&1 == 1 { // bit 3: 0 = timur, 1 = barat
		lon = -lon
	}
	return
}

// ParseLocation membaca isi pesan 0x0200 (28 byte data dasar, lalu item
// tambahan kalau ada). Fungsi ini hanya membaca dan tidak mencetak apa pun.
func ParseLocation(body []byte) (Location, error) {
	if len(body) < 28 {
		return Location{}, ErrBodyTooShort
	}

	var loc Location
	loc.Alarm = binary.BigEndian.Uint32(body[0:4])
	loc.Status = binary.BigEndian.Uint32(body[4:8])
	latRaw := binary.BigEndian.Uint32(body[8:12])
	lonRaw := binary.BigEndian.Uint32(body[12:16])
	loc.Elevation = binary.BigEndian.Uint16(body[16:18])
	loc.Speed = binary.BigEndian.Uint16(body[18:20])
	loc.Direction = binary.BigEndian.Uint16(body[20:22])

	loc.Latitude, loc.Longitude = DecodeLatLon(loc.Status, latRaw, lonRaw)
	loc.TimeBCD = body[22:28]
	loc.Time = ParseBCDTime(loc.TimeBCD)

	if len(body) > 28 {
		loc.ExtrasRaw = body[28:]
		loc.Extras = parseExtras(loc.ExtrasRaw)
	}
	return loc, nil
}

// parseExtras memecah rangkaian item tambahan (ID, panjang, isi).
// Kalau ada item yang terpotong di ujung data, item itu dibuang.
func parseExtras(data []byte) []ExtraItem {
	var items []ExtraItem
	offset := 0
	for offset+2 <= len(data) {
		id := data[offset]
		n := int(data[offset+1])
		if offset+2+n > len(data) {
			break
		}
		items = append(items, ExtraItem{ID: id, Data: data[offset+2 : offset+2+n]})
		offset += 2 + n
	}
	return items
}