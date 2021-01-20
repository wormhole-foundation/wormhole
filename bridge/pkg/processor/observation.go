package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/terra"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

// handleObservation processes a remote VAA observation, verifies it, checks whether the VAA has met quorum,
// and assembles and submits a valid VAA if possible.
func (p *Processor) handleObservation(ctx context.Context, m *gossipv1.SignedObservation) {
	// SECURITY: at this point, observations received from the p2p network are fully untrusted (all fields!)
	//
	// Note that observations are never tied to the (verified) p2p identity key - the p2p network
	// identity is completely decoupled from the guardian identity, p2p is just transport.

	p.logger.Info("received lockup observation",
		zap.String("digest", hex.EncodeToString(m.Hash)),
		zap.String("signature", hex.EncodeToString(m.Signature)),
		zap.String("addr", hex.EncodeToString(m.Addr)))

	// Verify the Guardian's signature. This verifies that m.Signature matches m.Hash and recovers
	// the public key that was used to sign the payload.
	pk, err := crypto.Ecrecover(m.Hash, m.Signature)
	if err != nil {
		p.logger.Warn("failed to verify signature on lockup observation",
			zap.String("digest", hex.EncodeToString(m.Hash)),
			zap.String("signature", hex.EncodeToString(m.Signature)),
			zap.String("addr", hex.EncodeToString(m.Addr)),
			zap.Error(err))
		return
	}

	// Verify that m.Addr matches the public key that signed m.Hash.
	their_addr := common.BytesToAddress(m.Addr)
	signer_pk := common.BytesToAddress(crypto.Keccak256(pk[1:])[12:])

	if their_addr != signer_pk {
		p.logger.Info("invalid lockup observation - address does not match pubkey",
			zap.String("digest", hex.EncodeToString(m.Hash)),
			zap.String("signature", hex.EncodeToString(m.Signature)),
			zap.String("addr", hex.EncodeToString(m.Addr)),
			zap.String("pk", signer_pk.Hex()))
		return
	}

	// Verify that m.Addr is included in the current guardian set.
	_, ok := p.gs.KeyIndex(their_addr)
	if !ok {
		p.logger.Warn("received observation by unknown guardian - is our guardian set outdated?",
			zap.String("their_addr", their_addr.Hex()),
			zap.Any("current_set", p.gs.KeysAsHexStrings()),
		)
		return
	}

	// Hooray! Now, we have verified all fields on SignedObservation and know that it includes
	// a valid signature by an active guardian. We still don't fully trust them, as they may be
	// byzantine, but now we know who we're dealing with.

	// []byte isn't hashable in a map. Paying a small extra cost for encoding for easier debugging.
	hash := hex.EncodeToString(m.Hash)

	if p.state.vaaSignatures[hash] == nil {
		// We haven't yet seen this lockup ourselves, and therefore  do not know what the VAA looks like.
		// However, we have established that a valid guardian has signed it, and therefore we can
		// already start aggregating signatures for it.
		//
		// A malicious guardian can potentially DoS this by creating fake lockups at a faster rate than they decay,
		// leading to a slow out-of-memory crash. We do not attempt to automatically mitigate spam attacks with valid
		// signatures - such byzantine behavior would be plainly visible and would be dealt with by kicking them.

		p.state.vaaSignatures[hash] = &vaaState{
			firstObserved: time.Now(),
			signatures:    map[common.Address][]byte{},
		}
	}

	p.state.vaaSignatures[hash].signatures[their_addr] = m.Signature

	// Aggregate all valid signatures into a list of vaa.Signature and construct signed VAA.
	agg := make([]bool, len(p.gs.Keys))
	var sigs []*vaa.Signature
	for i, a := range p.gs.Keys {
		s, ok := p.state.vaaSignatures[hash].signatures[a]

		if ok {
			var bs [65]byte
			if n := copy(bs[:], s); n != 65 {
				panic(fmt.Sprintf("invalid sig len: %d", n))
			}

			sigs = append(sigs, &vaa.Signature{
				Index:     uint8(i),
				Signature: bs,
			})
		}

		agg[i] = ok
	}

	if p.state.vaaSignatures[hash].ourVAA != nil {
		// We have seen it on chain!
		// Deep copy the VAA and add signatures
		v := p.state.vaaSignatures[hash].ourVAA
		signed := &vaa.VAA{
			Version:          v.Version,
			GuardianSetIndex: v.GuardianSetIndex,
			Signatures:       sigs,
			Timestamp:        v.Timestamp,
			Payload:          v.Payload,
		}

		// 2/3+ majority required for VAA to be valid - wait until we have quorum to submit VAA.
		quorum := CalculateQuorum(len(p.gs.Keys))

		p.logger.Info("aggregation state for VAA",
			zap.String("digest", hash),
			zap.Any("set", p.gs.KeysAsHexStrings()),
			zap.Uint32("index", p.gs.Index),
			zap.Bools("aggregation", agg),
			zap.Int("required_sigs", quorum),
			zap.Int("have_sigs", len(sigs)),
		)

		if len(sigs) >= quorum && !p.state.vaaSignatures[hash].submitted {
			vaaBytes, err := signed.Marshal()
			if err != nil {
				panic(err)
			}

			// Submit every VAA to Solana for data availability.
			p.logger.Info("submitting signed VAA to Solana",
				zap.String("digest", hash),
				zap.Any("vaa", signed),
				zap.String("bytes", hex.EncodeToString(vaaBytes)))
			p.vaaC <- signed

			switch t := v.Payload.(type) {
			case *vaa.BodyTransfer:
				// Depending on the target chain, guardians submit VAAs directly to the chain.

				switch t.TargetChain {
				case vaa.ChainIDSolana:
					// No-op.
				case vaa.ChainIDEthereum:
					// Ethereum is special because it's expensive, and guardians cannot
					// be expected to pay the fees. We only submit to Ethereum in devnet mode.
					p.devnetVAASubmission(ctx, signed, hash)
				case vaa.ChainIDTerra:
					go p.terraVAASubmission(ctx, signed, hash)
				default:
					p.logger.Error("unknown target chain ID",
						zap.String("digest", hash),
						zap.Any("vaa", signed),
						zap.String("bytes", hex.EncodeToString(vaaBytes)),
						zap.Stringer("target_chain", t.TargetChain))
				}
			case *vaa.BodyGuardianSetUpdate:
				// A guardian set update is broadcast to every chain that we talk to.
				p.devnetVAASubmission(ctx, signed, hash)
				p.terraVAASubmission(ctx, signed, hash)
			case *vaa.BodyContractUpgrade:
				switch t.ChainID {
				case vaa.ChainIDSolana:
				// Already submitted to Solana.
				default:
					p.logger.Error("unsupported target chain for contract upgrade",
						zap.String("digest", hash),
						zap.Any("vaa", signed),
						zap.String("bytes", hex.EncodeToString(vaaBytes)),
						zap.Uint8("target_chain", t.ChainID))
				}
			default:
				panic(fmt.Sprintf("unknown VAA payload type: %+v", v))
			}

			p.state.vaaSignatures[hash].submitted = true
		} else {
			p.logger.Info("quorum not met or already submitted, doing nothing",
				zap.String("digest", hash))
		}
	} else {
		p.logger.Info("we have not yet seen this VAA - temporarily storing signature",
			zap.String("digest", hash),
			zap.Bools("aggregation", agg))

	}
}

