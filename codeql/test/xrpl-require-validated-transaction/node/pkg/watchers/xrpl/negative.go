package xrpl

func negativeRejectFalse(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func negativeTrueBranch(parser *Parser, tx TxResponse) {
	if tx.Validated {
		parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
	}
}

func negativeBooleanAlias(parser *Parser, tx TxResponse) {
	validated := tx.Validated
	if !validated {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func negativeLoopContinue(parser *Parser, txs []TxResponse) {
	for _, tx := range txs {
		if !tx.Validated {
			continue
		}
		parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
	}
}

func negativeStreamRejectFalse(parser *Parser, tx StreamTransaction) {
	if !tx.Validated {
		return
	}
	parser.ParseTransactionStream(tx)
}

func negativeUnrelatedMethodNameInOtherDirectory(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}

func negativeUnrelatedSameNameMethods(other *OtherParser, tx TxResponse, stream StreamTransaction) {
	other.ParseTxResponse(&txResponseV2{TxResponse: tx})
	other.ParseTransactionStream(stream)
}
