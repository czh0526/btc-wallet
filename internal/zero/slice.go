package zero

// Bytes
func Bytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
