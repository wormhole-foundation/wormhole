package processor

import (
	"context"
	"encoding/hex"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/mr-tron/base58"

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
	}

	// A governance message should never be emitted on-chain
	if v.EmitterAddress == vaa.GovernanceEmitter && v.EmitterChain == vaa.GovernanceChain {
		supervisor.Logger(ctx).Error(
			"EMERGENCY: PLEASE REPORT THIS IMMEDIATELY! A Solana message was emitted from the governance emitter. This should never be possible.",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp))
		return
	}

	// Ignore incoming observations when our database already has a quorum VAA for it.
	// This can occur when we're receiving late observations due to node catchup, and
	// processing those won't do us any good.
	//
	// Exception: if an observation is made within the settlement time (30s), we'll
	// process it so other nodes won't consider it a miss.
	if vb, err := p.db.GetSignedVAABytes(*db.VaaIDFromVAA(&v.VAA)); err == nil {
		// unmarshal vaa
		var existing *vaa.VAA
		if existing, err = vaa.Unmarshal(vb); err != nil {
			panic("failed to unmarshal VAA from db")
		}

		if k.Timestamp.Sub(existing.Timestamp) > settlementTime {
			p.logger.Info("ignoring observation since we already have a quorum VAA for it",
				zap.Stringer("emitter_chain", k.EmitterChain),
				zap.Stringer("emitter_address", k.EmitterAddress),
				zap.String("emitter_address_b58", base58.Encode(k.EmitterAddress.Bytes())),
				zap.Uint32("nonce", k.Nonce),
				zap.Stringer("txhash", k.TxHash),
				zap.String("txhash_b58", base58.Encode(k.TxHash.Bytes())),
				zap.Time("timestamp", k.Timestamp),
				zap.String("message_id", v.MessageID()),
				zap.Duration("settlement_time", settlementTime),
			)
			return
		}
	} else if err != db.ErrVAANotFound {
		p.logger.Error("failed to get VAA from db",
			zap.Stringer("emitter_chain", k.EmitterChain),
			zap.Stringer("emitter_address", k.EmitterAddress),
			zap.Uint32("nonce", k.Nonce),
			zap.Stringer("txhash", k.TxHash),
			zap.Time("timestamp", k.Timestamp),
			zap.Error(err),
		)
	}

	// Generate digest of the unsigned VAA.
	digest := v.SigningMsg()

	// Sign the digest using our node's guardian key.
	s, err := crypto.Sign(digest.Bytes(), p.gk)
	if err != nil {
		panic(err)
	}

	p.logger.Info("observed and signed confirmed message publication",
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
