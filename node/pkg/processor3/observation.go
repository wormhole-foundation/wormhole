//nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor3

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_received_total3",
			Help: "Total number of raw VAA observations received from gossip",
		})
	observationsReceivedByGuardianAddressTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_signed_by_guardian_total3",
			Help: "Total number of signed and verified VAA observations grouped by guardian address",
		}, []string{"addr"})
	observationsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_verification_failures_total3",
			Help: "Total number of observations verification failure, grouped by failure reason",
		}, []string{"cause"})
)

func (p *Processor) dispatchObservation(ctx context.Context, logger *zap.Logger, o *gossipv1.SignedObservation) {
	inLeaderSet, processingBucket := calculateBucket(o.Hash, p.gst.MyKeyIndex(), len(p.gst.Get().Keys))
	logger.Debug("dispatchObservation", zap.String("msgId", o.MessageId), zap.Bool("leaderset", inLeaderSet))
	if !inLeaderSet {
		// we're not going to deal with this one
		return
	}

	hash := hex.EncodeToString(o.Hash)

	s, ok := p.state[hash]

	if !ok {
		s = p.NewStateFromForeignObserved()
		p.state[hash] = s
	}

	if s != nil && s.submitted {
		// already reached quorum; ignoring additional signatures for it.
		logger.Debug("dispatchObservation: already reached quorum", zap.String("msgId", o.MessageId), zap.Bool("leaderset", inLeaderSet))
		// TODO this performance optimization is disabled to allow apples-to-apples comparison with other parallelization approaches
		//return
	}

	p.observationChannels[processingBucket] <- observationProcessingJob{o: o, state: s}
}

