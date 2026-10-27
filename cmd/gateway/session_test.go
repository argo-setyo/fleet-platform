package main

import (
	"testing"

	"fleet-platform/internal/protocol/jt808"
)

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