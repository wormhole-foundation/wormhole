package processor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	// vaaState represents the local view of a given VAA
	vaaState struct {
		firstObserved time.Time
		ourVAA        *vaa.VAA
		signatures    map[ethcommon.Address][]byte
		submitted     bool
	}

	vaaMap map[string]*vaaState

	// aggregationState represents the node's aggregation of guardian signatures.
	aggregationState struct {
		vaaSignatures vaaMap
	}
)

type Processor struct {
	// lockC is a channel of observed chain lockups
	lockC chan *common.ChainLock
	// setC is a channel of guardian set updates
	setC chan *common.GuardianSet

	// sendC is a channel of outbound messages to broadcast on p2p
	sendC chan []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.LockupObservation

	// vaaC is a channel of VAAs to submit to store on Solana (either as target, or for data availability)
	vaaC chan *vaa.VAA

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	// devnetMode specified whether to submit transactions to the hardcoded Ethereum devnet
	devnetMode         bool
	devnetNumGuardians uint
	devnetEthRPC       string

	logger *zap.Logger

	// Runtime state

	// gs is the currently valid guardian set
	gs *common.GuardianSet
	// state is the current runtime VAA view
	state *aggregationState
}

func NewProcessor(
	ctx context.Context,
	lockC chan *common.ChainLock,
	setC chan *common.GuardianSet,
	sendC chan []byte,
	obsvC chan *gossipv1.LockupObservation,
	vaaC chan *vaa.VAA,
	gk *ecdsa.PrivateKey,
	devnetMode bool,
	devnetNumGuardians uint,
	devnetEthRPC string) *Processor {

	return &Processor{
		lockC:              lockC,
		setC:               setC,
		sendC:              sendC,
		obsvC:              obsvC,
		vaaC:               vaaC,
		gk:                 gk,
		devnetMode:         devnetMode,
		devnetNumGuardians: devnetNumGuardians,
		devnetEthRPC:       devnetEthRPC,

		logger: supervisor.Logger(ctx),
		state:  &aggregationState{vaaMap{}},
	}
}

func (p *Processor) Run(ctx context.Context) error {
	ourAddr := crypto.PubkeyToAddress(p.gk.PublicKey)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.gs = <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", p.gs.KeysAsHexStrings()),
				zap.Uint32("index", p.gs.Index))

			// Dev mode guardian set update check (no-op in production)
			err := p.checkDevModeGuardianSetUpdate(ctx)
			if err != nil {
				return err
			}
		case k := <-p.lockC:
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

			us, ok := p.gs.KeyIndex(ourAddr)
			if !ok {
				p.logger.Error("we're not in the guardian set - refusing to sign",
					zap.Uint32("index", p.gs.Index),
					zap.Stringer("our_addr", ourAddr),
					zap.Any("set", p.gs.KeysAsHexStrings()))
				break
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

			obsv := gossipv1.LockupObservation{
				Addr:      crypto.PubkeyToAddress(p.gk.PublicKey).Bytes(),
				Hash:      digest.Bytes(),
				Signature: s,
			}

			w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_LockupObservation{LockupObservation: &obsv}}

			msg, err := proto.Marshal(&w)
			if err != nil {
				panic(err)
			}

			p.sendC <- msg

			// Store our VAA in case we're going to submit it to Solana
			hash := hex.EncodeToString(digest.Bytes())

			if p.state.vaaSignatures[hash] == nil {
				p.state.vaaSignatures[hash] = &vaaState{
					firstObserved: time.Now(),
					signatures:    map[ethcommon.Address][]byte{},
				}
			}

			p.state.vaaSignatures[hash].ourVAA = v

			// Fast path for our own signature
			go func() { p.obsvC <- &obsv }()
		case m := <-p.obsvC:
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
				break
			}

			// Verify that m.Addr matches the public key that signed m.Hash.
			their_addr := ethcommon.BytesToAddress(m.Addr)
			signer_pk := ethcommon.BytesToAddress(crypto.Keccak256(pk[1:])[12:])

			if their_addr != signer_pk {
				p.logger.Info("invalid lockup observation - address does not match pubkey",
					zap.String("digest", hex.EncodeToString(m.Hash)),
					zap.String("signature", hex.EncodeToString(m.Signature)),
					zap.String("addr", hex.EncodeToString(m.Addr)),
					zap.String("pk", signer_pk.Hex()))
				break
			}

			// Verify that m.Addr is included in the current guardian set.
			_, ok := p.gs.KeyIndex(their_addr)
			if !ok {
				p.logger.Warn("received observation by unknown guardian - is our guardian set outdated?",
					zap.String("their_addr", their_addr.Hex()),
					zap.Any("current_set", p.gs.KeysAsHexStrings()),
				)
				break
			}

			// Hooray! Now, we have verified all fields on LockupObservation and know that it includes
			// a valid signature by an active guardian. We still don't fully trust them, as they may be
			// byzantine, but now we know who we're dealing with.

			// TODO: timeout/garbage collection for lockup state
			// TODO: rebroadcast signatures for VAAs that fail to reach consensus

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
					signatures:    map[ethcommon.Address][]byte{},
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
	}
}

func (p *Processor) checkDevModeGuardianSetUpdate(ctx context.Context) error {
	if p.devnetMode {
		if uint(len(p.gs.Keys)) != p.devnetNumGuardians {
			v := devnet.DevnetGuardianSetVSS(p.devnetNumGuardians)

			p.logger.Info(fmt.Sprintf("guardian set has %d members, expecting %d - submitting VAA",
				len(p.gs.Keys), p.devnetNumGuardians),
				zap.Any("v", v))

			timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			tx, err := devnet.SubmitVAA(timeout, p.devnetEthRPC, v)
			if err != nil {
				return fmt.Errorf("failed to submit devnet guardian set change: %v", err)
			}

			p.logger.Info("devnet guardian set change submitted to Ethereum", zap.Any("tx", tx), zap.Any("vaa", v))

			// Submit VAA to Solana as well. This is asynchronous and can fail, leading to inconsistent devnet state.
			p.vaaC <- v
		}
	}

	return nil
}
