package xrpl

func negativeXTCFGeneratedEmitter(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	emitterAddress := calculateGeneratedEmitterAddress(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

func negativeXACKGeneratedEmitter(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xackPrefix)
	emitterAddress := calculateGeneratedEmitterAddress(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

func negativeCoreRawEmitterException(account Address, memo []byte) MessagePublication {
	emitterAddress := addressToEmitter(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        memo,
	}
}
