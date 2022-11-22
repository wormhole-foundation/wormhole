package processor

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
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

// handleBatchMessage processes TransactionData received from a chain watcher.
// An event may be received multiple times and must be handled in an idempotent fashion.
func (p *Processor) handleBatchMessage(ctx context.Context, k *common.TransactionData) {
	if !p.batchVAAEnabled {
		// respect the feature flag
		return
	}

	if p.gs == nil {
		p.logger.Warn("dropping observation since we haven't initialized our guardian set yet",
			zap.Stringer("emitter_chain", &k.EmitterChain),
			zap.Stringer("transaction_id", k.TransactionID))
		return
	}

	supervisor.Logger(ctx).Debug("TransactonData received.",
		zap.Stringer("emitter_chain", &k.EmitterChain),
		zap.Stringer("transaction_id", k.TransactionID))

	// loop through all the messages from the transaction and collect messages by BatchID
	messageGroups := map[vaa.BatchID][]*common.MessagePublication{}

	for _, msg := range k.Messages {

		// exclude messages with nonce: 0 from batches.
		if msg.Nonce == 0 {
			continue
		}

		batchID := vaa.BatchID{
			EmitterChain:  k.EmitterChain,
			TransactionID: k.TransactionID,
			Nonce:         vaa.Nonce(msg.Nonce),
		}
		messageGroups[batchID] = append(messageGroups[batchID], msg)
	}

	// loop through the groups and see if they fit the criteria to be a BatchVAA
	for batchID, group := range messageGroups {

		if len(group) > vaa.MaxBatchObservations {
			// the maximum number of allowed messages in a batch is 255, as to not overflow the uint8
			// that holds the number messages in the binary representation of the BatchVAA.

			// ignore this batch rather than trimming or modifying it to conform.
			p.logger.Warn(
				fmt.Sprintf("encountered a batch of more than %d messages that "+
					"otherwise meet the criteria to be batched. "+
					"this group will be ignored, a BatchVAA will not be produced.",
					vaa.MaxBatchObservations),
				zap.Stringer("emitter_chain", &batchID.EmitterChain),
				zap.Stringer("transaction_id", batchID.TransactionID),
				zap.Uint32("nonce", uint32(batchID.Nonce)),
				zap.Stringer("batch_id", &batchID))
			continue
		}

		if len(group) >= 2 {
			// Should produce a Batch for this group of messages.
			// Check if we've seen this BatchID before
			// Add the Batch to p.state and consider progress toward "finality"
			// (all the individual parts independently reaching quorum).

			// add the Batch to Processor state, start tracking it's progress
			p.considerBatchForTracking(&batchID, group)
		}
	}
}

// considerBatchForTracking handles Batches in an idempotent fashion, adding them to Processor state
// if they have not already been seen.
func (p *Processor) considerBatchForTracking(batchID *vaa.BatchID, msgs []*common.MessagePublication) {
	// first, see if we've already reached quorm for this BatchID
	if _, err := p.db.GetSignedBatchBytes(*batchID); err == nil {
		// this VAA has already been seen/signed. No need to continue.
		p.logger.Debug("got batchVAA from DB, aborting BatchMessage processing",
			zap.Stringer("batch_id", batchID))
		return
	} else if err != db.ErrVAANotFound {
		// an error occured trying to access the DB.
		p.logger.Error("failed to query db for batchVAA",
			zap.Stringer("batch_id", batchID),
			zap.Error(err))
	}

	// Check to see if this BatchID is already being tracked in Processor state.

	// Add Batch to state.batches, if it does not already exist.
	if _, ok := p.state.batches[*batchID]; !ok {
		// we do not have Batch data in state for this message, it's new to us.
		batchesObservedTotal.With(prometheus.Labels{
			"emitter_chain": batchID.EmitterChain.String(),
		}).Add(1)

		p.logger.Debug("adding BatchMessage to p.state.batches[batchID]",
			zap.Stringer("batch_id", batchID))
		// add the messages to batches as the canonical record of the messages
		// this Batch contains. This will be referenced as the individual messages
		// reach quorum, to see if we have all the messages we expect, and can sign
		// the entire group and broadcast to peers.
		p.state.batches[*batchID] = msgs
	}

	// Processor state setup for this Batch is complete.
	// Now as v1 VAAs reach quorum they will be checked to see if they are part
	// of a Batch that's being tracked, and if so, if the entire batch has been observed.

	// Now check to see if we have reached quorum on all messages in this Batch.
	// If the messages had finalty: 0, it's possible we could already have all
	// the VAAs we need to sign the Batch and broadcast it.
	p.evaluateBatchProgress(batchID)
}

