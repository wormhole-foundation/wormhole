//nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor

import (
	"encoding/hex"
	"fmt"
	"math"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_received_total",
			Help: "Total number of raw VAA observations received from gossip",
		})
	observationsReceivedByGuardianAddressTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_signed_by_guardian_total",
			Help: "Total number of signed and verified VAA observations grouped by guardian address",
		}, []string{"addr"})
	observationsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_observations_verification_failures_total",
			Help: "Total number of observations verification failure, grouped by failure reason",
		}, []string{"cause"})
	observationsUnknownTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_unknown_total",
			Help: "Total number of verified observations we haven't seen ourselves",
		})
)

// signaturesToVaaFormat converts a map[common.Address][]byte (processor state format) to []*vaa.Signature (VAA format) given a set of keys gsKeys
// The processor state format is used for efficiently storing signatures during aggregation while the VAA format is more efficient for on-chain verification.
func signaturesToVaaFormat(signatures map[common.Address][]byte, gsKeys []common.Address) []*vaa.Signature {
	// Aggregate all valid signatures into a list of vaa.Signature and construct signed VAA.
	var sigs []*vaa.Signature

	if len(gsKeys) > math.MaxUint8 {
		panic(fmt.Sprintf("guardian set too large: %d", len(gsKeys)))
	}

	for i, a := range gsKeys {
		sig, ok := signatures[a]

		if ok {
			var bs [65]byte
			if n := copy(bs[:], sig); n != 65 {
				panic(fmt.Sprintf("invalid sig len: %d", n))
			}

			sigs = append(sigs, &vaa.Signature{
				Index:     uint8(i), // #nosec G115 -- This is validated above
				Signature: bs,
			})
		}
	}
	return sigs
}

// handleBatchObservation processes a batch of remote VAA observations.
func (p *Processor) handleBatchObservation(m *node_common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]) {
	for _, obs := range m.Msg.Observations {
		p.handleSingleObservation(m.Msg.Addr, obs)
	}
	batchObservationTotalDelay.Observe(float64(time.Since(m.Timestamp).Microseconds()))
}

