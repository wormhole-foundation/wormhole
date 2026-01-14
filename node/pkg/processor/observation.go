package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	guardianNotary "github.com/certusone/wormhole/node/pkg/notary"
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
	delegateObservationsReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_delegate_observations_received_total",
			Help: "Total number of delegate observations received from gossip",
		})
	delegateObservationsReceivedByGuardianAddressTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_delegate_observations_by_guardian_total",
			Help: "Total number of valid delegate observations grouped by guardian address",
		}, []string{"addr"})
	delegateObservationsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_delegate_observations_verification_failures_total",
			Help: "Total number of delegate observations verification failure, grouped by failure reason",
		}, []string{"cause"})
	delegateObservationsUnknownTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_delegate_observations_unknown_total",
			Help: "Total number of valid delegate observations we haven't seen ourselves",
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

// handleDelegateMessagePublication converts the MessagePublication into a DelegateObservation and sends it to the delegateObsvSendC channel.
// This should only be called by a delegated guardian for the chain.
func (p *Processor) handleDelegateMessagePublication(k *node_common.MessagePublication) error {
	p.logger.Info("handleDelegateMessagePublication: CALLED - converting message to delegate observation",
		zap.String("msgID", k.MessageIDString()),
		zap.Uint32("emitter_chain", uint32(k.EmitterChain)),
		zap.Uint64("sequence", k.Sequence),
	)
	d, err := messagePublicationToDelegateObservation(k)
	if err != nil {
		p.logger.Warn("failed to build delegate observation from message publication",
			zap.String("msgID", k.MessageIDString()),
			zap.Error(err),
		)
		return err
	}
	d.GuardianAddr = p.ourAddr.Bytes()

	p.logger.Debug("handleDelegateMessagePublication: delegate observation created, sending to channel",
		zap.String("msgID", k.MessageIDString()),
		zap.Uint32("emitter_chain", d.EmitterChain),
		zap.Uint64("sequence", d.Sequence),
		zap.String("guardian_addr", p.ourAddr.Hex()),
	)

	select {
	case p.delegateObsvSendC <- d:
		p.logger.Debug("handleDelegateMessagePublication: successfully sent to delegateObsvSendC channel",
			zap.String("msgID", k.MessageIDString()),
		)
	default:
		p.logger.Warn("delegate observation send channel full, dropping",
			zap.String("msgID", k.MessageIDString()),
			zap.Uint32("emitter_chain", d.EmitterChain),
			zap.Uint64("sequence", d.Sequence),
		)
	}
	return nil
}

// This is the main message processing loop. It is responsible for handling messages that are
// received on the message channel. Depending on the configuration, a message may be processed
// by the Notary, the Governor, and/or the Accountant.
// This loop effectively causes each of these components to process messages in a modular
// manner. The Notary, Governor, and Accountant can be enabled or disabled independently.
// As a consequence of this loop, each of these components updates its internal state, tracking
// whether a message is ready to be processed from its perspective. This state is used by the
// processor to determine whether a message should be processed or not. This occurs elsewhere
// in the processor code.
func (p *Processor) handleMessagePublication(ctx context.Context, k *node_common.MessagePublication) error {
	if !p.processWithNotary(k) || !p.processWithGovernor(k) {
		return nil
	}

	return p.processWithAccountant(ctx, k)
}

// processWithNotary processes a message using the Notary to check whether it is well-formed.
// Returns true if the message was processed successfully and we can continue processing the message.
func (p *Processor) processWithNotary(k *node_common.MessagePublication) bool {
	// Track transfer verification states for analytics and log unusual states
	p.trackVerificationState(k)

	// Notary: check whether a message is well-formed.
	if p.notary != nil {
		p.logger.Debug("processor: sending message to notary for evaluation", k.ZapFields()...)

		// NOTE: Always returns Approve for messages that are not token transfers.
		verdict, err := p.notary.ProcessMsg(k)
		if err != nil {
			// TODO: The error is deliberately ignored so that the processor does not panic and restart.
			// In contrast, the Accountant does not ignore the error and restarts the processor if it fails.
			// The error-handling strategy can be revisited once the Notary is considered stable.
			p.logger.Error("notary failed to process message", zap.Error(err), zap.String("messageID", k.MessageIDString()))
			return false
		}

		// Based on the verdict, we can decide what to do with the message.
		switch verdict {
		case guardianNotary.Blackhole, guardianNotary.Delay:
			p.logger.Error("notary evaluated message as threatening", k.ZapFields(zap.String("verdict", verdict.String()))...)
			if verdict == guardianNotary.Blackhole {
				// Black-holed messages should not be processed.
				p.logger.Error("message will not be processed", k.ZapFields(zap.String("verdict", verdict.String()))...)
			} else {
				// Delayed messages are added to a separate queue and processed elsewhere.
				p.logger.Error("message will be delayed", k.ZapFields(zap.String("verdict", verdict.String()))...)
			}
			// We're done processing the message.
			return false
		case guardianNotary.Unknown:
			p.logger.Error("notary returned Unknown verdict", k.ZapFields(zap.String("verdict", verdict.String()))...)
		case guardianNotary.Approve:
			// no-op: process normally
			p.logger.Debug("notary evaluated message as approved", k.ZapFields(zap.String("verdict", verdict.String()))...)
		default:
			p.logger.Error("notary returned unrecognized verdict", k.ZapFields(zap.String("verdict", verdict.String()))...)
		}
	}

	return true
}

