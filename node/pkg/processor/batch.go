package processor

import (
	"context"
	"encoding/hex"
	"fmt"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	batchesObservedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_batch_observations_total",
			Help: "Total number of Batch messages observed",
		},
		[]string{"emitter_chain"})

	batchesSignedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_batch_observations_signed_total",
			Help: "Total number of Batched observations that were successfully signed",
		},
		[]string{"emitter_chain"})
)

// handleBatchMessage processes a batch message received from a chain.
// An event may be received multiple times and must be handled in an idempotent fashion.
func (p *Processor) handleBatchMessage(ctx context.Context, k *common.BatchMessage) {
	if p.gs == nil {
		p.logger.Warn("dropping observation since we haven't initialized our guardian set yet",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("transaction", k.TransactionID),
		)
		return
	}

	supervisor.Logger(ctx).Info("BatchMessage received.",
		zap.Stringer("emitter_chain", k.EmitterChain),
		zap.Stringer("transaction", k.TransactionID),
	)

	batchesObservedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String(),
	}).Add(1)

	if k.Messages[0].Nonce == 0 {
		// Nonce is zero, don't create batch
		p.logger.Info("aborting processing BatchMessage because nonce==0",
			zap.String("batchID", k.String()),
		)
		return
	}

	batchID := &k.BatchMessageID

	// Check the DB for the BatchMessage
	batchVAAID := db.BatchVAAID{
		EmitterChain:  batchID.EmitterChain,
		TransactionID: batchID.TransactionID,
	}

	if vb, err := p.db.GetSignedBatchVAABytes(batchVAAID); err == nil {

		if _, err := vaa.UnmarshalBatch(vb); err != nil {
			panic("failed to unmarshal VAA from db")
		}

		p.logger.Info("got batchVAA from DB, aborting BatchMessage processing",
			zap.String("messageID", batchID.String()))

		// this VAA has already been seen/signed. No need to continue.
		return

	} else if err != db.ErrVAANotFound {
		p.logger.Error("failed to get batchVAA from db",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transactionId", batchID.TransactionID),
			zap.Error(err),
		)
	}

	// add batchMessage to state.batches, if it does not already exist
	if _, ok := p.state.batches[batchID]; !ok {
		// do not have batch data in state for this message
		p.logger.Info("adding BatchMessage to p.state.batches[batchID]",
			zap.String("batchID", batchID.String()),
		)
		p.state.batches[batchID] = k
	}

	// check for state for this transaction.
	if _, ok := p.state.transactions[batchID]; !ok {
		// no messages have been seen for this batch yet.
		// initialize the state for this transaction.
		p.logger.Info("no messages from this BatchMessage are in processor.state",
			zap.String("batchID", batchID.String()),
		)
		p.state.transactions[batchID] = map[string]*state{}
	}

	// check to see if we have reached quorum on all messages in the batch
	evaluateBatchProgress(p, batchID)
}

// handleBatchPart takes an Observation and determines if it is part of an ongoing batch message.
// An event may be received multiple times and must be handled in an idempotent fashion.
func (p *Processor) handleBatchPart(o Observation) {
	if p.gs == nil {
		p.logger.Warn("dropping batch message since we haven't initialized our guardian set yet")
		return
	}

	messageID := o.MessageID()

	p.logger.Info("BatchPart received", zap.String("MessageID", messageID))

	digest := o.SigningMsg()
	hash := hex.EncodeToString(digest.Bytes())

	if _, ok := p.state.signatures[hash]; !ok {
		p.logger.Info("no state.signatures[hash] for",
			zap.String("hash", hash),
			zap.String("MessageID", messageID),
		)
	}

	sigState := p.state.signatures[hash]

	batchID := &common.BatchMessageID{
		EmitterChain:  o.GetEmitterChain(),
		TransactionID: ethCommon.BytesToHash(sigState.txHash),
	}

	// Step 1) add the Observation to state.transactions[batchID][messageID]

	// check state for this transaction
	if _, ok := p.state.transactions[batchID]; !ok {
		// initalize state.transactions for this batchID
		p.state.transactions[batchID] = map[string]*state{}
	}
	// take the state from signatures, add it to the map by batchID/messageID
	p.state.transactions[batchID][messageID] = sigState

	// Step 2) check state for the BatchMessage this Observation belongs to

	if _, ok := p.state.batches[batchID]; !ok {
		// do not have the batch data in state for this message
		p.logger.Info("BatchMessage not found in state.batches[batchID]",

			zap.String("MessageID", o.MessageID()),
			zap.String("batchID", batchID.String()),
		)
		p.logger.Info("going to push batchID onto batchReqC", zap.String("batchID", batchID.String()))
		p.batchReqC <- batchID
		return
	}

	// Step 3) see if all the Messages in the Batch have reached quorum.
	// Step 4) if complete, create BatchVAA, sign, and broadcast.
	evaluateBatchProgress(p, batchID)
}

