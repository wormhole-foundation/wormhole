package processor

import (
	"encoding/hex"
	"time"

	"github.com/mr-tron/base58"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// SECURITY: source_chain/target_chain are untrusted uint8 values. An attacker could cause a maximum of 255**2 label
	// pairs to be created, which is acceptable.

	messagesObservedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_total",
			Help: "Total number of messages observed",
		},
		[]string{"emitter_chain"})
)

// handleMessage processes a message received from a chain and instantiates our deterministic copy of the VAA. An
// event may be received multiple times and must be handled in an idempotent fashion.
func (p *Processor) handleMessage(k *common.MessagePublication) {
	if p.gs == nil {
		p.logger.Warn("dropping observation since we haven't initialized our guardian set yet",
			zap.String("message_id", k.MessageIDString()),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp),
		)
		return
	}

	messagesObservedTotal.WithLabelValues(k.EmitterChain.String()).Inc()

	// All nodes will create the exact same VAA and sign its digest.
	// Consensus is established on this digest.

	v := &VAA{
		VAA: vaa.VAA{
			Version:          vaa.SupportedVAAVersion,
			GuardianSetIndex: p.gs.Index,
			Signatures:       nil,
			Timestamp:        k.Timestamp,
			Nonce:            k.Nonce,
			EmitterChain:     k.EmitterChain,
			EmitterAddress:   k.EmitterAddress,
			Payload:          k.Payload,
			Sequence:         k.Sequence,
			ConsistencyLevel: k.ConsistencyLevel,
		},
		Unreliable:    k.Unreliable,
		Reobservation: k.IsReobservation,
	}

	// Generate digest of the unsigned VAA.
	digest := v.SigningDigest()
	hash := hex.EncodeToString(digest.Bytes())

	// Sign the digest using the node's GuardianSigner
	signature, err := p.guardianSigner.Sign(digest.Bytes())
	if err != nil {
		panic(err)
	}

	shouldPublishImmediately := p.shouldPublishImmediately(&v.VAA)

	if p.logger.Core().Enabled(zapcore.DebugLevel) {
		p.logger.Debug("observed and signed confirmed message publication",
			zap.String("message_id", k.MessageIDString()),
			zap.Stringer("txhash", k.TxHash),
			zap.String("txhash_b58", base58.Encode(k.TxHash.Bytes())),
			zap.String("hash", hash),
			zap.Uint32("nonce", k.Nonce),
			zap.Time("timestamp", k.Timestamp),
			zap.Uint8("consistency_level", k.ConsistencyLevel),
			zap.String("signature", hex.EncodeToString(signature)),
			zap.Bool("shouldPublishImmediately", shouldPublishImmediately),
			zap.Bool("isReobservation", k.IsReobservation),
		)
	}

	// Broadcast the signature.
	ourObs, msg := p.broadcastSignature(v.MessageID(), k.TxHash.Bytes(), digest, signature, shouldPublishImmediately)

	// Indicate that we observed this one.
	observationsReceivedTotal.Inc()
	observationsReceivedByGuardianAddressTotal.WithLabelValues(p.ourAddr.Hex()).Inc()

	// Get / create our state entry.
	s := p.state.signatures[hash]
	if s == nil {
		s = &state{
			firstObserved: time.Now(),
			nextRetry:     time.Now().Add(nextRetryDuration(0)),
			signatures:    map[ethCommon.Address][]byte{},
			source:        "loopback",
		}

		p.state.signatures[hash] = s
	}

	// Update our state.
	s.ourObservation = v
	s.txHash = k.TxHash.Bytes()
	s.source = v.GetEmitterChain().String()
	s.gs = p.gs // guaranteed to match ourObservation - there's no concurrent access to p.gs
	s.signatures[p.ourAddr] = signature
	s.ourObs = ourObs
	s.ourMsg = msg

	// Fast path for our own signature.
	if !s.submitted {
		start := time.Now()
		p.checkForQuorum(ourObs, s, s.gs, hash)
		timeToHandleObservation.Observe(float64(time.Since(start).Microseconds()))
	}
}
