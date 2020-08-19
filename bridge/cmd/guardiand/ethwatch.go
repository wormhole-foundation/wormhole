package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
)

// aggregationState represents a single node's aggregation of guardian signatures.
type (
	lockupState struct {
		firstObserved time.Time
		signatures    map[ethcommon.Address][]byte
	}

	lockupMap map[string]*lockupState

	aggregationState struct {
		lockupSignatures lockupMap
	}
)

func ethLockupProcessor(lockC chan *common.ChainLock, setC chan *common.GuardianSet, gk *ecdsa.PrivateKey, sendC chan []byte, obsvC chan *gossipv1.EthLockupObservation) func(ctx context.Context) error {
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
					zap.String("lockhash", hex.EncodeToString(k.Hash())),
				)

				us, ok := gs.KeyIndex(our_addr)
				if !ok {
					logger.Error("we're not in the guardian set - refusing to sign",
						zap.Uint32("index", gs.Index),
						zap.String("our_addr", our_addr.Hex()),
						zap.Any("set", gs.KeysAsHexStrings()))
					break
				}

				s, err := gk.Sign(rand.Reader, k.Hash(), nil)
				if err != nil {
					panic(err)
				}

				logger.Info("observed and signed confirmed lockup on Ethereum",
					zap.String("txhash", k.TxHash.String()),
					zap.String("lockhash", hex.EncodeToString(k.Hash())),
					zap.String("signature", hex.EncodeToString(s)),
					zap.Int("us", us))

				obsv := gossipv1.EthLockupObservation{
					Addr:      crypto.PubkeyToAddress(gk.PublicKey).Bytes(),
					Hash:      k.Hash(),
					Signature: s,
				}

				w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_EthLockupObservation{EthLockupObservation: &obsv}}

				msg, err := proto.Marshal(&w)
				if err != nil {
					panic(err)
				}

				sendC <- msg
			case m := <-obsvC:
				logger.Info("received another guardian's eth lockup observation",
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
				for i, a := range gs.Keys {
					// TODO: verify signature
					_, ok := state.lockupSignatures[hash].signatures[a]
					agg[i] = ok
				}

				logger.Info("aggregation state for eth lockup",
					zap.String("hash", hash),
					zap.Any("set", gs.KeysAsHexStrings()),
					zap.Uint32("index", gs.Index),
					zap.Bools("aggregation", agg))

				// TODO: submit to Solana
			}
		}
	}
}