// handleBatchPart takes an Observation and determines if it is part of a pending Batch.
// An event may be received multiple times and must be handled in an idempotent fashion.
// This is called every time we see a v1 VAA with quorum (from gossip or accumulating signatures).
func (p *Processor) handleBatchPart(o Observation) {
	if !p.batchVAAEnabled {
		// respect the feature flag
		return
	}

	if p.gs == nil {
		p.logger.Warn("dropping batch message since we haven't initialized our guardian set yet")
		return
	}

	// Fetch the various identifiers needed to consider a VAA for inclusion in a Batch:
	// MessageID, TransactionID, Nonce

	messageID := o.MessageID()

	// fetch the VAA from the db
	VAAID, err := db.VaaIDFromString(messageID)
	if err != nil {
		panic(fmt.Errorf("failed parsing MessageID for VaaID. %w", err))
	}
	v, err := p.getSignedVAA(*VAAID)
	if err != nil {
		p.logger.Debug("did not find VAA in db",
			zap.String("message_id", messageID),
			zap.Error(err))
		return
	}

	// Check if this message is deliberately excluded from batches
	nonce := vaa.Nonce(v.Nonce)
	if nonce == 0 {
		// messages with Nonce: 0 are not included in Batches
		return
	}

	// Get the txHash that was observed with the message from state
	digest := o.SigningMsg()
	hash := hex.EncodeToString(digest.Bytes())

	// Ensure Processor state exists for this message
	if _, ok := p.state.signatures[hash]; !ok {
		p.logger.Debug("no Processor state exists for",
			zap.String("hash", hash),
			zap.String("message_id", messageID))
		// We don't have information about the observation of the message,
		// nothing to do.
		return
	}
	// Get the state created when the message was observed
	msgState := p.state.signatures[hash]

	// ensure the txHash exists
	if msgState.txHash == nil {
		p.logger.Debug("no txHash in Processor state for message",
			zap.String("message_id", messageID),
			zap.String("hash", hash))
		// need txHash to consider a message as part of a Batch.
		return
	}

	// Create the normalized TransactionID from the observation txHash.
	tx, err := vaa.BytesToTransactionID(msgState.txHash)
	if err != nil {
		p.logger.Error("failed processing txHash of message",
			zap.String("message_id", messageID),
			zap.String("hash", hash),
			zap.Error(err))
		return
	}

	// Create the BatchID
	batchID := &vaa.BatchID{
		EmitterChain:  v.EmitterChain,
		TransactionID: tx,
		Nonce:         nonce,
	}

	// Accumulate messages in state in order to identify transactions with multiple messages

	// Step 1) add the message to state.batchMessages[batchID][messageID]

	// check state for this transaction
	if _, ok := p.state.batchMessages[*batchID]; !ok {
		// initalize state.batchMessages for this batchID
		p.state.batchMessages[*batchID] = map[string]*state{}
	}
	// take the state from the observation, add it to the map by batchID/messageID
	p.state.batchMessages[*batchID][messageID] = msgState

	// if this is the first message we've seen for the Batch, bail out, no need
	// to check if this message completes the Batch.
	if len(p.state.batchMessages[*batchID]) == 1 {
		return
	}
	// We have more than 1 message for this Batch, so check on the progress toward completion.

	// Step 2) check state for the Batch this Observation belongs to

	// Check if we've fetched the TransactionData for this Batch. If we don't have it, request it.
	if _, ok := p.state.batches[*batchID]; !ok {
		// we do not have the TransactionData that produced this message stored in state yet.
		p.logger.Debug("going to request TransactionData for batch",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.Stringer("batch_id", batchID),
			zap.String("message_id", messageID))

		// Make a request for the watcher of the chain to get data about the transaction
		// that emitted this message.
		queryRequest := &common.TransactionQuery{
			EmitterChain:  batchID.EmitterChain,
			TransactionID: batchID.TransactionID,
		}

		// publish the request
		p.batchReqC <- queryRequest
		return
	}

	// Step 3) see if all the Messages in the Batch have reached quorum.
	p.evaluateBatchProgress(batchID)
}

