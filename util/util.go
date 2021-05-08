package util

// Min returns the minimum between two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum between two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ReverseBytes reverses a byte slice
func ReverseBytes(s []byte) []byte {
	res := make([]byte, len(s))
	prevPos, resPos := 0, len(s)
	for pos := range s {
		resPos -= pos - prevPos
		copy(res[resPos:], s[prevPos:pos])
		prevPos = pos
	}
	copy(res[0:], s[prevPos:])
	return res
}
