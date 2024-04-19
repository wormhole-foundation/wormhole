package processor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
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

	// Store signed VAA in database.
	p.logger.Info("signed VAA with quorum",
		zap.String("message_id", signed.MessageID()),
		zap.String("digest", hash),
	)

	if err := p.storeSignedVAA(signed); err != nil {
		p.logger.Error("failed to store signed VAA",
			zap.String("message_id", signed.MessageID()),
			zap.String("digest", hash),
			zap.Error(err),
		)
	}

	p.broadcastSignedVAA(signed)
	p.state.signatures[hash].submitted = true
}

func (v *VAA) IsReliable() bool {
	return !v.Unreliable
}

func (v *VAA) IsReobservation() bool {
	return v.Reobservation
}
