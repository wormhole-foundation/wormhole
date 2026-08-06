package xrpl

func testFixtureBadXACK(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xackPrefix)
	emitterAddress := addressToEmitter(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}
