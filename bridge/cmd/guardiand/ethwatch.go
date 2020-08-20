package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
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
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

// aggregationState represents a single node's aggregation of guardian signatures.
type (
	lockupState struct {
		firstObserved time.Time
		ourVAA        *vaa.VAA
		signatures    map[ethcommon.Address][]byte
	}

	lockupMap map[string]*lockupState

	aggregationState struct {
		lockupSignatures lockupMap
	}
)

func ethLockupProcessor(lockC chan *common.ChainLock, setC chan *common.GuardianSet, gk *ecdsa.PrivateKey, sendC chan []byte, obsvC chan *gossipv1.EthLockupObservation, vaaC chan *vaa.VAA) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		our_addr := crypto.PubkeyToAddress(gk.PublicKey)
		state := &aggregationState{lockupMap{}}

		// Get initial validator set
		logger.Info("waiting for initial validator set to be fetched from Ethereum")
		gs := <-setC
		logger.Info("current guardian set received",
			zap.Strings("set", gs.KeysAsHexStrings()),
			zap.Uint32("index", gs.Index))

		if *unsafeDevMode {
			idx, err := devnet.GetDevnetIndex()
			if err != nil {
				return err
			}

			if idx == 0 && (uint(len(gs.Keys)) != *devNumGuardians) {
				vaa := devnet.DevnetGuardianSetVSS(*devNumGuardians)

				logger.Info(fmt.Sprintf("guardian set has %d members, expecting %d - submitting VAA",
					len(gs.Keys), *devNumGuardians),
					zap.Any("vaa", vaa))

				timeout, _ := context.WithTimeout(ctx, 15*time.Second)
				tx, err := devnet.SubmitVAA(timeout, *ethRPC, vaa)
				if err != nil {
					logger.Error("failed to submit devnet guardian set change", zap.Error(err))
				}

				logger.Info("devnet guardian set change submitted", zap.Any("tx", tx))
			}
		}

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
					zap.String("source", hex.EncodeToString(k.SourceAddress[:])),
					zap.String("target", hex.EncodeToString(k.TargetAddress[:])),
					zap.String("amount", k.Amount.String()),
					zap.String("txhash", k.TxHash.String()),
					zap.Time("timestamp", k.Timestamp),
				)

				us, ok := gs.KeyIndex(our_addr)
				if !ok {
					logger.Error("we're not in the guardian set - refusing to sign",
						zap.Uint32("index", gs.Index),
						zap.String("our_addr", our_addr.Hex()),
						zap.Any("set", gs.KeysAsHexStrings()))
					break
				}

				// All nodes will create the exact same VAA and sign its SHA256 digest.
				// Consensus is established on this digest.

				v := &vaa.VAA{
					Version:          vaa.SupportedVAAVersion,
					GuardianSetIndex: gs.Index,
					Signatures:       nil,
					Timestamp:        k.Timestamp,
					Payload: &vaa.BodyTransfer{
						Nonce:         0, // TODO
						SourceChain:   vaa.ChainIDEthereum,
						TargetChain:   vaa.ChainIDSolana,
						SourceAddress: k.SourceAddress,
						TargetAddress: k.TargetAddress,
						Asset: &vaa.AssetMeta{
							Chain:   vaa.ChainIDEthereum,
							Address: k.TokenAddress,
						},
						Amount: k.Amount,
					},
				}

				b, err := v.Marshal()
				if err != nil {
					panic(err)
				}

				h := sha256.Sum256(b)  // TODO: use SigningMsg?

				signData, err := v.SigningMsg()
				if err != nil {
					panic(err)
				}
				s, err := crypto.Sign(signData.Bytes(), gk)
				if err != nil {
					panic(err)
				}


				logger.Info("observed and signed confirmed lockup on Ethereum",
					zap.String("txhash", k.TxHash.String()),
					zap.String("vaahash", hex.EncodeToString(h[:])),
					zap.String("signature", hex.EncodeToString(s)),
					zap.Int("us", us))

				obsv := gossipv1.EthLockupObservation{
					Addr:      crypto.PubkeyToAddress(gk.PublicKey).Bytes(),
					Hash:      h[:],
					Signature: s,
				}

				w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_EthLockupObservation{EthLockupObservation: &obsv}}

				msg, err := proto.Marshal(&w)
				if err != nil {
					panic(err)
				}

				sendC <- msg

				// Store our VAA in case we're going to submit it to Solana
				hash := hex.EncodeToString(h[:])

				if state.lockupSignatures[hash] == nil {
					state.lockupSignatures[hash] = &lockupState{
						firstObserved: time.Now(),
						signatures:    map[ethcommon.Address][]byte{},
					}
					// TODO: do we receive and add our own signature below?
				}

				state.lockupSignatures[hash].ourVAA = v
			case m := <-obsvC:
				logger.Info("received eth lockup observation",
					zap.String("hash", hex.EncodeToString(m.Hash)),
					zap.Binary("signature", m.Signature),
					zap.Binary("addr", m.Addr))

				their_addr := ethcommon.BytesToAddress(m.Addr)
				_, ok := gs.KeyIndex(their_addr)
				if !ok {
					logger.Warn("received eth observation by unknown guardian - is our guardian set outdated?",
						zap.String("their_addr", their_addr.Hex()),
						zap.Any("current_set", gs.KeysAsHexStrings()),
					)
					break
				}

				// TODO: timeout/garbage collection for lockup state

				// []byte isn't hashable in a map. Paying a small extra cost to for encoding for easier debugging.
				hash := hex.EncodeToString(m.Hash)

				if state.lockupSignatures[hash] == nil {
					state.lockupSignatures[hash] = &lockupState{
						firstObserved: time.Now(),
						signatures:    map[ethcommon.Address][]byte{},
					}
				}

				state.lockupSignatures[hash].signatures[their_addr] = m.Signature

				// Enumerate guardian set and check for signatures
				agg := make([]bool, len(gs.Keys))
				var sigs []*vaa.Signature
				for i, a := range gs.Keys {
					// TODO: verify signature
					s, ok := state.lockupSignatures[hash].signatures[a]

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

				// Deep copy the VAA and add signatures
				v := state.lockupSignatures[hash].ourVAA
				signed := &vaa.VAA{
					Version:          v.Version,
					GuardianSetIndex: v.GuardianSetIndex,
					Signatures:       sigs,
					Timestamp:        v.Timestamp,
					Payload:          v.Payload,
				}

				// 2/3+ majority required for VAA to be valid - wait until we have quorum to submit VAA.
				quorum := int(math.Ceil((float64(len(gs.Keys)) / 3) * 2))

				logger.Info("aggregation state for eth lockup",
					zap.String("vaahash", hash),
					zap.Any("set", gs.KeysAsHexStrings()),
					zap.Uint32("index", gs.Index),
					zap.Bools("aggregation", agg),
					zap.Int("required_sigs", quorum),
					zap.Int("have_sigs", len(sigs)),
					)

				if *unsafeDevMode && len(sigs) >= quorum {
					_, err := devnet.GetDevnetIndex()
					if err != nil {
						return err
					}

					vaaBytes, err := signed.Marshal()
					if err != nil {
						panic(err)
					}

					logger.Info("submitting signed VAA to Solana",
						zap.String("vaahash", hash),
						zap.Any("vaa", signed),
						zap.Binary("bytes", vaaBytes))

					// TODO: actually submit to Solana once the agent works and has a reasonable key
					//if idx == 0 {
					//	vaaC <- state.lockupSignatures[hash].ourVAA
					//}
				} else if !*unsafeDevMode {
					panic("not implemented")  // TODO
				} else {
					logger.Info("quorum not met, doing nothing",
						zap.String("vaahash", hash))
				}
			}
		}
	}
}
