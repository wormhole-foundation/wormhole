package gethtypes

type Hash [32]byte

type Receipt struct {
	TxHash    Hash
	BlockHash Hash
	Status    uint64
}

const ReceiptStatusSuccessful = 1

func BytesToHash(b []byte) Hash { return Hash{} }
