package processor

import (
	"context"
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
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
func (p *Processor) handleMessage(ctx context.Context, k *common.MessagePublication) {
	supervisor.Logger(ctx).Info("message publication confirmed",
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

	v := &vaa.VAA{
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
	}

	// Generate digest of the unsigned VAA.
	digest, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}

	// Sign the digest using our node's guardian key.
	s, err := crypto.Sign(digest.Bytes(), p.gk)
	if err != nil {
		panic(err)
	}

	p.logger.Info("observed and signed confirmed message publication",
		zap.Stringer("source_chain", k.EmitterChain),
		zap.Stringer("txhash", k.TxHash),
		zap.String("digest", hex.EncodeToString(digest.Bytes())),
		zap.String("signature", hex.EncodeToString(s)))

	messagesSignedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String()}).Add(1)

	p.attestationEvents.ReportMessagePublication(&reporter.MessagePublication{VAA: *v, InitiatingTxID: k.TxHash})

	p.broadcastSignature(v, s)
}
