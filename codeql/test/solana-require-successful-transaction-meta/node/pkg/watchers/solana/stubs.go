package solana

type TransactionMeta struct {
	Err         error
	LogMessages []string
}

type TransactionResult struct {
	Meta        *TransactionMeta
	Transaction EncodedTransaction
}

type EncodedTransaction struct{}

type Transaction struct{}

type SolanaWatcher struct{}

func (EncodedTransaction) GetTransaction() (*Transaction, error) { return nil, nil }

func (TransactionResult) GetTransaction() (*Transaction, error) { return nil, nil }

func validateTransactionMeta(meta *TransactionMeta) error { return nil }

func (s *SolanaWatcher) processTransaction(tx *Transaction, meta *TransactionMeta) uint32 {
	if metadataErr := validateTransactionMeta(meta); metadataErr != nil {
		return 0
	}
	_ = meta.LogMessages
	return 1
}

func processAccountSubscriptionData(account string) {}
