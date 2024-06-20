package processor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type VAA struct {
	vaa.VAA
	Unreliable    bool
	Reobservation bool
}

func (v *VAA) HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor) {
	// Deep copy the observation and add signatures
	signed := &vaa.VAA{
		Version:          v.Version,
		GuardianSetIndex: v.GuardianSetIndex,
		Signatures:       sigs,
		Timestamp:        v.Timestamp,
		Nonce:            v.Nonce,
		Sequence:         v.Sequence,
		EmitterChain:     v.EmitterChain,
		EmitterAddress:   v.EmitterAddress,
		Payload:          v.Payload,
		ConsistencyLevel: v.ConsistencyLevel,
	}

	p.postSignedVAA(signed, hash)
}

func (v *VAA) IsReliable() bool {
	return !v.Unreliable
}

func (v *VAA) IsReobservation() bool {
	return v.Reobservation
}
