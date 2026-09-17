package gethtypes

const ReceiptStatusSuccessful uint64 = 1

type Hash [32]byte

type Log struct {
	Removed bool
}

type Receipt struct {
	Status    uint64
	Logs      []*Log
	TxHash    Hash
	BlockHash Hash
}
