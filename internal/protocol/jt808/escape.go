package jt808

// Unescape mengembalikan byte yang disamarkan di dalam frame:
// 7D 02 menjadi 7E, dan 7D 01 menjadi 7D.
func Unescape(src []byte) []byte {
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

// Escape menyamarkan byte 7E dan 7D di dalam data supaya tidak
// tertukar dengan penanda awal/akhir frame.
func Escape(src []byte) []byte {
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