// devnetVAASubmission submits VAA to a local Ethereum devnet. For production, the bridge won't
// have an Ethereum account and the user retrieves the VAA and submits the transactions themselves.
func (p *Processor) devnetVAASubmission(ctx context.Context, signed *vaa.VAA, hash string) {
	if p.devnetMode {
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		tx, err := devnet.SubmitVAA(timeout, p.devnetEthRPC, signed)
		cancel()
		if err != nil {
			if strings.Contains(err.Error(), "VAA was already executed") {
				p.logger.Info("lockup already submitted to Ethereum by another node, ignoring",
					zap.Error(err), zap.String("digest", hash))
			} else {
				p.logger.Error("failed to submit lockup to Ethereum",
					zap.Error(err), zap.String("digest", hash))
			}
			return
		}
		p.logger.Info("lockup submitted to Ethereum", zap.Any("tx", tx), zap.String("digest", hash))
	}
}

// Submit VAA to Terra.
func (p *Processor) terraVAASubmission(ctx context.Context, signed *vaa.VAA, hash string) {
	if !p.devnetMode || !p.terraEnabled {
		p.logger.Warn("ignoring terra VAA submission",
			zap.String("digest", hash))
		return
	}

	tx, err := terra.SubmitVAA(ctx, p.terraLCD, p.terraChaidID, p.terraContract, p.terraFeePayer, signed)
	if err != nil {
		if strings.Contains(err.Error(), "VaaAlreadyExecuted") {
			p.logger.Info("lockup already submitted to Terra by another node, ignoring",
				zap.Error(err), zap.String("digest", hash))
		} else {
			p.logger.Error("failed to submit lockup to Terra",
				zap.Error(err), zap.String("digest", hash))
		}
		return
	}
	p.logger.Info("lockup submitted to Terra", zap.Any("tx", tx), zap.String("digest", hash))
}
