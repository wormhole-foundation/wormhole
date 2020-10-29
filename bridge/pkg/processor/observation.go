package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

func (p *Processor) handleObservation(ctx context.Context, m *gossipv1.LockupObservation) {
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

	// Hooray! Now, we have verified all fields on LockupObservation and know that it includes
	// a valid signature by an active guardian. We still don't fully trust them, as they may be
	// byzantine, but now we know who we're dealing with.

	// []byte isn't hashable in a map. Paying a small extra cost for encoding for easier debugging.
	hash := hex.EncodeToString(m.Hash)

	if p.state.vaaSignatures[hash] == nil {
		// We haven't yet seen this lockup ourselves, and therefore do not know what the VAA looks like.
		// However, we have established that a valid guardian has signed it, and therefore we can
		// already start aggregating signatures for it.
		//
		// TODO: a malicious guardian can DoS this by creating fake lockups
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

			if t, ok := v.Payload.(*vaa.BodyTransfer); ok {
				switch {
				case t.TargetChain == vaa.ChainIDEthereum:
					// In dev mode, submit VAA to Ethereum. For production, the bridge won't
					// have an Ethereum account and the user retrieves the VAA and submits the transactions themselves.
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
							break
						}
						p.logger.Info("lockup submitted to Ethereum", zap.Any("tx", tx), zap.String("digest", hash))
					}

					// Cross-submit to Solana for data availability
					fallthrough
				case t.TargetChain == vaa.ChainIDSolana:
					p.logger.Info("submitting signed VAA to Solana",
						zap.String("digest", hash),
						zap.Any("vaa", signed),
						zap.String("bytes", hex.EncodeToString(vaaBytes)))

					p.vaaC <- signed
				default:
					p.logger.Error("we don't know how to submit this VAA",
						zap.String("digest", hash),
						zap.Any("vaa", signed),
						zap.String("bytes", hex.EncodeToString(vaaBytes)),
						zap.Stringer("target_chain", t.TargetChain))
				}

				p.state.vaaSignatures[hash].submitted = true
			} else {
				panic(fmt.Sprintf("unknown VAA payload type: %+v", v))
			}
		} else {
			p.logger.Info("quorum not met, doing nothing",
				zap.String("digest", hash))
		}
	}
}