// handleObservation processes a remote observation, verifies it, checks whether the VAA has met quorum,
// and assembles and submits a valid VAA if possible.
func (p *ConcurrentProcessor) handleObservation(ctx context.Context, logger *zap.Logger, job observationProcessingJob) {
	// SECURITY: at this point, observations received from the p2p network are fully untrusted (all fields!)
	//
	// Note that observations are never tied to the (verified) p2p identity key - the p2p network
	// identity is completely decoupled from the guardian identity, p2p is just transport.

	obs := job.o
	s := job.state
	hash := hex.EncodeToString(obs.Hash)

	logger.Debug("handleObservation", zap.String("messageId", obs.MessageId))

	if s.submitted {
		// already submitted; ignoring additional signatures
		logger.Debug("already submitted, doing nothing", zap.String("messageId", obs.MessageId))
		// TODO this performance optimization is disabled to allow apples-to-apples comparison with other parallelization approaches
		// return
	}

	// check if we already have a VAA for this observation
	vaaId, err := db.VaaIDFromString(obs.MessageId)
	if err != nil {
		logger.Info("invalid messageID",
			zap.String("digest", hash),
			zap.String("messageId", obs.MessageId),
		)
		return
	}
	if p.haveSignedVAA(*vaaId) {
		// already have a VAA for it; ignoring additional signatures
		logger.Debug("already have VAA, doing nothing", zap.String("messageId", obs.MessageId))
		return
	}

	their_addr := ethcommon.BytesToAddress(obs.Addr)

	// check if we have a valid signature from this guardian already
	if _, ok := s.signatures[their_addr]; ok {
		// already have a valid signature from this guardian
		logger.Debug("already have signature from this guardian, doing nothing", zap.String("messageId", obs.MessageId))
		return
	}

	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("received observation",
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(obs.Signature)),
			zap.String("addr", hex.EncodeToString(obs.Addr)),
			zap.String("txhash", hex.EncodeToString(obs.TxHash)),
			zap.String("txhash_b58", base58.Encode(obs.TxHash)),
			zap.String("message_id", obs.MessageId),
		)
	}

	observationsReceivedTotal.Inc()

	// Verify the Guardian's signature. This verifies that m.Signature matches m.Hash and recovers
	// the public key that was used to sign the payload.
	pk, err := crypto.Ecrecover(obs.Hash, obs.Signature)
	if err != nil {
		logger.Warn("failed to verify signature on observation",
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(obs.Signature)),
			zap.String("addr", hex.EncodeToString(obs.Addr)),
			zap.String("messageId", obs.MessageId),
			zap.Error(err))
		observationsFailedTotal.WithLabelValues("invalid_signature").Inc()
		return
	}

	// Verify that m.Addr matches the public key that signed m.Hash.
	signer_pk := ethcommon.BytesToAddress(crypto.Keccak256(pk[1:])[12:])

	if their_addr != signer_pk {
		logger.Info("invalid observation - address does not match pubkey",
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(obs.Signature)),
			zap.String("addr", hex.EncodeToString(obs.Addr)),
			zap.String("messageId", obs.MessageId),
			zap.String("pk", signer_pk.Hex()))
		observationsFailedTotal.WithLabelValues("pubkey_mismatch").Inc()
		return
	}

	// Determine which guardian set to use. The following cases are possible:
	//
	//  - We have already seen the message and generated ourObservation. In this case, use the guardian set valid at the time,
	//    even if the guardian set was updated. Old guardian sets remain valid for longer than aggregation state,
	//    and the guardians in the old set stay online and observe and sign messages for the transition period.
	//
	//  - We have not yet seen the message. In this case, we assume the latest guardian set because that's what
	//    we will store once we do see the message.
	//
	// This ensures that during a guardian set update, a node which observed a given message with either the old
	// or the new guardian set can achieve consensus, since both the old and the new set would achieve consensus,
	// assuming that 2/3+ of the old and the new guardian set have seen the message and will periodically attempt
	// to retransmit their observations such that nodes who initially dropped the signature will get a 2nd chance.
	//
	// During an update, vaaState.signatures can contain signatures from *both* guardian sets.
	//
	var gs *common.GuardianSet
	if s != nil && s.gs != nil {
		gs = s.gs
	} else {
		gs = p.gst.Get()
	}

	// We haven't yet observed the trusted guardian set on Ethereum, and therefore, it's impossible to verify it.
	// May as well not have received it/been offline - drop it and wait for the guardian set.
	if gs == nil {
		logger.Warn("dropping observations since we haven't initialized our guardian set yet",
			zap.String("digest", hash),
			zap.String("their_addr", their_addr.Hex()),
			zap.String("messageId", obs.MessageId),
		)
		observationsFailedTotal.WithLabelValues("uninitialized_guardian_set").Inc()
		return
	}

	// Verify that m.Addr is included in the guardian set. If it's not, drop the message. In case it's us
	// who have the outdated guardian set, we'll just wait for the message to be retransmitted eventually.
	_, ok := gs.KeyIndex(their_addr)
	if !ok {
		logger.Debug("received observation by unknown guardian - is our guardian set outdated?",
			zap.String("digest", hash),
			zap.String("their_addr", their_addr.Hex()),
			zap.Uint32("index", gs.Index),
			zap.String("messageId", obs.MessageId),
			//zap.Any("keys", gs.KeysAsHexStrings()),
		)
		observationsFailedTotal.WithLabelValues("unknown_guardian").Inc()
		return
	}

	// Hooray! Now, we have verified all fields on SignedObservation and know that it includes
	// a valid signature by an active guardian. We still don't fully trust them, as they may be
	// byzantine, but now we know who we're dealing with.

	// We can now count events by guardian without worry about cardinality explosions:
	observationsReceivedByGuardianAddressTotal.WithLabelValues(their_addr.Hex()).Inc()

	s.signatures[their_addr] = obs.Signature

	quorum := vaa.CalculateQuorum(len(gs.Keys))

	if len(s.signatures) < quorum {
		// no quorum yet, we're done here
		logger.Debug("quorum not yet met",
			zap.String("digest", hash),
			zap.String("messageId", obs.MessageId),
		)
		return
	}

	// We have reached quorum!

	if s.msg == nil {
		// We have not made this observation ourselves (yet) and therefore cannot create the VAA.
		// But hopefully we'll make that observation at some point and then we'll still have quorum.
		logger.Debug("we have not yet seen this observation and therefore can't create the VAA",
			zap.String("digest", hash),
			zap.String("messageId", obs.MessageId),
		)
		return
	}

	// Aggregate all valid signatures into a list of vaa.Signature and construct signed VAA.
	agg := make([]bool, len(gs.Keys))
	var sigs []*vaa.Signature
	for i, a := range gs.Keys {
		sig, ok := s.signatures[a]

		if ok {
			var bs [65]byte
			if n := copy(bs[:], sig); n != 65 {
				panic(fmt.Sprintf("invalid sig len: %d", n))
			}

			sigs = append(sigs, &vaa.Signature{
				Index:     uint8(i),
				Signature: bs,
			})
		}

		agg[i] = ok
	}

	logger.Debug("aggregation state for observation", // 1.3M out of 3M info messages / hour / guardian
		zap.String("digest", hash),
		//zap.Any("set", gs.KeysAsHexStrings()),
		zap.Uint32("index", gs.Index),
		zap.Bools("aggregation", agg),
		zap.Int("required_sigs", quorum),
		zap.Int("have_sigs", len(sigs)),
		zap.Bool("quorum", len(sigs) >= quorum),
		zap.String("messageId", obs.MessageId),
	)

	if len(sigs) < quorum {
		// no quorum yet, we're done here
		logger.Debug("quorum not yet met after aggregating signatures -- maybe there is a guardian set change?",
			zap.String("digest", hash),
			zap.String("messageId", obs.MessageId),
		)
		return
	}

	signedVaa := s.msg.CreateVAA(s.gs.Index)
	signedVaa.Signatures = sigs

	// Store signed VAA in database.
	logger.Info("signed VAA with quorum",
		zap.String("digest", hash),
		zap.String("message_id", signedVaa.MessageID()))

	if err := p.storeSignedVAA(signedVaa); err != nil {
		logger.Error("failed to store signed VAA", zap.Error(err))
	}

	p.reachedQuorumC <- hash
	p.broadcastSignedVAA(signedVaa)
	p.attestationEvents.ReportVAAQuorum(signedVaa)
	s.submitted = true
}

