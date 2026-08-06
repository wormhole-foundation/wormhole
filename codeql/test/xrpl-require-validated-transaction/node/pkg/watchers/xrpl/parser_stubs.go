package xrpl

type Parser struct{}

type OtherParser struct{}

type TxResponse struct {
	Validated bool
	Result    string
}

type txResponseV2 struct {
	TxResponse
}

type StreamTransaction struct {
	Validated bool
	Hash      string
}

func (p *Parser) ParseTransactionStream(tx StreamTransaction) {}

func (p *Parser) ParseTxResponse(tx *txResponseV2) {}

func (p *OtherParser) ParseTransactionStream(tx StreamTransaction) {}

func (p *OtherParser) ParseTxResponse(tx *txResponseV2) {}

func logInvalid(any) {}

func shouldSkip(tx TxResponse) bool {
	return !tx.Validated
}
