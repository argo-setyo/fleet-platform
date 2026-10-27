package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"fleet-platform/internal/protocol/jt808"
)

// =======================================================
// Mencetak isi pesan 0x0200 (Location Report).
// Pembacaan datanya dikerjakan jt808.ParseLocation;
// fungsi ini hanya menampilkan hasilnya ke terminal.
// =======================================================
func parse0200Body(body []byte) {
	loc, err := jt808.ParseLocation(body)
	if err != nil {
		fmt.Println("Location Report            :", err)
		return
	}

	fmt.Printf("Alarm Flag              : 0x%08X (Binary: %032b)\n", loc.Alarm, loc.Alarm)
	fmt.Printf("Status Flag             : 0x%08X (Binary: %032b)\n", loc.Status, loc.Status)
	printStatusBits("Status Flag", loc.Status)
	fmt.Printf("Latitude                : %.6f\n", loc.Latitude)
	fmt.Printf("Longitude               : %.6f\n", loc.Longitude)
	fmt.Printf("Elevation               : %d m\n", loc.Elevation)
	fmt.Printf("Speed                   : %.1f km/h\n", float64(loc.Speed)/10)
	fmt.Printf("Direction               : %d°\n", loc.Direction)
	fmt.Printf("Positioning Time        : %s\n", loc.Time)

	if len(loc.Extras) == 0 {
		return
	}

	fmt.Println("====== Additional Information ===============================================")
	fmt.Printf("Raw HEX        \t\t: %s\n", hex.EncodeToString(loc.ExtrasRaw))
	for _, item := range loc.Extras {
		printExtraItem(item)
	}
}

// =======================================================
// Mencetak satu item tambahan (status kamera dan alarm ADAS/DMS/BSD)
// =======================================================
func printExtraItem(item jt808.ExtraItem) {
	itemID := item.ID
	itemContent := item.Data
	itemLen := len(itemContent)

	// --- ITEM 0x15: VIDEO SIGNAL LOSS / STATUS KAMERA ---
	if itemID == 0x15 && itemLen == 4 {
		videoLossBits := binary.BigEndian.Uint32(itemContent)
		ch1Loss := (videoLossBits & 0x01) != 0
		ch2Loss := (videoLossBits & 0x02) != 0
		ch3Loss := (videoLossBits & 0x04) != 0
		ch4Loss := (videoLossBits & 0x08) != 0

		fmt.Println("-----------------------------------------------------------------------------")
		fmt.Printf("Item 0x15 - Camera Signal Status Raw : 0x%08X\n", videoLossBits)
		fmt.Printf("  - Channel 1 : %s\n", statusCamera(!ch1Loss))
		fmt.Printf("  - Channel 2 : %s\n", statusCamera(!ch2Loss))
		fmt.Printf("  - Channel 3 : %s\n", statusCamera(!ch3Loss))
		fmt.Printf("  - Channel 4 : %s\n", statusCamera(!ch4Loss))
		fmt.Println("-----------------------------------------------------------------------------")
	}

	// --- ITEM ALARM KAMERA (ADAS 0x64 / DMS 0x65 / BSD 0x67) ---
	if (itemID == 0x64 || itemID == 0x65 || itemID == 0x67) && itemLen >= 29 {
		itemAlarmID := binary.BigEndian.Uint32(itemContent[0:4])
		itemFlagStatus := itemContent[4]
		itemEvent := itemContent[5]
		itemAlarmLevel := itemContent[6]
		itemSpeed := itemContent[12]
		itemLatitudeRaw := binary.BigEndian.Uint32(itemContent[15:19])
		itemLongitudeRaw := binary.BigEndian.Uint32(itemContent[19:23])
		itemTime := jt808.ParseBCDTime(itemContent[23:29])

		itemLatitude := float64(itemLatitudeRaw) / 1e6
		itemLongitude := float64(itemLongitudeRaw) / 1e6

		var alarmType string
		switch itemID {
		case 0x64:
			alarmType = "ADAS Camera"
		case 0x65:
			alarmType = "DMS Camera"
		case 0x67:
			alarmType = "BSD Camera"
		}

		hasVideoAlarm = true
		lastAlarmID = itemAlarmID
		lastAlarmType = itemID

		fmt.Println(">>> VIDEO ALARM DETECTED")

		eventType := parseEventTypeName(itemID, itemEvent)

		fmt.Printf("Alarm Type \t\t: %s (0x%02X)\n", alarmType, itemID)
		fmt.Printf("Alarm Length \t\t: %d (0x%02X)\n", itemLen, itemLen)
		fmt.Printf("Alarm Content \t\t: %s\n", hex.EncodeToString(itemContent))
		fmt.Printf("Alarm ID \t\t: %d (0x%02X)\n", itemAlarmID, itemAlarmID)
		fmt.Printf("Flag Status \t\t: %d (0x%02X)\n", itemFlagStatus, itemFlagStatus)
		fmt.Printf("Event Type \t\t: %s (0x%02X)\n", eventType, itemEvent)
		fmt.Printf("Alarm Level \t\t: %d (0x%02X)\n", itemAlarmLevel, itemAlarmLevel)
		fmt.Printf("Speed   \t\t: %d (0x%02X)\n", itemSpeed, itemSpeed)
		fmt.Printf("Latitude \t\t: %.6f\n", itemLatitude)
		fmt.Printf("Longitude \t\t: %.6f\n", itemLongitude)
		fmt.Printf("Item Time \t\t: %s\n", itemTime)
		fmt.Println("=============================================================================")
	}
}