package zero

func Bytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
