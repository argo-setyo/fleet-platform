package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"
)

// =======================================================
// GLOBAL FLAG UNTUK VIDEO ALARM
// (nanti akan dipindah ke per-koneksi, bukan global)
// =======================================================
var hasVideoAlarm bool
var lastAlarmID uint32
var lastAlarmType byte

// =======================================================
// Fungsi utama (main) -> setup TCP server
// =======================================================
func main() {
	port := "38000"

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("JTT808 TCP Server listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println("New connection from:", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

// =======================================================
// Handle koneksi client
// =======================================================
func handleConnection(conn net.Conn) {
	defer conn.Close()

	var buffer []byte
	readBuf := make([]byte, 1024)

	for {
		conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
		n, err := conn.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client:", conn.RemoteAddr())
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}

		buffer = append(buffer, readBuf[:n]...)
		frames := extractFrames(&buffer)
		for _, frame := range frames {
			handleFrame(frame, conn)
		}
	}
}

// =======================================================
// Unescape & Escape Data JT/T 808
// =======================================================
func unescapeJT808(src []byte) []byte {
	var res []byte
	for i := 0; i < len(src); i++ {
		if src[i] == 0x7D && i+1 < len(src) {
			if src[i+1] == 0x02 {
				res = append(res, 0x7E)
				i++
				continue
			} else if src[i+1] == 0x01 {
				res = append(res, 0x7D)
				i++
				continue
			}
		}
		res = append(res, src[i])
	}
	return res
}

func escapeJT808(src []byte) []byte {
	var res []byte
	for _, b := range src {
		if b == 0x7E {
			res = append(res, 0x7D, 0x02)
		} else if b == 0x7D {
			res = append(res, 0x7D, 0x01)
		} else {
			res = append(res, b)
		}
	}
	return res
}

// =======================================================
// Mengekstrak frame JT/T 808 dari buffer (7E...7E)
// =======================================================
func extractFrames(buffer *[]byte) [][]byte {
	var frames [][]byte
	data := *buffer

	for {
		start := bytes.IndexByte(data, 0x7E)
		if start == -1 {
			break
		}
		end := bytes.IndexByte(data[start+1:], 0x7E)
		if end == -1 {
			break
		}
		end = start + 1 + end
		frame := data[start : end+1]
		frames = append(frames, frame)
		data = data[end+1:]
	}

	*buffer = data
	return frames
}

// =======================================================
// Handle frame: parsing + balasan jika perlu
// =======================================================
func handleFrame(frame []byte, conn net.Conn) {
	fmt.Println("====== Parsed JTT808 Frame =======================================================")
	parsed := parseJTT808Frame(frame)
	printParsedFrame(parsed)
	if !parsed.ChecksumValid {
		fmt.Println("!!! Checksum salah, frame diabaikan")
		return
	}
	switch parsed.MessageID {
	case 0x0100: // Registration
		resp := build8100Response(parsed.PhoneBCD, parsed.Serial)
		fmt.Println("====== SEND 0x8100 RESPONSE ======================================================")
		fmt.Printf("Raw HEX                 : %s\n", hex.EncodeToString(resp))
		parsedResp := parseJTT808Frame(resp)
		printParsedFrame(parsedResp)
		conn.Write(resp)

	case 0x0102: // Authentication
		resp := build8001Response(parsed.PhoneBCD, parsed.MessageID, parsed.Serial, 0x00)
		fmt.Println("====== SEND 0x8001 RESPONSE (AUTH SUCCESS) ======================================")
		fmt.Printf("Raw HEX                 : %s\n", hex.EncodeToString(resp))
		parsedResp := parseJTT808Frame(resp)
		printParsedFrame(parsedResp)
		conn.Write(resp)

	case 0x0002: // Heartbeat
		resp := build8001Response(parsed.PhoneBCD, parsed.MessageID, parsed.Serial, 0x00)
		fmt.Println("====== SEND 0x8001 RESPONSE ======================================================")
		fmt.Printf("Raw HEX                 : %s\n", hex.EncodeToString(resp))
		parsedResp := parseJTT808Frame(resp)
		printParsedFrame(parsedResp)
		conn.Write(resp)

	case 0x0200: // Location / Alarm Report
		resp := build8001Response(parsed.PhoneBCD, parsed.MessageID, parsed.Serial, 0x00)
		conn.Write(resp)

		// CEK VIDEO ALARM
		if hasVideoAlarm {
			fmt.Println(">>> VIDEO ALARM FOUND, READY TO REQUEST")
			// nanti di sini kirim 0x9208
		}
	}

	fmt.Println("=====================================================================================")
}

// =======================================================
// Struktur hasil parsing
// =======================================================
type ParsedFrame struct {
	MessageID     uint16
	MsgBodyProps  uint16
	PhoneBCD      []byte
	PhoneNumber   string
	Serial        uint16
	MsgBody       []byte
	Checksum      byte
	ChecksumValid bool
	Raw           []byte
}

// =======================================================
// Parsing frame JT/T 808
// =======================================================
func parseJTT808Frame(frame []byte) ParsedFrame {
	result := ParsedFrame{Raw: frame}
	if len(frame) < 3 {
		return result
	}

	// Unescape payload di antara delimiter 0x7E
	rawPayload := frame[1 : len(frame)-1]
	data := unescapeJT808(rawPayload)

	if len(data) < 13 {
		return result
	}

	result.MessageID = binary.BigEndian.Uint16(data[0:2])
	result.MsgBodyProps = binary.BigEndian.Uint16(data[2:4])
	result.PhoneBCD = data[4:10]
	result.PhoneNumber = parseBCDPhone(data[4:10])
	result.Serial = binary.BigEndian.Uint16(data[10:12])

	bodyLen := int(result.MsgBodyProps & 0x03FF)
	if bodyLen > len(data)-13 {
		bodyLen = len(data) - 13
	}
	result.MsgBody = data[12 : 12+bodyLen]
	result.Checksum = data[len(data)-1]

	cs := byte(0)
	for _, b := range data[:len(data)-1] {
		cs ^= b
	}
	result.ChecksumValid = (cs == result.Checksum)
	return result
}

// =======================================================
// Tampilkan hasil parsing frame
// =======================================================
func printParsedFrame(p ParsedFrame) {
	fmt.Printf("Raw HEX                 : %s\n", hex.EncodeToString(p.Raw))
	fmt.Printf("Message ID              : 0x%04X\n", p.MessageID)
	fmt.Printf("MsgBodyProps            : 0x%04X\n", p.MsgBodyProps)
	fmt.Printf("Phone Number            : %s\n", p.PhoneNumber)
	fmt.Printf("Serial Number           : %d\n", p.Serial)
	fmt.Printf("MsgBody HEX             : %s\n", hex.EncodeToString(p.MsgBody))

	switch p.MessageID {
	case 0x0100:
		parse0100Body(p.MsgBody)
	case 0x0102:
		parse0102Body(p.MsgBody)
	case 0x8001:
		parse8001Body(p.MsgBody)
	case 0x0200:
		parse0200Body(p.MsgBody)
	}

	fmt.Printf("Checksum                : 0x%02X (Computed: %v)\n", p.Checksum, p.ChecksumValid)
	if p.ChecksumValid {
		fmt.Println("Checksum valid ✓")
	} else {
		fmt.Println("Checksum INVALID ✗")
	}
}

// =======================================================
// Parsing message body 0x0100 (Registration)
// =======================================================
func parse0100Body(body []byte) {
	if len(body) < 37 {
		fmt.Println("Province ID             : (msgBody too short)")
		return
	}
	provinceID := binary.BigEndian.Uint16(body[0:2])
	cityID := binary.BigEndian.Uint16(body[2:4])
	manufacturer := string(bytes.Trim(body[4:9], "\x00"))
	model := string(bytes.Trim(body[9:29], "\x00"))
	terminalID := string(bytes.Trim(body[29:36], "\x00"))
	plateColor := body[36]
	plateNumber := ""
	if len(body) > 37 {
		plateNumber = string(body[37:])
	}

	fmt.Printf("Province ID             : %d (0x%04X)\n", provinceID, provinceID)
	fmt.Printf("City ID                 : %d (0x%04X)\n", cityID, cityID)
	fmt.Printf("Manufacturer ID         : %s (HEX: %s)\n", manufacturer, hex.EncodeToString(body[4:9]))
	fmt.Printf("Terminal Model          : %s (HEX: %s)\n", model, hex.EncodeToString(body[9:29]))
	fmt.Printf("Terminal ID             : %s (HEX: %s)\n", terminalID, hex.EncodeToString(body[29:36]))
	fmt.Printf("License Plate Color     : %d (0x%02X)\n", plateColor, plateColor)
	fmt.Printf("License Plate Number    : %s (HEX: %s)\n", plateNumber, hex.EncodeToString(body[37:]))
}

// =======================================================
// Parsing message body 0x0102 (Authentication)
// =======================================================
func parse0102Body(body []byte) {
	authCode := string(body)
	fmt.Printf("Auth Code               : %s (HEX: %s)\n", authCode, hex.EncodeToString(body))
}

// =======================================================
// Parsing message body 0x8001 (Generic Response)
// =======================================================
func parse8001Body(body []byte) {
	if len(body) < 5 {
		fmt.Println("Response too short")
		return
	}
	replySerial := binary.BigEndian.Uint16(body[0:2])
	msgID := binary.BigEndian.Uint16(body[2:4])
	result := body[4]
	fmt.Printf("Reply Serial            : %d (0x%04X)\n", replySerial, replySerial)
	fmt.Printf("Response MsgID          : 0x%04X\n", msgID)
	fmt.Printf("Result                  : %d (0x%02X)\n", result, result)
}

// =======================================================
// decodeLatLon mengubah nilai mentah JT808 menjadi derajat
// desimal bertanda, memakai bit 2 dan bit 3 dari status flag.
// =======================================================
func decodeLatLon(status, latRaw, lonRaw uint32) (lat, lon float64) {
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

// =======================================================
// Parsing message body 0x0200 (Location Report)
// =======================================================
func parse0200Body(body []byte) {
	if len(body) < 28 {
		fmt.Println("Location Report            : (msgBody too short)")
		return
	}

	alarm := binary.BigEndian.Uint32(body[0:4])
	status := binary.BigEndian.Uint32(body[4:8])
	latitudeRaw := binary.BigEndian.Uint32(body[8:12])
	longitudeRaw := binary.BigEndian.Uint32(body[12:16])
	elevation := binary.BigEndian.Uint16(body[16:18])
	speed := binary.BigEndian.Uint16(body[18:20])
	direction := binary.BigEndian.Uint16(body[20:22])
	timeBCD := body[22:28]

	latitude, longitude := decodeLatLon(status, latitudeRaw, longitudeRaw)

	positionTime := parseBCDTime(timeBCD)

	fmt.Printf("Alarm Flag              : 0x%08X (Binary: %032b)\n", alarm, alarm)
	fmt.Printf("Status Flag             : 0x%08X (Binary: %032b)\n", status, status)
	printStatusBits("Status Flag", status)
	fmt.Printf("Latitude                : %.6f\n", latitude)
	fmt.Printf("Longitude               : %.6f\n", longitude)
	fmt.Printf("Elevation               : %d m\n", elevation)
	fmt.Printf("Speed                   : %.1f km/h\n", float64(speed)/10)
	fmt.Printf("Direction               : %d°\n", direction)
	fmt.Printf("Positioning Time        : %s\n", positionTime)

	// =======================================================
	// Parsing Additional Information Items
	// =======================================================
	if len(body) > 28 {
		additional := body[28:]
		fmt.Println("====== Additional Information ===============================================")
		fmt.Printf("Raw HEX        \t\t: %s\n", hex.EncodeToString(additional))
		offset := 0
		for offset+2 <= len(additional) {
			itemID := additional[offset]
			itemLen := int(additional[offset+1])
			if offset+2+itemLen > len(additional) {
				break
			}
			itemContent := additional[offset+2 : offset+2+itemLen]

			// --- CEK ITEM 0x15: VIDEO SIGNAL LOSS / STATUS KAMERA ---
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

			// --- CEK ITEM ALARM KAMERA (ADAS/DMS/BSD) ---
			if (itemID == 0x64 || itemID == 0x65 || itemID == 0x67) && len(itemContent) >= 29 {
				itemAlarmID := binary.BigEndian.Uint32(itemContent[0:4])
				itemFlagStatus := uint8(itemContent[4])
				itemEvent := uint8(itemContent[5])
				itemAlarmLevel := uint8(itemContent[6])
				itemSpeed := uint8(itemContent[12])
				itemLatitudeRaw := binary.BigEndian.Uint32(itemContent[15:19])
				itemLongitudeRaw := binary.BigEndian.Uint32(itemContent[19:23])
				itemTimeRaw := itemContent[23:29]

				itemLatitude := float64(itemLatitudeRaw) / 1e6
				itemLongitude := float64(itemLongitudeRaw) / 1e6
				itemTime := parseItemBCDTime(itemTimeRaw)

				var alarmType string
				if itemID == 0x64 {
					alarmType = "ADAS Camera"
				} else if itemID == 0x65 {
					alarmType = "DMS Camera"
				} else if itemID == 0x67 {
					alarmType = "BSD Camera"
				} else {
					alarmType = fmt.Sprintf("0x%02X", itemID)
				}

				hasVideoAlarm = true
				lastAlarmID = itemAlarmID
				lastAlarmType = itemID

				fmt.Println(">>> VIDEO ALARM DETECTED")

				// Menggunakan fungsi pemetaan event type yang sesuai berdasarkan jenis kamera (ItemID)
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

			offset += 2 + itemLen
		}
	}
}

// =======================================================
// Fungsi Penerjemah Event Type
// (CATATAN: cocokkan dengan dokumen TJSATL milikmu)
// =======================================================
func parseEventTypeName(itemID byte, event byte) string {
	switch itemID {
	case 0x64: // ADAS Camera Events
		switch event {
		case 0x01:
			return "Forward Collision Warning (FCW)"
		case 0x02:
			return "Lane Departure Warning (LDW)"
		case 0x03:
			return "Headway Monitoring / Unsafe Distance"
		case 0x04:
			return "Pedestrian Collision Warning (PCW)"
		case 0x05:
			return "Frequent Lane Changing Warning"
		case 0x06:
			return "Road Sign Limit Violation"
		case 0x07:
			return "Obstacle Detected"
		case 0x08:
			return "Region / Area Intrusion Warning"
		default:
			return fmt.Sprintf("Unknown ADAS Event (0x%02X)", event)
		}

	case 0x65: // DMS Camera Events
		switch event {
		case 0x01:
			return "Fatigue Driving"
		case 0x02:
			return "Phone Calling"
		case 0x03:
			return "Smoking"
		case 0x04:
			return "Distracted Driving"
		case 0x05:
			return "Driver Not Detected / Absent"
		case 0x06:
			return "Camera Blocked / Occlusion"
		case 0x07:
			return "Infrared Sunglasses / Obstruction"
		case 0x08:
			return "Seatbelt Not Fastened"
		default:
			return fmt.Sprintf("Unknown DMS Event (0x%02X)", event)
		}

	case 0x67: // BSD Camera Events
		switch event {
		case 0x01:
			return "Rear/Side Pedestrian Detection Warning"
		case 0x02:
			return "Rear/Side Vehicle Collision Warning"
		default:
			return fmt.Sprintf("Unknown BSD Event (0x%02X)", event)
		}

	default:
		return fmt.Sprintf("0x%02X", event)
	}
}

// Helper untuk format teks status kamera
func statusCamera(connected bool) string {
	if connected {
		return "TERPASANG (OK)"
	}
	return "TERCABUT / LOSS ✗"
}

// =======================================================
// Cetak Bit Detail
// =======================================================
func printStatusBits(label string, status uint32) {
	fmt.Printf("%s Bit Fields:\n", label)
	fmt.Printf("  ACC on        : %d (%s)\n", (status>>0)&1, onOff((status>>0)&1))
	fmt.Printf("  Positioning   : %d (%s)\n", (status>>1)&1, yesNo((status>>1)&1))
	fmt.Printf("  SouthNorth lat: %d (%s)\n", (status>>2)&1, southNorth((status>>2)&1))
	fmt.Printf("  East West long: %d (%s)\n", (status>>3)&1, eastWest((status>>3)&1))
	fmt.Printf("  Stop status   : %d (%s)\n", (status>>4)&1, stopRun((status>>4)&1))
}

func onOff(v uint32) string {
	if v == 1 {
		return "ON"
	}
	return "OFF"
}

func yesNo(v uint32) string {
	if v == 1 {
		return "Valid"
	}
	return "Invalid"
}

func southNorth(v uint32) string {
	if v == 1 {
		return "South"
	}
	return "North"
}

func eastWest(v uint32) string {
	if v == 1 {
		return "West"
	}
	return "East"
}

func stopRun(v uint32) string {
	if v == 1 {
		return "Stopped"
	}
	return "Running"
}

// =======================================================
// Konversi BCD 6 byte -> string waktu
// =======================================================
func parseBCDTime(bcd []byte) string {
	if len(bcd) != 6 {
		return "(invalid BCD time)"
	}
	return fmt.Sprintf("20%02d-%02d-%02d %02d:%02d:%02d",
		bcdToInt(bcd[0]),
		bcdToInt(bcd[1]),
		bcdToInt(bcd[2]),
		bcdToInt(bcd[3]),
		bcdToInt(bcd[4]),
		bcdToInt(bcd[5]))
}

func parseItemBCDTime(bcd []byte) string {
	return parseBCDTime(bcd)
}

func bcdToInt(b byte) int {
	return int((b>>4)*10 + (b & 0x0F))
}

func parseBCDPhone(bcd []byte) string {
	var phone string
	for _, b := range bcd {
		hi := (b & 0xF0) >> 4
		lo := b & 0x0F
		phone += fmt.Sprintf("%d%d", hi, lo)
	}
	return phone
}

// =======================================================
// Bangun message 0x8100 (Registrasi Response)
// =======================================================
func build8100Response(phone []byte, serial uint16) []byte {
	authCode := []byte("OK")
	body := append([]byte{
		byte(serial >> 8), byte(serial),
		0x00,
	}, authCode...)
	return buildJTMessage(0x8100, phone, 1, body)
}

// =======================================================
// Bangun message 0x8001 (Universal Response)
// =======================================================
func build8001Response(phone []byte, msgID, serial uint16, result byte) []byte {
	body := []byte{
		byte(serial >> 8), byte(serial),
		byte(msgID >> 8), byte(msgID),
		result,
	}
	return buildJTMessage(0x8001, phone, 1, body)
}

// =======================================================
// Bangun frame JT/T 808 lengkap
// =======================================================
func buildJTMessage(msgID uint16, phone []byte, serial uint16, body []byte) []byte {
	msgProps := uint16(len(body)) & 0x03FF
	packet := []byte{
		byte(msgID >> 8), byte(msgID),
		byte(msgProps >> 8), byte(msgProps),
	}
	packet = append(packet, phone...)
	packet = append(packet, byte(serial>>8), byte(serial))
	packet = append(packet, body...)

	checksum := byte(0)
	for _, b := range packet {
		checksum ^= b
	}

	escapedPayload := escapeJT808(append(packet, checksum))

	frame := []byte{0x7E}
	frame = append(frame, escapedPayload...)
	frame = append(frame, 0x7E)
	return frame
}