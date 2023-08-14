package types

type IbcTranslatorQueryMsg struct {
	IbcChannel QueryIbcChannel `json:"ibc_channel"`
}

type QueryIbcChannel struct {
	ChainID uint16 `json:"chain_id"`
}

type IbcTranslatorQueryRsp struct {
	Channel string `json:"channel,omitempty"`
}
