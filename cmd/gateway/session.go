package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"fleet-platform/internal/protocol/jt808"
	"fleet-platform/internal/store"
)

// dbTimeout adalah batas waktu tiap perintah ke database.
const dbTimeout = 5 * time.Second

// lastSeenInterval adalah jeda minimal antar pembaruan last_seen_at.
// Tanpa jeda ini, tiap pesan (bisa tiap beberapa detik) akan menulis ke
// database hanya untuk memperbarui satu kolom waktu.
const lastSeenInterval = 30 * time.Second

// LocationStore adalah bagian dari database yang dibutuhkan gateway.
// Dengan interface ini, kode gateway tidak bergantung langsung pada
// PostgreSQL, dan di tes kita bisa memakai "database palsu".
type LocationStore interface {
	EnsureDevice(ctx context.Context, phone string) (store.Device, error)
	InsertLocation(ctx context.Context, r store.LocationRecord) (bool, error)
	TouchDevice(ctx context.Context, deviceID int64) error
}

// Session menyimpan status satu koneksi perangkat.
//
// Tiap perangkat yang terhubung dilayani satu goroutine dengan Session-nya
// sendiri, jadi data di dalamnya tidak pernah dipakai bersama oleh perangkat
// lain. Itu yang membedakannya dari variabel global, yang dipakai bersama
// oleh semua koneksi dan bisa saling tertukar.
type Session struct {
	conn          net.Conn
	db            LocationStore // nil = mode cetak saja, data tidak disimpan
	device        *store.Device // diisi setelah perangkat dikenali di database
	outSerial     uint16        // nomor serial pesan terakhir yang dikirim server
	lastTouch     time.Time     // kapan last_seen_at terakhir diperbarui
	now           func() time.Time // jam; kosong berarti jam sistem (diganti di tes)
	hasVideoAlarm bool          // pernah menerima alarm kamera di koneksi ini
	lastAlarmID   uint32        // ID alarm kamera terakhir
	lastAlarmType byte          // jenis alarm kamera terakhir (0x64 ADAS, 0x65 DMS, 0x67 BSD)
}

func newSession(conn net.Conn, db LocationStore) *Session {
	return &Session{conn: conn, db: db}
}

// nextSerial memberi nomor serial untuk pesan berikutnya dari server.
// Nomor bertambah satu tiap kali dipanggil (dimulai dari 1), dan kembali ke
// 0 sendiri setelah 65535 karena tipenya uint16.
func (s *Session) nextSerial() uint16 {
	s.outSerial++
	return s.outSerial
}

// clock mengembalikan waktu sekarang. Di tes, jam bisa diganti lewat s.now.
func (s *Session) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// send mengirim frame ke perangkat dan mencatat kalau gagal.
// Di mode ringkas, satu baris tentang pesan yang dikirim ikut dicetak.
func (s *Session) send(frame []byte) {
	if !debugMode {
		sent := jt808.ParseFrame(frame)
		fmt.Printf("%s %s -> 0x%04X #%d\n", time.Now().Format("15:04:05"), sent.PhoneNumber, sent.MessageID, sent.Serial)
	}
	if _, err := s.conn.Write(frame); err != nil {
		fmt.Println("Gagal mengirim ke", s.conn.RemoteAddr(), ":", err)
	}
}

// noteVideoAlarm mencatat alarm kamera (ADAS 0x64, DMS 0x65, BSD 0x67)
// yang ada di dalam laporan lokasi.
func (s *Session) noteVideoAlarm(loc jt808.Location) {
	for _, item := range loc.Extras {
		isCameraAlarm := item.ID == 0x64 || item.ID == 0x65 || item.ID == 0x67
		if isCameraAlarm && len(item.Data) >= 29 {
			s.hasVideoAlarm = true
			s.lastAlarmID = binary.BigEndian.Uint32(item.Data[0:4])
			s.lastAlarmType = item.ID
		}
	}
}

// ensureDevice mengenali perangkat di database berdasarkan nomor terminal.
// Dikerjakan sekali per koneksi. Kalau database tidak dipakai, tidak
// melakukan apa pun.
func (s *Session) ensureDevice(phone string) error {
	if s.db == nil || s.device != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	dev, err := s.db.EnsureDevice(ctx, phone)
	if err != nil {
		return err
	}
	s.device = &dev
	s.lastTouch = s.clock() // EnsureDevice sudah memperbarui last_seen_at
	return nil
}

// touch memperbarui last_seen_at perangkat di database, paling sering sekali
// per lastSeenInterval, supaya status online/offline bisa diketahui tanpa
// menulis ke database di setiap pesan.
func (s *Session) touch() {
	if s.db == nil || s.device == nil {
		return
	}
	now := s.clock()
	if now.Sub(s.lastTouch) < lastSeenInterval {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := s.db.TouchDevice(ctx, s.device.ID); err != nil {
		fmt.Println("Gagal memperbarui last_seen_at:", err)
		return
	}
	s.lastTouch = now
}

// saveLocations menyimpan sekumpulan laporan lokasi ke database dan
// mengembalikan jumlah titik yang benar-benar baru tersimpan (titik yang
// sudah ada tidak dihitung). backfill menandai data susulan.
//
// Titik dengan waktu atau koordinat yang mustahil dilewati dan dicatat di
// layar, bukan dianggap gagal, supaya satu data rusak tidak membuat
// perangkat mengirim ulang terus-menerus.
func (s *Session) saveLocations(locs []jt808.Location, backfill bool) (int, error) {
	if s.db == nil {
		return 0, nil
	}
	if s.device == nil {
		return 0, errors.New("perangkat belum dikenali di database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	saved := 0
	for _, loc := range locs {
		deviceTime, err := jt808.BCDTimeToUTC(loc.TimeBCD, s.device.TimezoneOffsetMinutes)
		if err != nil {
			fmt.Println("Lokasi dilewati, waktu tidak valid:", err)
			continue
		}
		if loc.Latitude < -90 || loc.Latitude > 90 || loc.Longitude < -180 || loc.Longitude > 180 {
			fmt.Println("Lokasi dilewati, koordinat di luar batas")
			continue
		}

		inserted, err := s.db.InsertLocation(ctx, store.LocationRecord{
			DeviceID:     s.device.ID,
			DeviceTime:   deviceTime,
			Latitude:     loc.Latitude,
			Longitude:    loc.Longitude,
			AltitudeM:    int32(loc.Elevation),
			SpeedKmh:     float32(loc.Speed) / 10,
			DirectionDeg: int16(loc.Direction % 360),
			AlarmFlags:   int64(loc.Alarm),
			StatusFlags:  int64(loc.Status),
			IsBackfill:   backfill,
		})
		if err != nil {
			return saved, err
		}
		if inserted {
			saved++
		}
	}
	return saved, nil
}