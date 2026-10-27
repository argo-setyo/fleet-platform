package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"fleet-platform/internal/protocol/jt808"
)

// Session menyimpan status satu koneksi perangkat.
//
// Tiap perangkat yang terhubung dilayani satu goroutine dengan Session-nya
// sendiri, jadi data di dalamnya tidak pernah dipakai bersama oleh perangkat
// lain. Itu yang membedakannya dari variabel global, yang dipakai bersama
// oleh semua koneksi dan bisa saling tertukar.
type Session struct {
	conn          net.Conn
	outSerial     uint16 // nomor serial pesan terakhir yang dikirim server
	hasVideoAlarm bool   // pernah menerima alarm kamera di koneksi ini
	lastAlarmID   uint32 // ID alarm kamera terakhir
	lastAlarmType byte   // jenis alarm kamera terakhir (0x64 ADAS, 0x65 DMS, 0x67 BSD)
}

func newSession(conn net.Conn) *Session {
	return &Session{conn: conn}
}

// nextSerial memberi nomor serial untuk pesan berikutnya dari server.
// Nomor bertambah satu tiap kali dipanggil (dimulai dari 1), dan kembali ke
// 0 sendiri setelah 65535 karena tipenya uint16.
func (s *Session) nextSerial() uint16 {
	s.outSerial++
	return s.outSerial
}

// send mengirim frame ke perangkat dan mencatat kalau gagal.
func (s *Session) send(frame []byte) {
	sent := jt808.ParseFrame(frame)
	fmt.Printf("<<< Kirim 0x%04X, serial %d\n", sent.MessageID, sent.Serial)

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