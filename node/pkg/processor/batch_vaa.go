package processor

import (
	"encoding/hex"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type BatchVAA struct {
	vaa.BatchVAA
}

func (v *BatchVAA) HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor) {
	// Deep copy the observation and add signatures
	signed := &vaa.BatchVAA{
		Version:          v.Version,
		GuardianSetIndex: v.GuardianSetIndex,
		Signatures:       sigs,
		EmitterChain:     v.EmitterChain,
		TransactionID:    v.TransactionID,
		Hashes:           v.Hashes,
		Observations:     v.Observations,
	}

	vaaBytes, err := signed.Marshal()
	if err != nil {
		panic(err)
	}

	// Store signed batch VAA in database.
	p.logger.Info("batchVAA with quorum",
		zap.String("digest", hash),
		zap.Any("batch_vaa", signed),
		zap.String("bytes", hex.EncodeToString(vaaBytes)),
		zap.String("batch_id", signed.BatchID()))

	if err := p.db.StoreSignedBatchVAA(signed); err != nil {
		p.logger.Error("failed to store signed batchVAA in db", zap.Error(err))
	}

	p.broadcastSignedBatchVAA(signed)

	// TODO: store batchVAAs in bigtable
	// p.attestationEvents.ReportBatchVAAQuorum(signed)

	p.state.batchSignatures[hash].submitted = true

}
