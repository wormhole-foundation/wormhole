package xrpl

func nearMissXACKHelperNamedGeneratedButNoOverlay(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xackPrefix)
	emitterAddress := calculateGeneratedEmitterAddressNoOverlay(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

func calculateGeneratedEmitterAddressNoOverlay(account Address) Hash {
	return addressToEmitter(account)
}

func nearMissUnreachableBadXTCF(account Address) *MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	if false {
		emitterAddress := addressToEmitter(account)
		return &MessagePublication{
			EmitterChain:   vaa.ChainIDXRPL,
			EmitterAddress: emitterAddress,
			Payload:        payload,
		}
	}
	return nil
}
