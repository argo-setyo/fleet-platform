# Pesan JT808 yang didukung

Versi protokol: JT808-2013 (header 12 byte, nomor terminal BCD 6 byte).

## Struktur frame

    7E | Header (12 byte) | Body | Checksum (1 byte) | 7E

- Penanda awal dan akhir: `7E`
- Escape: `7E` di dalam data ditulis `7D 02`, `7D` ditulis `7D 01`
- Checksum: XOR seluruh byte dari header sampai akhir body

## 0x0200 - Laporan lokasi (terminal ke platform)

| Offset | Panjang | Field | Contoh | Catatan |
|---|---|---|---|---|
| 0 | 4 | Alarm flag | 00000001 | Bit 0 = SOS |
| 4 | 4 | Status | 000c0007 | Bit 0 ACC, bit 1 posisi valid, bit 2 selatan, bit 3 barat |
| 8 | 4 | Lintang | 00777c09 | Dibagi 1.000.000 |
| 12 | 4 | Bujur | 0694cfcc | Dibagi 1.000.000 |
| 16 | 2 | Ketinggian | 005c | Meter |
| 18 | 2 | Kecepatan | 0000 | Satuan 0,1 km/jam |
| 20 | 2 | Arah | 0000 | Derajat |
| 22 | 6 | Waktu | 261026153006 | BCD YYMMDDhhmmss. Mengikuti pengaturan zona waktu MDVR (saat ini GMT+07:00). Standar JT/T 808 umumnya menyebut GMT+8, jadi zona waktu per perangkat harus disimpan di sistem |
| 28 | ... | Item tambahan | | Format: ID (1 byte), panjang (1 byte), isi |

### Item tambahan yang terlihat di perangkat

| ID | Panjang | Contoh | Arti |
|---|---|---|---|
| 0x01 | 4 | 00000001 | Jarak tempuh, satuan 0,1 km |
| 0x03 | 2 | 0000 | Kecepatan dari perekam |
| 0x14 | 4 | 00000004 | Flag alarm video |
| 0x15 | 4 | 00000000 | Status sinyal video hilang, tiap bit = satu kanal |
| 0x16 | 4 | 00000000 | Status video tertutup |
| 0x17 | 2 | 0006 | Status kerusakan penyimpanan |
| 0x25 | 4 | 00000000 | Status sinyal kendaraan tambahan |
| 0x2A | 2 | 0000 | Status IO |
| 0x30 | 1 | 00 | Kekuatan sinyal jaringan |
| 0x31 | 1 | 14 | Jumlah satelit GNSS (0x14 = 20) |
| 0x18, 0x19, 0x52, 0xAD | | | TODO: cek dokumen perangkat |

## Frame nyata (disingkat)

    7e 0200 005e 202505250001 0020 ...body... ae 7e

- ID pesan 0x0200, panjang body 94 byte (0x005e)
- Nomor terminal 202505250001, serial 32

## Urutan koneksi perangkat

1. 0x0100 registrasi (hanya saat pertama kali), dibalas 0x8100 dengan kode otentikasi
2. 0x0102 otentikasi dengan kode tersebut, dibalas 0x8001
3. 0x0900 (data transparan, jenis 0xA1), dibalas 0x8001
4. 0x0200 laporan lokasi berkala, dibalas 0x8001
5. 0x0107 atribut terminal, tidak dibalas (jawaban atas pertanyaan platform)

## 0x0704 - Unggahan lokasi massal (terminal ke platform)

| Offset | Panjang | Field | Contoh | Catatan |
|---|---|---|---|---|
| 0 | 2 | Jumlah data | 0001 | |
| 2 | 1 | Jenis data | 01 | 0 = massal biasa, 1 = data susulan |
| 3 | 2 | Panjang data ke-1 | 005e | 94 byte |
| 5 | n | Isi data ke-1 | ... | Format sama dengan isi 0x0200 |

Dibalas dengan 0x8001.