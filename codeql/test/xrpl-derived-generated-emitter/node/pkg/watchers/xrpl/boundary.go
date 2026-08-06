package xrpl

func boundaryPayloadOnlyXTCF() []byte {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	return payload
}

func boundaryNonXRPLChain(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDSolana,
		EmitterAddress: addressToEmitter(account),
		Payload:        payload,
	}
}

func boundaryRoutingOnly(publication MessagePublication) MessagePublication {
	return publication
}
