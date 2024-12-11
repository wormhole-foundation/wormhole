package cw_wormhole

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GuardianSetQueryResponse struct {
	Data GuardianSetInfoResponse `json:"data"`
}

type VerifyVAAQueryResponse struct {
	Data ParsedVAA `json:"data"`
}

type GetStateQueryResponse struct {
	Data GetStateResponse `json:"data"`
}

type QueryAddressHexQueryResponse struct {
	Data GetAddressHexResponse `json:"data"`
}

// Custom response type to handle string numbers
type TxResponse struct {
	Code uint32              `json:"code"`
	Logs sdk.ABCIMessageLogs `json:"logs"`
}
