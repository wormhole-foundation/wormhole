package xrpl

func positiveNoGuard(parser *Parser, tx TxResponse) {
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveStreamNoGuard(parser *Parser, tx StreamTransaction) {
	parser.ParseTransactionStream(tx)
}

func positiveAfterCallGuard(parser *Parser, tx TxResponse) {
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
	if !tx.Validated {
		return
	}
}

func positiveNonTerminatingFalseBranch(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		logInvalid(tx)
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveDifferentValueGuard(parser *Parser, guarded TxResponse, parsed TxResponse) {
	if !guarded.Validated {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: parsed})
}

func positiveReassignedAfterGuard(parser *Parser, tx TxResponse, replacement TxResponse) {
	if !tx.Validated {
		return
	}
	tx = replacement
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveResultOnly(parser *Parser, tx TxResponse) {
	if tx.Result != "tesSUCCESS" {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveValidatedCheckOnlyInHelper(parser *Parser, tx TxResponse) {
	if shouldSkip(tx) {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveValidatedFieldMutatedAfterGuard(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	tx.Validated = false
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveWrappedTxResponseReplacedAfterGuard(parser *Parser, tx TxResponse, replacement TxResponse) {
	if !tx.Validated {
		return
	}
	wrapper := &txResponseV2{TxResponse: tx}
	wrapper.TxResponse = replacement
	parser.ParseTxResponse(wrapper)
}

func positiveWrappedValidatedFieldMutatedAfterGuard(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	wrapper := &txResponseV2{TxResponse: tx}
	wrapper.TxResponse.Validated = false
	parser.ParseTxResponse(wrapper)
}

func positivePromotedValidatedFieldMutatedAfterGuard(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	wrapper := &txResponseV2{TxResponse: tx}
	wrapper.Validated = false
	parser.ParseTxResponse(wrapper)
}

func positiveMutationThenLaterValidatedRead(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	tx.Validated = false
	_ = tx.Validated
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positiveSameLineMutationAfterGuard(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	tx.Validated = false; parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func positivePointerAliasMutationAfterGuard(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	pointer := &tx
	pointer.Validated = false
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}
