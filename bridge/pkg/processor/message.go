package processor

import (
	"context"
	"encoding/hex"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

var (
	// SECURITY: source_chain/target_chain are untrusted uint8 values. An attacker could cause a maximum of 255**2 label
	// pairs to be created, which is acceptable.

	lockupsObservedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_lockups_observed_total",
			Help: "Total number of lockups received on-chain",
		},
		[]string{"source_chain", "target_chain"})

	lockupsSignedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_lockups_signed_total",
			Help: "Total number of lockups that were successfully signed",
		},
		[]string{"source_chain", "target_chain"})
)

func init() {
	prometheus.MustRegister(lockupsObservedTotal)
	prometheus.MustRegister(lockupsSignedTotal)
}

// handleLockup processes a lockup received from a chain and instantiates our deterministic copy of the VAA. A lockup
// event may be received multiple times until it has been successfully completed.
func (p *Processor) handleLockup(ctx context.Context, k *common.MessagePublication) {
	supervisor.Logger(ctx).Info("message publication confirmed",
		zap.Stringer("emitter_chain", k.EmitterChain),
		zap.Stringer("emitter_address", k.EmitterAddress),
		zap.Uint32("nonce", k.Nonce),
		zap.Stringer("txhash", k.TxHash),
		zap.Time("timestamp", k.Timestamp),
	)

	lockupsObservedTotal.With(prometheus.Labels{
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

	lockupsSignedTotal.With(prometheus.Labels{
		"emitter_chain": k.EmitterChain.String()}).Add(1)

	p.broadcastSignature(v, s)
}
