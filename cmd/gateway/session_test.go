package main

import (
	"context"
	"math"
	"testing"
	"time"

	"fleet-platform/internal/protocol/jt808"
	"fleet-platform/internal/store"
)

// fakeStore adalah "database palsu" yang hanya mencatat apa yang diminta.
type fakeStore struct {
	timezoneOffset int
	ensured        []string
	records        []store.LocationRecord
}

func (f *fakeStore) EnsureDevice(ctx context.Context, phone string) (store.Device, error) {
	f.ensured = append(f.ensured, phone)
	return store.Device{ID: 42, TimezoneOffsetMinutes: f.timezoneOffset}, nil
}

func (f *fakeStore) InsertLocation(ctx context.Context, r store.LocationRecord) (bool, error) {
	f.records = append(f.records, r)
	return true, nil
}

func TestSessionNextSerial(t *testing.T) {
	s := &Session{}

	for want := uint16(1); want <= 3; want++ {
		if got := s.nextSerial(); got != want {
			t.Errorf("nextSerial = %d, want %d", got, want)
		}
	}
}

func TestSessionsAreIndependent(t *testing.T) {
	a := &Session{}
	b := &Session{}

	a.nextSerial()
	a.nextSerial()

	if got := b.nextSerial(); got != 1 {
		t.Errorf("serial sesi b = %d, want 1 (tidak terpengaruh sesi a)", got)
	}
}

func TestNoteVideoAlarm(t *testing.T) {
	// Isi item alarm: 4 byte pertama = ID alarm (7), total minimal 29 byte
	data := make([]byte, 29)
	data[3] = 0x07

	loc := jt808.Location{
		Extras: []jt808.ExtraItem{{ID: 0x65, Data: data}},
	}

	s := &Session{}
	s.noteVideoAlarm(loc)

	if !s.hasVideoAlarm {
		t.Fatal("hasVideoAlarm seharusnya true")
	}
	if s.lastAlarmID != 7 {
		t.Errorf("lastAlarmID = %d, want 7", s.lastAlarmID)
	}
	if s.lastAlarmType != 0x65 {
		t.Errorf("lastAlarmType = 0x%02X, want 0x65", s.lastAlarmType)
	}
}

func TestNoteVideoAlarmIgnoresShortOrOtherItems(t *testing.T) {
	loc := jt808.Location{
		Extras: []jt808.ExtraItem{
			{ID: 0x65, Data: make([]byte, 10)}, // alarm kamera tapi terlalu pendek
			{ID: 0x31, Data: []byte{0x1c}},     // bukan alarm kamera
		},
	}

	s := &Session{}
	s.noteVideoAlarm(loc)

	if s.hasVideoAlarm {
		t.Error("hasVideoAlarm seharusnya tetap false")
	}
}

// Waktu perangkat 10:51:19 di UTC+7 harus tersimpan sebagai 03:51:19 UTC.
func TestSaveLocationsConvertsTimeToUTC(t *testing.T) {
	fake := &fakeStore{timezoneOffset: 420}
	s := &Session{db: fake}

	if err := s.ensureDevice("202505250001"); err != nil {
		t.Fatal(err)
	}

	loc := jt808.Location{
		TimeBCD:   []byte{0x26, 0x10, 0x27, 0x10, 0x51, 0x19},
		Latitude:  -7.830589,
		Longitude: 110.415860,
		Elevation: 85,
		Speed:     123, // satuan 0,1 km/jam, jadi 12,3 km/jam
		Direction: 90,
		Alarm:     1,
		Status:    0x000C0007,
	}

	saved, err := s.saveLocations([]jt808.Location{loc}, true)
	if err != nil {
		t.Fatal(err)
	}
	if saved != 1 || len(fake.records) != 1 {
		t.Fatalf("saved = %d, records = %d, want 1 dan 1", saved, len(fake.records))
	}

	r := fake.records[0]
	wantTime := time.Date(2026, 10, 27, 3, 51, 19, 0, time.UTC)
	if !r.DeviceTime.Equal(wantTime) {
		t.Errorf("DeviceTime = %v, want %v", r.DeviceTime, wantTime)
	}
	if r.DeviceID != 42 {
		t.Errorf("DeviceID = %d, want 42", r.DeviceID)
	}
	if !r.IsBackfill {
		t.Error("IsBackfill seharusnya true")
	}
	if math.Abs(float64(r.SpeedKmh)-12.3) > 0.001 {
		t.Errorf("SpeedKmh = %v, want 12.3", r.SpeedKmh)
	}
	if r.AlarmFlags != 1 || r.StatusFlags != 0x000C0007 {
		t.Errorf("flags = %d / 0x%X, want 1 / 0xC0007", r.AlarmFlags, r.StatusFlags)
	}
}

func TestSaveLocationsWithoutDatabase(t *testing.T) {
	s := &Session{} // db nil: mode cetak saja

	saved, err := s.saveLocations([]jt808.Location{{}}, false)
	if saved != 0 || err != nil {
		t.Errorf("saved = %d, err = %v, want 0 dan nil", saved, err)
	}
}

func TestSaveLocationsRequiresKnownDevice(t *testing.T) {
	s := &Session{db: &fakeStore{}} // ensureDevice belum dipanggil

	if _, err := s.saveLocations([]jt808.Location{{}}, false); err == nil {
		t.Error("seharusnya error karena perangkat belum dikenali")
	}
}

func TestSaveLocationsSkipsInvalidData(t *testing.T) {
	fake := &fakeStore{timezoneOffset: 420}
	s := &Session{db: fake}
	if err := s.ensureDevice("202505250001"); err != nil {
		t.Fatal(err)
	}

	badTime := jt808.Location{TimeBCD: []byte{0x26, 0x13, 0x01, 0x00, 0x00, 0x00}} // bulan 13
	badCoord := jt808.Location{
		TimeBCD:  []byte{0x26, 0x10, 0x27, 0x10, 0x51, 0x19},
		Latitude: 91,
	}

	saved, err := s.saveLocations([]jt808.Location{badTime, badCoord}, false)
	if err != nil {
		t.Fatalf("data rusak seharusnya dilewati tanpa error, dapat: %v", err)
	}
	if saved != 0 || len(fake.records) != 0 {
		t.Errorf("saved = %d, records = %d, want 0 dan 0", saved, len(fake.records))
	}
}

func TestEnsureDeviceOnlyOncePerSession(t *testing.T) {
	fake := &fakeStore{}
	s := &Session{db: fake}

	s.ensureDevice("202505250001")
	s.ensureDevice("202505250001")

	if len(fake.ensured) != 1 {
		t.Errorf("EnsureDevice dipanggil %d kali, want 1", len(fake.ensured))
	}
}