// handleObservation processes a remote VAA observation, verifies it, checks whether the VAA has met quorum, and assembles and submits a valid VAA if possible.
func (p *Processor) handleSingleObservation(addr []byte, m *gossipv1.Observation) {
	// SECURITY: at this point, observations received from the p2p network are fully untrusted (all fields!)
	//
	// Note that observations are never tied to the (verified) p2p identity key - the p2p network
	// identity is completely decoupled from the guardian identity, p2p is just transport.

	start := time.Now()
	observationsReceivedTotal.Inc()

	their_addr := common.BytesToAddress(addr)
	hash := hex.EncodeToString(m.Hash)
	s := p.state.signatures[hash]
	if s != nil && s.submitted {
		// already submitted; ignoring additional signatures for it.
		timeToHandleObservation.Observe(float64(time.Since(start).Microseconds()))
		return
	}

	if p.logger.Core().Enabled(zapcore.DebugLevel) {
		p.logger.Debug("received observation",
			zap.String("message_id", m.MessageId),
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(m.Signature)),
			zap.String("addr", hex.EncodeToString(addr)),
			zap.String("txhash", hex.EncodeToString(m.TxHash)),
			zap.String("txhash_b58", base58.Encode(m.TxHash)),
		)
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
	var gs *node_common.GuardianSet
	if s != nil && s.gs != nil {
		gs = s.gs
	} else {
		gs = p.gs
	}

	// We haven't yet observed the trusted guardian set on Ethereum, and therefore, it's impossible to verify it.
	// May as well not have received it/been offline - drop it and wait for the guardian set.
	if gs == nil {
		p.logger.Warn("dropping observations since we haven't initialized our guardian set yet",
			zap.String("messageId", m.MessageId),
			zap.String("digest", hash),
			zap.String("their_addr", their_addr.Hex()),
		)
		observationsFailedTotal.WithLabelValues("uninitialized_guardian_set").Inc()
		return
	}

	// Verify that addr is included in the guardian set. If it's not, drop the message. In case it's us
	// who have the outdated guardian set, we'll just wait for the message to be retransmitted eventually.
	_, ok := gs.KeyIndex(their_addr)
	if !ok {
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("received observation by unknown guardian - is our guardian set outdated?",
				zap.String("messageId", m.MessageId),
				zap.String("digest", hash),
				zap.String("their_addr", their_addr.Hex()),
				zap.Uint32("index", gs.Index),
				//zap.Any("keys", gs.KeysAsHexStrings()),
			)
		}
		observationsFailedTotal.WithLabelValues("unknown_guardian").Inc()
		return
	}

	// Verify the Guardian's signature. This verifies that m.Signature matches m.Hash and recovers
	// the public key that was used to sign the payload.
	pk, err := crypto.Ecrecover(m.Hash, m.Signature)
	if err != nil {
		p.logger.Warn("failed to verify signature on observation",
			zap.String("messageId", m.MessageId),
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(m.Signature)),
			zap.String("addr", hex.EncodeToString(addr)),
			zap.Error(err))
		observationsFailedTotal.WithLabelValues("invalid_signature").Inc()
		return
	}

	// Verify that addr matches the public key that signed m.Hash.
	signer_pk := common.BytesToAddress(crypto.Keccak256(pk[1:])[12:])

	if their_addr != signer_pk {
		p.logger.Info("invalid observation - address does not match pubkey",
			zap.String("messageId", m.MessageId),
			zap.String("digest", hash),
			zap.String("signature", hex.EncodeToString(m.Signature)),
			zap.String("addr", hex.EncodeToString(addr)),
			zap.String("pk", signer_pk.Hex()))
		observationsFailedTotal.WithLabelValues("pubkey_mismatch").Inc()
		return
	}

	// Hooray! Now, we have verified all fields on the observation and know that it includes
	// a valid signature by an active guardian. We still don't fully trust them, as they may be
	// byzantine, but now we know who we're dealing with.

	// We can now count events by guardian without worry about cardinality explosions:
	observationsReceivedByGuardianAddressTotal.WithLabelValues(their_addr.Hex()).Inc()

	// []byte isn't hashable in a map. Paying a small extra cost for encoding for easier debugging.
	if s == nil {
		// We haven't yet seen this event ourselves, and therefore do not know what the VAA looks like.
		// However, we have established that a valid guardian has signed it, and therefore we can
		// already start aggregating signatures for it.
		//
		// A malicious guardian can potentially DoS this by creating fake observations at a faster rate than they decay,
		// leading to a slow out-of-memory crash. We do not attempt to automatically mitigate spam attacks with valid
		// signatures - such byzantine behavior would be plainly visible and would be dealt with by kicking them.

		observationsUnknownTotal.Inc()

		s = &state{
			firstObserved: time.Now(),
			nextRetry:     time.Now().Add(nextRetryDuration(0)),
			signatures:    map[common.Address][]byte{},
			source:        "unknown",
		}

		p.state.signatures[hash] = s
	}

	s.signatures[their_addr] = m.Signature

	if s.ourObservation != nil {
		p.checkForQuorum(m, s, gs, hash)
	} else {
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("we have not yet seen this observation yet",
				zap.String("messageId", m.MessageId),
				zap.String("digest", hash),
			)
		}
		// Keep going to update metrics.
	}

	timeToHandleObservation.Observe(float64(time.Since(start).Microseconds()))
}

