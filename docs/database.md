# Database

PostgreSQL 18, dijalankan lewat Docker (docker-compose.yml).
Struktur tabel dikelola dengan goose (folder migrations/).

## Aturan penting

- Semua waktu disimpan sebagai UTC (TIMESTAMPTZ).
- Waktu dari perangkat dikonversi memakai devices.timezone_offset_minutes
  (UTC+7 = 420) saat data masuk.

## Tabel devices

Satu baris per perangkat, dikenali dari nomor terminal (phone).
Didaftarkan otomatis saat pertama terhubung (sementara, nanti diganti
daftar perangkat resmi).

## Tabel locations

Satu baris per titik lokasi. Kunci utama (device_id, device_time):
titik yang sama yang dikirim ulang tidak menggandakan data.

| Kolom | Arti |
|---|---|
| device_time | Waktu titik, UTC |
| received_at | Waktu server menerima data |
| is_backfill | true kalau berasal dari data susulan (0x0704 jenis 1) |
| alarm_flags | Bit alarm dari pesan 0x0200 (arti tiap bit: TODO cek dokumen perangkat) |