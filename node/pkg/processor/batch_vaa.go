package processor

import (
	"encoding/hex"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type Batch struct {
	vaa.Batch
}
type BatchID string

func (v *Batch) HandleQuorum(sigs []*vaa.Signature, hash string, p *Processor) {
	// Deep copy the observation and add signatures
	signed := &vaa.Batch{
		BatchVAA: vaa.BatchVAA{
			Version:          v.Version,
			GuardianSetIndex: v.GuardianSetIndex,
			Signatures:       sigs,
			Hashes:           v.Hashes,
			Observations:     v.Observations,
		},
		BatchID: vaa.BatchID{
			EmitterChain:  v.BatchID.EmitterChain,
			TransactionID: v.BatchID.TransactionID,
			Nonce:         v.BatchID.Nonce,
		},
	}

	vaaBytes, err := signed.BatchVAA.Marshal()
	if err != nil {
		panic(err)
	}

	// Store signed batch VAA in database.
	p.logger.Info("batch with quorum",
		zap.String("digest", hash),
		zap.String("bytes", hex.EncodeToString(vaaBytes)),
		zap.Stringer("emitter_chain", signed.BatchID.EmitterChain),
		zap.Stringer("transaction_id", signed.BatchID.TransactionID),
		zap.Uint32("nonce", uint32(signed.BatchID.Nonce)),
		zap.Stringer("batch_id", &signed.BatchID))

	if err := p.db.StoreSignedBatch(signed); err != nil {
		p.logger.Error("failed to store signed batchVAA in db",
			zap.Stringer("emitter_chain", signed.BatchID.EmitterChain),
			zap.Stringer("transaction_id", signed.BatchID.TransactionID),
			zap.Uint32("nonce", uint32(signed.BatchID.Nonce)),
			zap.Stringer("batch_id", &signed.BatchID),
			zap.Error(err))
	}

	p.broadcastSignedBatchVAA(signed)

	// TODO: store batchVAAs in bigtable
	// p.attestationEvents.ReportBatchVAAQuorum(signed)

	p.state.batchSignatures[hash].submitted = true
}