// evaluateBatchProgress compares processor.state Observations to see if
// state.transactions contains sufficient VAAs to construct a BatchVAA for the
// supplied BatchMessage.
func evaluateBatchProgress(p *Processor, batchID *common.BatchMessageID) {

	// Check the DB for the BatchMessage
	batchVAAID := db.BatchVAAID{
		EmitterChain:  batchID.EmitterChain,
		TransactionID: batchID.TransactionID,
	}
	if vb, err := p.db.GetSignedBatchVAABytes(batchVAAID); err == nil {
		// just make sure it unmarshals, dont need the contents
		if _, err = vaa.UnmarshalBatch(vb); err != nil {
			panic("failed to unmarshal VAA from db")
		}

		// TODO: maybe allow some grace period like processor/message.go?
		p.logger.Info("ignoring observation since we already have a quorum VAA for it",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transactionID", batchID.TransactionID),
		)
		return
	} else if err != db.ErrVAANotFound {
		p.logger.Error("failed to get BatchVAA from db",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transactionId", batchID.TransactionID),
			zap.Error(err),
		)
	}

	messages := p.state.batches[batchID].Messages

	obsvs := make([]*vaa.Observation, len(messages))

	txState := p.state.transactions[batchID]

	p.logger.Info("going to start checking p.state for BatchMessage.Messages.",
		zap.String("batchID", batchID.String()),
	)

	for msgIndex, msg := range messages {
		p.logger.Info("processing message from batch",
			zap.String("messageID", msg.MessageIDString()),
			zap.String("batchID", batchID.String()),
		)

		msgID := msg.MessageIDString()
		if _, ok := txState[msgID]; !ok {
			// have not seen this message, or don't have it in state any longer.

			p.logger.Info("batch message not found in state.transactions[batchID]",
				zap.String("batchID", batchID.String()),
				zap.String("messageID", msgID),
			)

			// Check the DB for the Message
			vaaID, err := db.VaaIDFromString(msgID)
			if err != nil {
				p.logger.Error("failed parsing MessageID for db.VAAID: %w",
					zap.Error(err))
			}
			if vb, err := p.db.GetSignedVAABytes(*vaaID); err == nil {

				msgVAA, err := vaa.Unmarshal(vb)
				if err != nil {
					panic("failed to unmarshal VAA from db")
				}

				p.logger.Info("got VAA for Batch from DB",
					zap.String("messageID", msgID))

				// create the Observation for the BatchVAA
				ob := &vaa.Observation{
					Index:       uint8(msgIndex),
					Observation: msgVAA,
				}
				// add the Observation to the list
				obsvs[msgIndex] = ob

				continue

			} else if err != db.ErrVAANotFound {
				p.logger.Error("failed to get VAA from db",
					zap.Stringer("emitter_chain", batchID.EmitterChain),
					zap.Stringer("transactionId", batchID.TransactionID),
					zap.Error(err),
				)
			}

			p.logger.Info("did not find message for batch in state or db",
				zap.String("batchID", batchID.String()),
				zap.String("messageID", msgID))

			return
		}

		// state.transactions has the message
		msgState := txState[msgID]

		p.logger.Info("found batch message in state",
			zap.String("messageID", msg.MessageIDString()),
			zap.String("batchID", batchID.String()),
			zap.Int("num_signatures", len(msgState.signatures)),
			zap.Bool("settled", msgState.settled),
			zap.Bool("submitted", msgState.submitted),
			zap.String("ourMsg", string(msgState.ourMsg)),
		)

		var sigs []*vaa.Signature
		for i, a := range msgState.gs.Keys {
			s, ok := msgState.signatures[a]
			if ok {
				var bs [65]byte
				if n := copy(bs[:], s); n != 65 {
					panic(fmt.Sprintf("invalid sig len: %d", n))
				}
				sigs = append(sigs, &vaa.Signature{
					Index:     uint8(i),
					Signature: bs,
				})
			}
		}
		v := &vaa.VAA{
			Version:          vaa.SupportedVAAVersion,
			GuardianSetIndex: msgState.gs.Index,
			Signatures:       sigs,
			Timestamp:        msg.Timestamp,
			Nonce:            msg.Nonce,
			Sequence:         msg.Sequence,
			EmitterChain:     msg.EmitterChain,
			EmitterAddress:   msg.EmitterAddress,
			Payload:          msg.Payload,
			ConsistencyLevel: msg.ConsistencyLevel,
		}

		quorum := CalculateQuorum(len(msgState.gs.Keys))

		if len(sigs) >= quorum {
			// message has reached quorum.
			p.logger.Info("Batch Observation from state has reached quorum",
				zap.String("MessageID", msgID),
				zap.String("BatchID", batchID.String()),
			)

			// create the Observation for the BatchVAA
			ob := &vaa.Observation{
				Index:       uint8(msgIndex),
				Observation: v,
			}
			// add the Observation to the list
			obsvs[msgIndex] = ob

		} else {
			// message has not reached quorum.
			p.logger.Info("Batch Observation from state does not have quorum",
				zap.String("MessageID", msgID),
				zap.String("BatchID", batchID.String()),
			)
		}
	}

	if len(obsvs) == len(messages) {
		// looks like all the messages of the batch reached quorum.
		// now could create a BatchVAA, calculate hash & hashes,
		// sign, and then p.broadcastBatchSignature()

		p.logger.Info("BatchMessage Observations complete.",
			zap.String("BatchID", batchID.String()),
		)

		b := &BatchVAA{
			BatchVAA: vaa.BatchVAA{
				Version:          vaa.BatchVAAVersion,
				GuardianSetIndex: p.gs.Index,
				Signatures:       []*vaa.Signature{},
				EmitterChain:     batchID.EmitterChain,
				TransactionID:    batchID.TransactionID,
				Observations:     obsvs,
			},
		}

		b.Hashes = b.ObsvHashArray()

		p.logger.Info("BatchVAA complete",
			zap.String("BatchID", batchID.String()),
		)

		sig, err := crypto.Sign(b.SigningBatchMsg().Bytes(), p.gk)
		if err != nil {
			panic(err)
		}

		p.logger.Info("just signed BatchVAA")

		p.logger.Info("observed and signed BatchVAA",
			zap.String("batch_id", batchID.String()),
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.String("transaction_id", hex.EncodeToString(batchID.TransactionID.Bytes())),
			zap.String("txhash_b58", base58.Encode(batchID.TransactionID.Bytes())),
			zap.String("signature", hex.EncodeToString(sig)),
		)

		batchesSignedTotal.With(prometheus.Labels{
			"emitter_chain": batchID.EmitterChain.String()}).Add(1)

		p.broadcastBatchSignature(b, sig, batchID.TransactionID.Bytes())
	} else {
		p.logger.Info("BatchMessage Observations are missing from state",
			zap.String("BatchID", batchID.String()),
			zap.Int("BatchMessage.Messages", len(messages)),
			zap.Int("p.state.transactions[batchID]_WITH_QUORUM", len(obsvs)),
			zap.Int("p.state.transactions[batchID]", len(txState)),
		)
	}

}
