package processor

import (
	"context"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

// handleLockup processes a lockup received from a chain and instantiates our deterministic copy of the VAA
func (p *Processor) handleLockup(ctx context.Context, k *common.ChainLock) {
	supervisor.Logger(ctx).Info("lockup confirmed",
		zap.Stringer("source_chain", k.SourceChain),
		zap.Stringer("target_chain", k.TargetChain),
		zap.Stringer("source_addr", k.SourceAddress),
		zap.Stringer("target_addr", k.TargetAddress),
		zap.Stringer("token_chain", k.TokenChain),
		zap.Stringer("token_addr", k.TokenAddress),
		zap.Stringer("amount", k.Amount),
		zap.Stringer("txhash", k.TxHash),
		zap.Time("timestamp", k.Timestamp),
	)

	if p.gs == nil {
		p.logger.Warn("received observation, but we don't know the guardian set yet")
		return
	}

	us, ok := p.gs.KeyIndex(p.ourAddr)
	if !ok {
		p.logger.Error("we're not in the guardian set - refusing to sign",
			zap.Uint32("index", p.gs.Index),
			zap.Stringer("our_addr", p.ourAddr),
			zap.Any("set", p.gs.KeysAsHexStrings()))
		return
	}

	// All nodes will create the exact same VAA and sign its digest.
	// Consensus is established on this digest.

	v := &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: p.gs.Index,
		Signatures:       nil,
		Timestamp:        k.Timestamp,
		Payload: &vaa.BodyTransfer{
			Nonce:         k.Nonce,
			SourceChain:   k.SourceChain,
			TargetChain:   k.TargetChain,
			SourceAddress: k.SourceAddress,
			TargetAddress: k.TargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    k.TokenChain,
				Address:  k.TokenAddress,
				Decimals: k.TokenDecimals,
			},
			Amount: k.Amount,
		},
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

	p.logger.Info("observed and signed confirmed lockup",
		zap.Stringer("source_chain", k.SourceChain),
		zap.Stringer("target_chain", k.TargetChain),
		zap.Stringer("txhash", k.TxHash),
		zap.String("digest", hex.EncodeToString(digest.Bytes())),
		zap.String("signature", hex.EncodeToString(s)),
		zap.Int("our_index", us))

	p.broadcastSignature(v, s)
}
