package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

// aggregationState represents a single node's aggregation of guardian signatures.
type (
	vaaState struct {
		firstObserved time.Time
		ourVAA        *vaa.VAA
		signatures    map[ethcommon.Address][]byte
	}

	vaaMap map[string]*vaaState

	aggregationState struct {
		vaaSignatures vaaMap
	}
)

func vaaConsensusProcessor(lockC chan *common.ChainLock, setC chan *common.GuardianSet, gk *ecdsa.PrivateKey, sendC chan []byte, obsvC chan *gossipv1.LockupObservation, vaaC chan *vaa.VAA) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		our_addr := crypto.PubkeyToAddress(gk.PublicKey)
		state := &aggregationState{vaaMap{}}

		// Get initial validator set from Ethereum. We could also fetch it from Solana,
		// because both sets are synchronized, we simply made an arbitrary decision to use Ethereum.

		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		idx, ethGs, err := ethereum.FetchCurrentGuardianSet(timeout, logger, *ethRPC, ethcommon.HexToAddress(*ethContract))
		if err != nil {
			return fmt.Errorf("failed requesting guardian set from Ethereum: %w", err)
		}

		gs := &common.GuardianSet{
			Keys:  ethGs.Keys,
			Index: idx,
		}

		// In debug mode, node 0 submits a VAA that configures the desired number of guardians to both chains.
		if *unsafeDevMode {
			idx, err := devnet.GetDevnetIndex()
			if err != nil {
				return err
			}

			if idx == 0 && (uint(len(gs.Keys)) != *devNumGuardians) {
				v := devnet.DevnetGuardianSetVSS(*devNumGuardians)

				logger.Info(fmt.Sprintf("guardian set has %d members, expecting %d - submitting VAA",
					len(gs.Keys), *devNumGuardians),
					zap.Any("v", v))

				timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
				defer cancel()
				tx, err := devnet.SubmitVAA(timeout, *ethRPC, v)
				if err != nil {
					logger.Error("failed to submit devnet guardian set change", zap.Error(err))
				}
				logger.Info("devnet guardian set change submitted to Ethereum", zap.Any("tx", tx))

				// TODO: submit to Solana
			}
		}

		supervisor.Signal(ctx, supervisor.SignalHealthy)

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case gs = <-setC:
				logger.Info("guardian set updated",
					zap.Strings("set", gs.KeysAsHexStrings()),
					zap.Uint32("index", gs.Index))
			case k := <-lockC:
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

				us, ok := gs.KeyIndex(our_addr)
				if !ok {
					logger.Error("we're not in the guardian set - refusing to sign",
						zap.Uint32("index", gs.Index),
						zap.Stringer("our_addr", our_addr),
						zap.Any("set", gs.KeysAsHexStrings()))
					break
				}

				// All nodes will create the exact same VAA and sign its digest.
				// Consensus is established on this digest.

				v := &vaa.VAA{
					Version:          vaa.SupportedVAAVersion,
					GuardianSetIndex: gs.Index,
					Signatures:       nil,
					Timestamp:        k.Timestamp,
					Payload: &vaa.BodyTransfer{
						Nonce:         k.Nonce,
						SourceChain:   k.SourceChain,
						TargetChain:   k.TargetChain,
						SourceAddress: k.SourceAddress,
						TargetAddress: k.TargetAddress,
						Asset: &vaa.AssetMeta{
							Chain:   k.TokenChain,
							Address: k.TokenAddress,
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
				s, err := crypto.Sign(digest.Bytes(), gk)
				if err != nil {
					panic(err)
				}

				logger.Info("observed and signed confirmed lockup",
					zap.Stringer("source_chain", k.SourceChain),
					zap.Stringer("target_chain", k.TargetChain),
					zap.Stringer("txhash", k.TxHash),
					zap.String("digest", hex.EncodeToString(digest.Bytes())),
					zap.String("signature", hex.EncodeToString(s)),
					zap.Int("our_index", us))

				obsv := gossipv1.LockupObservation{
					Addr:      crypto.PubkeyToAddress(gk.PublicKey).Bytes(),
					Hash:      digest.Bytes(),
					Signature: s,
				}

				w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_LockupObservation{LockupObservation: &obsv}}

				msg, err := proto.Marshal(&w)
				if err != nil {
					panic(err)
				}

				sendC <- msg

				// Store our VAA in case we're going to submit it to Solana
				hash := hex.EncodeToString(digest.Bytes())

				if state.vaaSignatures[hash] == nil {
					state.vaaSignatures[hash] = &vaaState{
						firstObserved: time.Now(),
						signatures:    map[ethcommon.Address][]byte{},
					}
				}

				state.vaaSignatures[hash].ourVAA = v
			case m := <-obsvC:
				logger.Info("received lockup observation",
					zap.String("digest", hex.EncodeToString(m.Hash)),
					zap.String("signature", hex.EncodeToString(m.Signature)),
					zap.String("addr", hex.EncodeToString(m.Addr)))

				their_addr := ethcommon.BytesToAddress(m.Addr)
				_, ok := gs.KeyIndex(their_addr)
				if !ok {
					logger.Warn("received observation by unknown guardian - is our guardian set outdated?",
						zap.String("their_addr", their_addr.Hex()),
						zap.Any("current_set", gs.KeysAsHexStrings()),
					)
					break
				}

				// TODO: timeout/garbage collection for lockup state
				// TODO: rebroadcast signatures for VAAs that fail to reach consensus

				// []byte isn't hashable in a map. Paying a small extra cost to for encoding for easier debugging.
				hash := hex.EncodeToString(m.Hash)

				if state.vaaSignatures[hash] == nil {
					state.vaaSignatures[hash] = &vaaState{
						firstObserved: time.Now(),
						signatures:    map[ethcommon.Address][]byte{},
					}
				}

				state.vaaSignatures[hash].signatures[their_addr] = m.Signature

				// Enumerate guardian set and check for signatures
				agg := make([]bool, len(gs.Keys))
				var sigs []*vaa.Signature
				for i, a := range gs.Keys {
					// TODO: verify signature
					s, ok := state.vaaSignatures[hash].signatures[a]

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

				if state.vaaSignatures[hash].ourVAA != nil {
					// We have seen it on chain!
					// Deep copy the VAA and add signatures
					v := state.vaaSignatures[hash].ourVAA
					signed := &vaa.VAA{
						Version:          v.Version,
						GuardianSetIndex: v.GuardianSetIndex,
						Signatures:       sigs,
						Timestamp:        v.Timestamp,
						Payload:          v.Payload,
					}

					// 2/3+ majority required for VAA to be valid - wait until we have quorum to submit VAA.
					quorum := int(math.Ceil((float64(len(gs.Keys)) / 3) * 2))

					logger.Info("aggregation state for VAA",
						zap.String("digest", hash),
						zap.Any("set", gs.KeysAsHexStrings()),
						zap.Uint32("index", gs.Index),
						zap.Bools("aggregation", agg),
						zap.Int("required_sigs", quorum),
						zap.Int("have_sigs", len(sigs)),
					)

					if *unsafeDevMode && len(sigs) >= quorum {
						idx, err := devnet.GetDevnetIndex()
						if err != nil {
							return err
						}

						vaaBytes, err := signed.Marshal()
						if err != nil {
							panic(err)
						}

						// TODO: proper leader selection

						if t, ok := v.Payload.(*vaa.BodyTransfer); ok {
							switch {
							case t.TargetChain == vaa.ChainIDSolana:
								logger.Info("submitting signed VAA to Solana",
									zap.String("digest", hash),
									zap.Any("vaa", signed),
									zap.String("bytes", hex.EncodeToString(vaaBytes)))

								if idx == 0 {
									vaaC <- signed
								}
							case t.TargetChain == vaa.ChainIDEthereum:
								timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
								tx, err := devnet.SubmitVAA(timeout, *ethRPC, signed)
								cancel()
								if err != nil {
									logger.Error("failed to submit lockup to Ethereum", zap.Error(err))
									break
								}
								logger.Info("lockup submitted to Ethereum", zap.Any("tx", tx))

								// cross-submit to Solana for data availability
								if idx == 0 {
									vaaC <- signed
								}
							default:
								logger.Error("we don't know how to submit this VAA",
									zap.String("digest", hash),
									zap.Any("vaa", signed),
									zap.String("bytes", hex.EncodeToString(vaaBytes)),
									zap.Stringer("target_chain", t.TargetChain))
							}
						} else {
							panic(fmt.Sprintf("unknown VAA payload type: %+v", v))
						}
					} else if !*unsafeDevMode {
						panic("not implemented") // TODO
					} else {
						logger.Info("quorum not met, doing nothing",
							zap.String("digest", hash))
					}
				}
			}
		}
	}
}
