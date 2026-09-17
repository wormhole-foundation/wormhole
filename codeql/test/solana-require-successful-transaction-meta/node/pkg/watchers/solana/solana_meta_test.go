package solana

import "testing"

func TestExcludedTransactionMetaUse(t *testing.T) {
	meta := &TransactionMeta{}
	_ = meta.LogMessages
}
