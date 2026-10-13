package algorand

import "errors"

const publishMessage = "publishMessage"

var errBadLength = errors.New("bad length")

type SignedTxnWithAD struct {
	Txn       Transaction
	EvalDelta EvalDelta
}

type Transaction struct {
	ApplicationArgs [][]byte
	ApplicationID   uint64
}

type EvalDelta struct {
	Logs []string
}

type MessagePublication struct {
	Nonce    uint32
	Sequence uint64
}

func isPublishMessage(at SignedTxnWithAD) bool {
	return len(at.Txn.ApplicationArgs) > 2 && string(at.Txn.ApplicationArgs[0]) == publishMessage
}

func publish(mp MessagePublication) {
	_ = mp
}
