package cw_wormhole

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
