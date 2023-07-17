package processor3

import (
	"context"
	"encoding/hex"

	"github.com/mr-tron/base58"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// SECURITY: source_chain/target_chain are untrusted uint8 values. An attacker could cause a maximum of 255**2 label
	// pairs to be created, which is acceptable.

	messagesObservedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_total3",
			Help: "Total number of messages observed",
		},
		[]string{"emitter_chain"})

	messagesSignedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_signed_total3",
			Help: "Total number of message observations that were successfully signed",
		},
		[]string{"emitter_chain"})
)

// handleMessage processes a message received from a chain and instantiates our deterministic copy of the VAA. An
// event may be received multiple times and must be handled in an idempotent fashion.
func (p *ConcurrentProcessor) handleMessage(_ context.Context, logger *zap.Logger, k *common.MessagePublication) {
	if p.gst.Get() == nil {
		logger.Warn("dropping observation since we haven't initialized our guardian set yet", k.ZapFields()...)
		return
	}

	logger.Debug("message publication confirmed", k.ZapFields()...)

	messagesObservedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String(),
	}).Add(1)

	v := vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: p.gst.Get().Index,
		Signatures:       nil,
		Timestamp:        k.Timestamp,
		Nonce:            k.Nonce,
		EmitterChain:     k.EmitterChain,
		EmitterAddress:   k.EmitterAddress,
		Payload:          k.Payload,
		Sequence:         k.Sequence,
		ConsistencyLevel: k.ConsistencyLevel,
	}

	// A governance message should never be emitted on-chain
	if v.EmitterAddress == vaa.GovernanceEmitter && v.EmitterChain == vaa.GovernanceChain {
		logger.Error(
			"EMERGENCY: PLEASE REPORT THIS IMMEDIATELY! A Solana message was emitted from the governance emitter. This should never be possible.",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp))
		return
	}

	// Generate digest of the unsigned VAA.
	digest := v.SigningDigest()

	// Sign the digest using our node's guardian key.
	s, err := crypto.Sign(digest.Bytes(), p.gk)
	if err != nil {
		panic(err)
	}

	if logger.Level().Enabled(zapcore.DebugLevel) { // check if logging is enabled first for better performance
		logger.Debug("observed and signed confirmed message publication",
			zap.Stringer("source_chain", k.EmitterChain),
			zap.Stringer("txhash", k.TxHash),
			zap.String("txhash_b58", base58.Encode(k.TxHash.Bytes())),
			zap.String("digest", hex.EncodeToString(digest.Bytes())),
			zap.Uint32("nonce", k.Nonce),
			zap.Uint64("sequence", k.Sequence),
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.String("emitter_address_b58", base58.Encode(k.EmitterAddress.Bytes())),
			zap.Uint8("consistency_level", k.ConsistencyLevel),
			zap.String("message_id", v.MessageID()),
			zap.String("signature", hex.EncodeToString(s)))
	}

	messagesSignedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String()}).Add(1)

	p.attestationEvents.ReportMessagePublication(&reporter.MessagePublication{VAA: v, InitiatingTxID: k.TxHash})

	p.msgSelfObservedC <- k

	p.broadcastSignature(digest[:], s, k.TxHash.Bytes(), v.MessageID())
}

func (p *Processor) dispatchSelfObservation(_ context.Context, _ *zap.Logger, k *common.MessagePublication) {
	hashBytes := k.CreateHash()
	inLeaderSet, processingBucket := calculateBucket(hashBytes[:], p.gst.MyKeyIndex(), len(p.gst.Get().Keys))
	if inLeaderSet {

		hash := k.CreateDigest()
		s, ok := p.state[hash]

		if !ok {
			s = p.NewStateFromSelfObserved()
			p.state[hash] = s
		}

		p.selfObservationChannels[processingBucket] <- selfObservationEvent{
			m:     k,
			state: s,
		}
	}
}
