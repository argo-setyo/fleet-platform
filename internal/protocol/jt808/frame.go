package jt808

import "bytes"

// FrameDelimiter adalah penanda awal dan akhir setiap frame JT808.
const FrameDelimiter = 0x7E

// ExtractFrames memotong data mentah dari koneksi TCP menjadi frame utuh
// (dari 7E sampai 7E).
//
// Data TCP datang sebagai aliran byte, bukan pesan terpisah. Satu bacaan
// bisa berisi beberapa frame sekaligus, atau hanya sebagian frame. Karena
// itu fungsi ini mengembalikan frame yang sudah lengkap, lalu menyisakan
// bagian yang belum lengkap di dalam buffer untuk disambung dengan data
// berikutnya.
func ExtractFrames(buffer *[]byte) [][]byte {
	var frames [][]byte
	data := *buffer

	for {
		start := bytes.IndexByte(data, FrameDelimiter)
		if start == -1 {
			break
		}
		end := bytes.IndexByte(data[start+1:], FrameDelimiter)
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