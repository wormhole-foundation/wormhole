package xrpl

func positiveXTCFRawEmitter(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	emitterAddress := addressToEmitter(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

func positiveXACKRawEmitterDirect(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xackPrefix)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: addressToEmitter(account),
		Payload:        payload,
	}
}

func positiveXTCFUndomainSeparatedWrapper(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xtcfPrefix)
	emitterAddress := rawAccountEmitterWrapper(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

func rawAccountEmitterWrapper(account Address) Hash {
	return addressToEmitter(account)
}

func parseNttTransaction(destination Address, sourceToken Hash) []MessagePublication {
	sourceNTTManager := addressToEmitter(destination)
	payload := buildNTTPayload(sourceToken)
	rawManagerEmitter := addressToEmitter(destination)
	noPrefixEmitter := calculateEmitterAddressWithoutPrefix(sourceNTTManager, sourceToken)
	wrongHashEmitter := calculateEmitterAddressWrongHash(sourceNTTManager, sourceToken)

	return []MessagePublication{
		{
			EmitterChain:   vaa.ChainIDXRPL,
			EmitterAddress: rawManagerEmitter,
			Payload:        payload,
		},
		{
			EmitterChain:   vaa.ChainIDXRPL,
			EmitterAddress: noPrefixEmitter,
			Payload:        payload,
		},
		{
			EmitterChain:   vaa.ChainIDXRPL,
			EmitterAddress: wrongHashEmitter,
			Payload:        payload,
		},
		{
			EmitterChain:   vaa.ChainIDXRPL,
			EmitterAddress: calculateEmitterAddress(sourceNTTManager, sourceToken),
			Payload:        payload,
		},
	}
}

func calculateEmitterAddressWithoutPrefix(sourceNTTManager Hash, sourceToken Hash) Hash {
	buf := make([]byte, 64)
	copy(buf[:32], sourceNTTManager[:])
	copy(buf[32:64], sourceToken[:])
	return Keccak256(buf)
}

func calculateEmitterAddressWrongHash(sourceNTTManager Hash, sourceToken Hash) Hash {
	buf := make([]byte, 67)
	copy(buf[:3], "ntt")
	copy(buf[3:35], sourceNTTManager[:])
	copy(buf[35:67], sourceToken[:])
	return sha256Hash(buf)
}
