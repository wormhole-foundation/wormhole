package xrpl

func nearMissPositiveBranchWithoutExit(parser *Parser, tx TxResponse) {
	if tx.Validated {
		logInvalid(tx)
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func nearMissBareValidatedRead(parser *Parser, tx TxResponse) {
	_ = tx.Validated
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func nearMissAddressWrapper(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	wrapper := &txResponseV2{TxResponse: tx}
	parser.ParseTxResponse(wrapper)
}

func nearMissStalePointerAlias(parser *Parser, tx TxResponse, replacement TxResponse) {
	if !tx.Validated {
		return
	}
	pointer := &tx
	pointer = &replacement
	pointer.Validated = false
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}