// processWithGovernor processes a message using the Governor to check if it is ready to be published.
// Returns true if the message was processed successfully and we can continue processing the message.
func (p *Processor) processWithGovernor(k *node_common.MessagePublication) bool {
	if p.governor != nil {
		if !p.governor.ProcessMsg(k) {
			// We're done processing the message.
			return false
		}
	}
	return true
}

// processWithAccountant processes a message using the Accountant to check if it is ready to be published
// (i.e. if it has enough observations).
func (p *Processor) processWithAccountant(ctx context.Context, k *node_common.MessagePublication) error {
	if p.acct != nil {
		shouldPub, err := p.acct.SubmitObservation(k)
		if err != nil {
			return fmt.Errorf("accountant: failed to process message `%s`: %w", k.MessageIDString(), err)
		}
		if !shouldPub {
			// We're done processing the message.
			return nil
		}
	}
	p.handleMessage(ctx, k)
	return nil
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

	// Normalize guardian set index if needed
	if v.GuardianSetIndex != p.gs.Index {
		p.logger.Info("normalizing guardian set index on inbound signed VAA",
			zap.String("message_id", v.MessageID()),
			zap.String("digest", hex.EncodeToString(v.SigningDigest().Bytes())),
			zap.Uint32("from_index", v.GuardianSetIndex),
			zap.Uint32("to_index", p.gs.Index),
		)
		v.GuardianSetIndex = p.gs.Index
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

// handleDelegateObservation processes a delegate observation
func (p *Processor) handleDelegateObservation(ctx context.Context, m *gossipv1.DelegateObservation) error {
	delegateObservationsReceivedTotal.Inc()

	if p.logger.Core().Enabled(zapcore.DebugLevel) {
		p.logger.Debug("received delegate observation",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("txhash", hex.EncodeToString(m.TxHash)),
			zap.String("txhash_b58", base58.Encode(m.TxHash)),
			zap.String("guardian_addr", hex.EncodeToString(m.GuardianAddr)),
			zap.Uint32("timestamp", m.Timestamp),
			zap.Int("payload_len", len(m.Payload)),
		)
	}

	c, err := vaa.KnownChainIDFromNumber(m.EmitterChain)
	if err != nil {
		p.logger.Warn("invalid delegate observation emitter chain",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("guardian_addr", hex.EncodeToString(m.GuardianAddr)),
			zap.Error(err),
		)
		delegateObservationsFailedTotal.WithLabelValues("invalid_emitter_chain").Inc()
		return nil
	}

	cfg := p.dgc.GetChainConfig(c)
	if cfg == nil {
		p.logger.Debug("ignoring delegate observation for chain without delegate chain config",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("guardian_addr", hex.EncodeToString(m.GuardianAddr)),
		)
		delegateObservationsFailedTotal.WithLabelValues("no_delegate_chain_config").Inc()
		return nil
	}

	_, ok := cfg.KeyIndex(p.ourAddr)
	if ok {
		p.logger.Debug("ignoring delegate observation since we are a delegated guardian for this chain",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("guardian_addr", hex.EncodeToString(m.GuardianAddr)),
		)
		delegateObservationsFailedTotal.WithLabelValues("self_delegated_guardian").Inc()
		return nil
	}

	addr := common.BytesToAddress(m.GuardianAddr)
	_, ok = cfg.KeyIndex(addr)
	if !ok {
		p.logger.Debug("ignoring delegate observation from non-delegated guardian for this chain",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("guardian_addr", addr.Hex()),
		)
		delegateObservationsFailedTotal.WithLabelValues("unknown_delegated_guardian").Inc()
		return nil
	}

	return p.handleCanonicalDelegateObservation(ctx, cfg, m)
}

// handleCanonicalDelegateObservation processes a delegate observation as a canonical guardian
// This function assumes cfg corresponds to m.EmitterChain
// TODO(delegated-guardian-sets): Should ^ be explicitly asserted?
func (p *Processor) handleCanonicalDelegateObservation(ctx context.Context, cfg *DelegatedGuardianChainConfig, m *gossipv1.DelegateObservation) error {
	addr := common.BytesToAddress(m.GuardianAddr)
	mp, err := delegateObservationToMessagePublication(m)
	if err != nil {
		p.logger.Warn("failed to convert delegate observation to message publication",
			zap.Uint32("emitter_chain", m.EmitterChain),
			zap.String("emitter_address", hex.EncodeToString(m.EmitterAddress)),
			zap.Uint64("sequence", m.Sequence),
			zap.String("guardian_addr", addr.Hex()),
			zap.Error(err),
		)
		delegateObservationsFailedTotal.WithLabelValues("invalid_delegate_observation").Inc()
		return nil
	}

	delegateObservationsReceivedByGuardianAddressTotal.WithLabelValues(addr.Hex()).Inc()

	hash := mp.CreateDigest()

	// Get / create our state entry.
	s := p.delegateState.observations[hash]
	if s == nil {
		delegateObservationsUnknownTotal.Inc()

		s = &delegateState{
			firstObserved: time.Now(),
			observations:  map[common.Address]*gossipv1.DelegateObservation{},
		}
		p.delegateState.observations[hash] = s
	}

	// Update our state.
	s.observations[addr] = m

	if !s.submitted {
		return p.checkForDelegateQuorum(ctx, mp, s, cfg)
	}
	return nil
}

// checkForDelegateQuorum checks for quorum after a delegate observation has been added to the state. If quorum is met, it runs the converted
// MessagePublication through the normal message pipeline.
// This function assumes mp corresponds to s
// TODO(delegated-guardian-sets): Should ^ be explicitly asserted?
func (p *Processor) checkForDelegateQuorum(ctx context.Context, mp *node_common.MessagePublication, s *delegateState, dgs *DelegatedGuardianChainConfig) error {
	// TODO(delegated-guardian-sets): Handle case for when delegated guardian set changes
	// Check if we have more delegate observations than required for quorum.
	if len(s.observations) < dgs.Quorum() {
		// no quorum yet, we're done here
		if p.logger.Level().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("quorum not yet met",
				zap.Stringer("emitter_chain", mp.EmitterChain),
				zap.Uint64("sequence", mp.Sequence),
			)
		}
		return nil
	}

	s.submitted = true
	return p.handleMessagePublication(ctx, mp)
}

// delegateObservationToMessagePublication converts a DelegateObservation into a MessagePublication that can be passed through the normal processor pipeline.
func delegateObservationToMessagePublication(d *gossipv1.DelegateObservation) (*node_common.MessagePublication, error) {
	const TxIDSizeMax = math.MaxUint8
	txIDLen := len(d.TxHash)
	if txIDLen > TxIDSizeMax {
		return nil, fmt.Errorf("delegate observation tx_hash too long: got %d; want at most %d", txIDLen, TxIDSizeMax)
	}
	if txIDLen < node_common.TxIDLenMin {
		return nil, fmt.Errorf("delegate observation tx_hash too short: got %d; want at least %d", txIDLen, node_common.TxIDLenMin)
	}

	if d.ConsistencyLevel > math.MaxUint8 {
		return nil, fmt.Errorf("invalid delegate observation consistency : %d", d.ConsistencyLevel)
	}

	c, err := vaa.KnownChainIDFromNumber(d.EmitterChain)
	if err != nil {
		return nil, fmt.Errorf("invalid delegate observation emitter chain: %w", err)
	}

	addr, err := vaa.BytesToAddress(d.EmitterAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid delegate observation emitter address: %w", err)
	}

	mp := &node_common.MessagePublication{
		TxID:             d.TxHash,
		Timestamp:        time.Unix(int64(d.Timestamp), 0), // Timestamp is uint32 representing seconds since UNIX epoch so is safe to convert.
		Nonce:            d.Nonce,
		Sequence:         d.Sequence,
		ConsistencyLevel: uint8(d.ConsistencyLevel),
		EmitterChain:     c,
		EmitterAddress:   addr,
		Payload:          d.Payload,
		IsReobservation:  false,
		Unreliable:       false,
		// verificationState intentionally left at the default (NotVerified).
	}

	return mp, nil
}

// messagePublicationToDelegateObservation converts a MessagePublication into a DelegateObservation to be sent by a delegated guardian.
// This does not populate the GuardianAddr field.
func messagePublicationToDelegateObservation(m *node_common.MessagePublication) (*gossipv1.DelegateObservation, error) {
	const TxIDSizeMax = math.MaxUint8
	txIDLen := len(m.TxID)
	if txIDLen > TxIDSizeMax {
		return nil, fmt.Errorf("message publication tx_hash too long: got %d; want at most %d", txIDLen, TxIDSizeMax)
	}
	if txIDLen < node_common.TxIDLenMin {
		return nil, fmt.Errorf("message publication tx_hash too short: got %d; want at least %d", txIDLen, node_common.TxIDLenMin)
	}

	d := &gossipv1.DelegateObservation{
		Timestamp:        uint32(m.Timestamp.Unix()), // #nosec G115 -- This conversion is safe until year 2106
		Nonce:            m.Nonce,
		EmitterChain:     uint32(m.EmitterChain),
		EmitterAddress:   m.EmitterAddress.Bytes(),
		Sequence:         m.Sequence,
		ConsistencyLevel: uint32(m.ConsistencyLevel),
		Payload:          m.Payload,
		TxHash:           m.TxID,
		// GuardianAddr will be populated in handleDelegateMessagePublication before p2p broadcast.
	}

	return d, nil
}
