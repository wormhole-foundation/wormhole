package solana

func positiveProcessNoValidation(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	s.processTransaction(tx, meta)
}

func positiveLogUseNoValidation(result TransactionResult) {
	_ = result.Meta.LogMessages
}

func positiveExtractionNoValidation(result TransactionResult) {
	result.Transaction.GetTransaction()
}

func positiveResponseExtractionNoValidation(result TransactionResult) {
	result.GetTransaction()
}

func positiveDifferentMeta(s *SolanaWatcher, tx *Transaction, checked *TransactionMeta, used *TransactionMeta) {
	if err := validateTransactionMeta(checked); err != nil {
		return
	}
	s.processTransaction(tx, used)
}

func positiveValidationAfterSink(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	s.processTransaction(tx, meta)
	if err := validateTransactionMeta(meta); err != nil {
		return
	}
}

func positiveReassignedAfterValidation(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta, replacement *TransactionMeta) {
	if err := validateTransactionMeta(meta); err != nil {
		return
	}
	meta = replacement
	s.processTransaction(tx, meta)
}

func positiveIgnoredValidatorError(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	_ = validateTransactionMeta(meta)
	s.processTransaction(tx, meta)
}

func positiveErrorCheckedAfterSink(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	err := validateTransactionMeta(meta)
	s.processTransaction(tx, meta)
	if err != nil {
		return
	}
}

func positivePartialNestedReturn(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta, shouldReturn bool) {
	if err := validateTransactionMeta(meta); err != nil {
		if shouldReturn {
			return
		}
	}
	s.processTransaction(tx, meta)
}

func positiveValidatorErrReassignedToNil(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta) {
	err := validateTransactionMeta(meta)
	err = nil
	if err != nil {
		return
	}
	s.processTransaction(tx, meta)
}

func positiveDirectProofReassigned(s *SolanaWatcher, tx *Transaction, meta *TransactionMeta, replacement *TransactionMeta) {
	if meta != nil {
		meta = replacement
		if meta.Err == nil {
			s.processTransaction(tx, meta)
		}
	}
}
