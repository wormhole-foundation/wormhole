package crypto

type Hash struct{}

func (h Hash) Hex() string { return "hash" }

func Keccak256Hash(data []byte) Hash { return Hash{} }