func (p *ConcurrentProcessor) dispatchInboundSignedVAAWithQuorum(ctx context.Context, logger *zap.Logger, m *gossipv1.SignedVAAWithQuorum) {
	v, err := vaa.Unmarshal(m.Vaa)
	if err != nil {
		logger.Warn("received invalid VAA in SignedVAAWithQuorum message",
			zap.Error(err), zap.Any("message", m))
		return
	}
	hash := v.SigningDigest()
	_, batchId := calculateBucket(hash[:], p.gst.MyKeyIndex(), len(p.gst.Get().Keys))
	p.inboundVaaChannels[batchId] <- v
}

func (p *ConcurrentProcessor) handleInboundSignedVAAWithQuorum(ctx context.Context, logger *zap.Logger, v *vaa.VAA) {
	logger.Debug("handleInboundSignedVAAWithQuorum", zap.String("messageId", v.MessageID()))
	// Check if we already store this VAA
	id := *db.VaaIDFromVAA(v)
	if p.haveSignedVAA(id) {
		logger.Debug("ignored SignedVAAWithQuorum message for VAA we already stored", zap.String("messageId", v.MessageID()))
		return
	}

	// Calculate digest for logging
	digest := v.SigningDigest()
	hash := hex.EncodeToString(digest.Bytes())

	gs := p.gst.Get()

	if gs == nil {
		logger.Warn("dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet",
			zap.String("digest", hash),
			zap.String("messageId", v.MessageID()),
		)
		return
	}

	// Check if guardianSet doesn't have any keys
	if len(gs.Keys) == 0 {
		logger.Warn("dropping SignedVAAWithQuorum message since we have a guardian set without keys",
			zap.String("digest", hash),
			zap.String("messageId", v.MessageID()),
		)
		return
	}

	if err := v.Verify(gs.Keys); err != nil {
		logger.Warn("dropping SignedVAAWithQuorum message because it failed verification: ",
			zap.Error(err),
			zap.String("messageId", v.MessageID()),
		)
		return
	}

	// We now established that:
	//  - all signatures on the VAA are valid
	//  - the signature's addresses match the node's current guardian set
	//  - enough signatures are present for the VAA to reach quorum

	// Store signed VAA in database.
	logger.Debug("storing inbound signed VAA with quorum",
		zap.String("digest", hash),
		zap.Any("vaa", v),
		zap.String("message_id", v.MessageID()))

	if err := p.storeSignedVAA(v); err != nil {
		logger.Error("failed to store signed VAA", zap.Error(err))
		return
	}
	p.reachedQuorumC <- hash
	p.attestationEvents.ReportVAAQuorum(v)
}

func (p *ConcurrentProcessor) handleSelfObservation(ctx context.Context, logger *zap.Logger, e selfObservationEvent) {
	logger.Debug("handleSelfObservation", zap.String("messageId", e.m.MessageIDString()))
	e.state.SelfObserved(e.m)
}
