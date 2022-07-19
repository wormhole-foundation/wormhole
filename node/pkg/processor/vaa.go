package processor

import (
	"encoding/hex"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"
)

type VAA struct {
	vaa.VAA
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
	vaaBytes, err := signed.Marshal()
	if err != nil {
		panic(err)
	}

	// Store signed VAA in database.
	p.logger.Info("signed VAA with quorum",
		zap.String("digest", hash),
		zap.Any("vaa", signed),
		zap.String("bytes", hex.EncodeToString(vaaBytes)),
		zap.String("message_id", signed.MessageID()))

	if err := p.db.StoreSignedVAA(signed); err != nil {
		p.logger.Error("failed to store signed VAA", zap.Error(err))
	}

	p.broadcastSignedVAA(signed)
	p.attestationEvents.ReportVAAQuorum(signed)
	p.state.signatures[hash].submitted = true
}
