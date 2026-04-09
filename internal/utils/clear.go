package utils

func Clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}