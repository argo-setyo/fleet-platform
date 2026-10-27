package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"

	"fleet-platform/internal/protocol/jt808"
)

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

	// Setiap koneksi punya Session sendiri
	s := newSession(conn)

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
		frames := jt808.ExtractFrames(&buffer)
		for _, frame := range frames {
			handleFrame(frame, s)
		}
	}
}

// =======================================================
// Handle frame: parsing + balasan jika perlu
// =======================================================
func handleFrame(frame []byte, s *Session) {
	fmt.Println("====== Parsed JTT808 Frame =======================================================")
	parsed := jt808.ParseFrame(frame)
	printParsedFrame(parsed)

	if !parsed.ChecksumValid {
		fmt.Println("!!! Checksum salah, frame diabaikan")
		return
	}

	switch parsed.MessageID {
	case 0x0100: // Registration
		resp := jt808.BuildRegisterResponse(parsed, s.nextSerial())
		fmt.Println("====== SEND 0x8100 RESPONSE ======================================================")
		printParsedFrame(jt808.ParseFrame(resp))
		s.send(resp)

	case 0x0102: // Authentication
		resp := jt808.BuildGeneralResponse(parsed, s.nextSerial(), 0x00)
		fmt.Println("====== SEND 0x8001 RESPONSE (AUTH SUCCESS) ======================================")
		printParsedFrame(jt808.ParseFrame(resp))
		s.send(resp)

	case 0x0002: // Heartbeat
		resp := jt808.BuildGeneralResponse(parsed, s.nextSerial(), 0x00)
		fmt.Println("====== SEND 0x8001 RESPONSE ======================================================")
		printParsedFrame(jt808.ParseFrame(resp))
		s.send(resp)

	case 0x0704, 0x0900: // Upload data massal dan data transparan: cukup dibalas 0x8001
		s.send(jt808.BuildGeneralResponse(parsed, s.nextSerial(), 0x00))
		fmt.Printf(">>> Balas 0x8001 untuk pesan 0x%04X (serial %d)\n", parsed.MessageID, parsed.Serial)

	case 0x0200: // Location / Alarm Report
		s.send(jt808.BuildGeneralResponse(parsed, s.nextSerial(), 0x00))

		// Catat alarm kamera (kalau ada) di Session koneksi ini
		if loc, err := jt808.ParseLocation(parsed.MsgBody); err == nil {
			s.noteVideoAlarm(loc)
		}

		// CEK VIDEO ALARM
		if s.hasVideoAlarm {
			fmt.Println(">>> VIDEO ALARM FOUND, READY TO REQUEST")
			// nanti di sini kirim 0x9208
		}
	}

	fmt.Println("=====================================================================================")
}

// =======================================================
// Tampilkan hasil parsing frame
// =======================================================
func printParsedFrame(p jt808.Frame) {
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