// evaluateBatchProgress compares the known messages in the Batch against the db
// of signed V1 VAAs to see if we have everything needed to contruct the BatchVAA
// that can be signed and broadcast to gossip.
func (p *Processor) evaluateBatchProgress(batchID *vaa.BatchID) {

	// Check the DB to see if this has already been seen
	if _, err := p.db.GetSignedBatchBytes(*batchID); err == nil {
		// this VAA has already been seen/signed. No need to continue.
		p.logger.Debug("ignoring observation since we already have a quorum VAA for it",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID))
		return
	} else if err != db.ErrVAANotFound {
		// an error occured trying to access the DB
		p.logger.Error("failed to query db for BatchVAA",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.Stringer("batch_id", batchID),
			zap.Error(err))
	}

	// this should never get called for a batchID not in state,
	// check and report just in case.
	if _, ok := p.state.batches[*batchID]; !ok {
		p.logger.Error("tried to evaluate batch progress for a batch not in state",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.Stringer("batch_id", batchID))
		return

	}

	// source of truth for the expected messages in the batch
	messages := p.state.batches[*batchID]

	// Create a list of Observations to be filled with signed v1 VAAs.
	// This will be the list of Observations of the BatchVAA.
	obsvs := make([]*vaa.Observation, len(messages))

	// loop through the expected set of messages in the batch, find the signed
	// version, and add it to the "obsvs" list.
	for msgIndex, msg := range messages {
		p.logger.Debug("processing message from batch",
			zap.String("message_id", msg.MessageIDString()),
			zap.String("batch_id", batchID.String()))

		msgID := msg.MessageIDString()

		// check the DB for the signed message
		vaaID := db.VaaIDFromVAA(&vaa.VAA{
			EmitterChain:   msg.EmitterChain,
			EmitterAddress: msg.EmitterAddress,
			Sequence:       msg.Sequence})
		if vb, err := p.db.GetSignedVAABytes(*vaaID); err == nil {
			p.logger.Debug("got VAA for Batch from DB",
				zap.String("message_id", msgID),
				zap.Stringer("batch_id", batchID))

			msgVAA, err := vaa.Unmarshal(vb)
			if err != nil {
				panic("failed to unmarshal VAA from db")
			}

			// create the Observation for the BatchVAA
			ob := &vaa.Observation{
				Index:       uint8(msgIndex),
				Observation: msgVAA,
			}
			// add the Observation to the list
			obsvs[msgIndex] = ob

			// Signed VAA found and added to obsvs, proceed to next message
			continue
		}

		p.logger.Debug("did not find VAA in db for batch",
			zap.Stringer("emitter_chain", msg.EmitterChain),
			zap.Stringer("emitter_address", msg.EmitterAddress),
			zap.Uint64("sequence", msg.Sequence),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.Stringer("batch_id", batchID))

		// A signed VAA was not found for this message, Batch is not complete.
		return
	}

	if len(obsvs) == len(messages) {
		// All the messages in the Batch achieved quorum.

		// Now create a BatchVAA, generate hashes of each Observation,
		// create a hash of the Obervation hashes, sign it,
		// begin tracking the signatures of this BatchVAA itself,
		// and broadcast it to peers over the gossip network.

		p.logger.Info("Batch Observation complete.",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.String("batch_id", batchID.String()))

		b := &vaa.Batch{
			BatchVAA: vaa.BatchVAA{
				Version:          vaa.BatchVAAVersion,
				GuardianSetIndex: p.gs.Index,
				Signatures:       []*vaa.Signature{},
				Observations:     obsvs,
			},
			BatchID: *batchID,
		}

		b.Hashes = b.ObsvHashArray()

		sig, err := crypto.Sign(b.SigningMsg().Bytes(), p.gk)
		if err != nil {
			panic(err)
		}

		p.logger.Debug("observed and signed BatchVAA",
			zap.Stringer("emitter_chain", batchID.EmitterChain),
			zap.Stringer("transaction_id", batchID.TransactionID),
			zap.Uint32("nonce", uint32(batchID.Nonce)),
			zap.Stringer("batch_id", batchID),
			zap.String("signature", hex.EncodeToString(sig)))

		batchesSignedTotal.With(prometheus.Labels{
			"emitter_chain": batchID.EmitterChain.String()}).Add(1)

		p.broadcastBatchSignature(b, sig)
	}

}