// checkForQuorum checks for quorum after a valid signature has been added to the observation state. If quorum is met, it broadcasts the signed VAA. This function
// is called both for local and external observations. It assumes we that we have made the observation ourselves but have not already submitted the VAA.
func (p *Processor) checkForQuorum(m *gossipv1.Observation, s *state, gs *node_common.GuardianSet, hash string) {
	// Check if we have more signatures than required for quorum.
	// s.signatures may contain signatures from multiple guardian sets during guardian set updates
	// Hence, if len(s.signatures) < quorum, then there is definitely no quorum and we can return early to save additional computation,
	// but if len(s.signatures) >= quorum, there is not necessarily quorum for the active guardian set.
	// We will later check for quorum again after assembling the VAA for a particular guardian set.
	if len(s.signatures) < gs.Quorum() {
		// no quorum yet, we're done here
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("quorum not yet met",
				zap.String("messageId", m.MessageId),
				zap.String("digest", hash),
			)
		}
		return
	}

	// Now we *may* have quorum, depending on the guardian set in use.
	// Let's construct the VAA and check if we actually have quorum.
	sigsVaaFormat := signaturesToVaaFormat(s.signatures, gs.Keys)

	if p.logger.Level().Enabled(zapcore.DebugLevel) {
		p.logger.Debug("aggregation state for observation", // 1.3M out of 3M info messages / hour / guardian
			zap.String("messageId", m.MessageId),
			zap.String("digest", hash),
			zap.Any("set", gs.KeysAsHexStrings()),
			zap.Uint32("index", gs.Index),
			zap.Int("required_sigs", gs.Quorum()),
			zap.Int("have_sigs", len(sigsVaaFormat)),
			zap.Bool("quorum", len(sigsVaaFormat) >= gs.Quorum()),
		)
	}

	if len(sigsVaaFormat) < gs.Quorum() {
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("quorum not met, doing nothing",
				zap.String("messageId", m.MessageId),
				zap.String("digest", hash),
			)
		}
		return
	}

	// We have reached quorum *with the active guardian set*.
	start := time.Now()
	s.ourObservation.HandleQuorum(sigsVaaFormat, hash, p)
	s.submitted = true
	timeToHandleQuorum.Observe(float64(time.Since(start).Microseconds()))
}

// handleInboundSignedVAAWithQuorum takes a VAA received from the network. If we have not already seen it and it is valid, we store it in the database.
func (p *Processor) handleInboundSignedVAAWithQuorum(m *gossipv1.SignedVAAWithQuorum) {
	v, err := vaa.Unmarshal(m.Vaa)
	if err != nil {
		p.logger.Warn("received invalid VAA in SignedVAAWithQuorum message",
			zap.Error(err), zap.Any("message", m))
		return
	}

	// Check if we already store this VAA
	if p.haveSignedVAA(*db.VaaIDFromVAA(v)) {
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("ignored SignedVAAWithQuorum message for VAA we already stored",
				zap.String("message_id", v.MessageID()),
			)
		}
		return
	}

	if p.gs == nil {
		p.logger.Warn("dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet",
			zap.String("message_id", v.MessageID()),
			zap.String("digest", hex.EncodeToString(v.SigningDigest().Bytes())),
			zap.Any("message", m),
		)
		return
	}

	// Check if guardianSet doesn't have any keys
	if len(p.gs.Keys) == 0 {
		p.logger.Warn("dropping SignedVAAWithQuorum message since we have a guardian set without keys",
			zap.String("message_id", v.MessageID()),
			zap.String("digest", hex.EncodeToString(v.SigningDigest().Bytes())),
			zap.Any("message", m),
		)
		return
	}

	if err := v.Verify(p.gs.Keys); err != nil {
		// We format the error as part of the message so the tests can check for it.
		p.logger.Warn("dropping SignedVAAWithQuorum message because it failed verification: "+err.Error(), zap.String("message_id", v.MessageID()))
		return
	}

	// We now established that:
	//  - all signatures on the VAA are valid
	//  - the signature's addresses match the node's current guardian set
	//  - enough signatures are present for the VAA to reach quorum

	// Store signed VAA in database.
	if p.logger.Level().Enabled(zapcore.DebugLevel) {
		p.logger.Debug("storing inbound signed VAA with quorum",
			zap.String("message_id", v.MessageID()),
			zap.String("digest", hex.EncodeToString(v.SigningDigest().Bytes())),
			zap.Any("vaa", v),
			zap.String("bytes", hex.EncodeToString(m.Vaa)),
		)
	}

	p.storeSignedVAA(v)
}
