package ethcrypto

func Keccak256(data []byte) []byte {
	out := make([]byte, 32)
	copy(out, data)
	return out
}
