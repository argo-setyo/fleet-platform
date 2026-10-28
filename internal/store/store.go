// Package store menyimpan data gateway ke PostgreSQL.
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Device adalah perangkat yang dikenali di database.
type Device struct {
	ID                    int64
	TimezoneOffsetMinutes int // selisih zona waktu jam perangkat terhadap UTC, dalam menit
}

// LocationRecord adalah satu titik lokasi yang siap disimpan.
type LocationRecord struct {
	DeviceID     int64
	DeviceTime   time.Time // sudah dalam UTC
	Latitude     float64
	Longitude    float64
	AltitudeM    int32
	SpeedKmh     float32
	DirectionDeg int16
	AlarmFlags   int64
	StatusFlags  int64
	IsBackfill   bool // berasal dari data susulan (0x0704)
}

// Store membungkus kumpulan koneksi (pool) ke PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// Open membuka kumpulan koneksi ke database dan memastikan database
// memang bisa dihubungi. databaseURL berisi alamat dan password database.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close menutup semua koneksi.
func (s *Store) Close() {
	s.pool.Close()
}

// EnsureDevice mencari perangkat berdasarkan nomor terminal. Kalau belum ada,
// perangkat dibuat. Waktu terakhir terlihat (last_seen_at) ikut diperbarui.
//
// Catatan: untuk tahap belajar, perangkat apa pun yang terhubung didaftarkan
// otomatis. Sebelum dipakai klien, ini harus diganti dengan daftar perangkat
// resmi, dan perangkat yang tidak terdaftar ditolak.
func (s *Store) EnsureDevice(ctx context.Context, phone string) (Device, error) {
	const query = `
INSERT INTO devices (phone, last_seen_at)
VALUES ($1, now())
ON CONFLICT (phone) DO UPDATE SET last_seen_at = now()
RETURNING id, timezone_offset_minutes`

	var d Device
	var offset int16
	if err := s.pool.QueryRow(ctx, query, phone).Scan(&d.ID, &offset); err != nil {
		return Device{}, err
	}
	d.TimezoneOffsetMinutes = int(offset)
	return d, nil
}

// InsertLocation menyimpan satu titik lokasi. Kalau perangkat yang sama sudah
// punya titik pada waktu yang sama, titik itu dilewati. Hasilnya true kalau
// baris baru benar-benar tersimpan.
func (s *Store) InsertLocation(ctx context.Context, r LocationRecord) (bool, error) {
	const query = `
INSERT INTO locations (
    device_id, device_time, latitude, longitude, altitude_m,
    speed_kmh, direction_deg, alarm_flags, status_flags, is_backfill
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (device_id, device_time) DO NOTHING`

	tag, err := s.pool.Exec(ctx, query,
		r.DeviceID, r.DeviceTime, r.Latitude, r.Longitude, r.AltitudeM,
		r.SpeedKmh, r.DirectionDeg, r.AlarmFlags, r.StatusFlags, r.IsBackfill,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// TouchDevice memperbarui last_seen_at perangkat ke waktu sekarang, tanda
// bahwa perangkat masih terhubung dan mengirim data.
func (s *Store) TouchDevice(ctx context.Context, deviceID int64) error {
	_, err := s.pool.Exec(ctx, "UPDATE devices SET last_seen_at = now() WHERE id = $1", deviceID)
	return err
}