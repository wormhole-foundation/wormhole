package solana

func negativeValidatorGuard(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	if err := validateTransactionMeta(meta); err != nil {
		return
	}
	s.processTransaction(tx, meta)
}

func negativeDirectEquivalentGuard(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	if meta != nil && meta.Err == nil {
		s.processTransaction(tx, meta)
	}
}

func negativeValidatedLogAndExtraction(result TransactionResult) {
	if metadataErr := validateTransactionMeta(result.Meta); metadataErr != nil {
		return
	}
	_ = result.Meta.LogMessages
	result.Transaction.GetTransaction()
	result.GetTransaction()
}

func negativeWrapperAccountException(account string) {
	processAccountSubscriptionData(account)
}
