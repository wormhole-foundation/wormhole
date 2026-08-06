package xrpl

import "codeql/xrplderivedgeneratedemitter/ethcrypto"

const nttEmitterDomainLen = 3

type Parser struct{}

func (p *Parser) parseNttTransaction(destination Address, tokenInfo nttTokenInfo) MessagePublication {
	sourceNTTManager := addressToEmitter(destination)
	payload := buildNTTPayload(tokenInfo.sourceToken)

	// Calculate emitter address: keccak256("ntt" + source_ntt_manager + source_token)
	emitterAddress := p.calculateEmitterAddress(sourceNTTManager, tokenInfo.sourceToken)

	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}

type nttTokenInfo struct {
	sourceToken Hash
}

func (p *Parser) calculateEmitterAddress(sourceNTTManager, sourceToken Hash) Hash {
	const addrLen = len(sourceNTTManager)
	data := make([]byte, nttEmitterDomainLen+2*addrLen)
	copy(data[:nttEmitterDomainLen], "ntt")
	copy(data[nttEmitterDomainLen:nttEmitterDomainLen+addrLen], sourceNTTManager[:])
	copy(data[nttEmitterDomainLen+addrLen:], sourceToken[:])
	hash := ethcrypto.Keccak256(data)
	var emitter Hash
	copy(emitter[:], hash)
	return emitter
}
