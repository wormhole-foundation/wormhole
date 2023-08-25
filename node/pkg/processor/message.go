package processor

import (
	"encoding/hex"

	"github.com/mr-tron/base58"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/reporter"
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

	messagesSignedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_message_observations_signed_total",
			Help: "Total number of message observations that were successfully signed",
		},
		[]string{"emitter_chain"})
)

// handleMessage processes a message received from a chain and instantiates our deterministic copy of the VAA. An
// event may be received multiple times and must be handled in an idempotent fashion.
func (p *Processor) handleMessage(k *common.MessagePublication) {
	if p.gs == nil {
		p.logger.Warn("dropping observation since we haven't initialized our guardian set yet",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp),
		)
		return
	}

	p.logger.Debug("message publication confirmed",
		zap.Stringer("emitter_chain", k.EmitterChain),
		zap.Stringer("emitter_address", k.EmitterAddress),
		zap.Uint32("nonce", k.Nonce),
		zap.Stringer("txhash", k.TxHash),
		zap.Time("timestamp", k.Timestamp),
	)

	messagesObservedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String(),
	}).Add(1)

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
		Unreliable: k.Unreliable,
	}

	// Generate digest of the unsigned VAA.
	digest := v.SigningDigest()

	// Sign the digest using our node's guardian key.
	s, err := crypto.Sign(digest.Bytes(), p.gk)
	if err != nil {
		panic(err)
	}

	p.logger.Debug("observed and signed confirmed message publication",
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

	messagesSignedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String()}).Add(1)

	p.attestationEvents.ReportMessagePublication(&reporter.MessagePublication{VAA: v.VAA, InitiatingTxID: k.TxHash})

	p.broadcastSignature(v, s, k.TxHash.Bytes())
}
