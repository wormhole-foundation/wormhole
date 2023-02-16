package heartbeat

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"

	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var (
	p2pHeartbeatsSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_heartbeats_sent_total",
			Help: "Total number of p2p heartbeats sent",
		})

	heartbeatMessagePrefix = []byte("heartbeat|")

	// heartbeatMaxTimeDifference specifies the maximum time difference between the local clock and the timestamp in incoming heartbeat messages. Heartbeats that are this old or this much into the future will be dropped. This value should encompass clock skew and network delay.
	heartbeatMaxTimeDifference = time.Minute * 15
)

func HeartbeatSenderRunnable(
	nodeName string,
	features []string,
	gk *ecdsa.PrivateKey,
	io p2p.GossipSender,
	localPeerID peer.ID,
	gst *node_common.GuardianSetState,
	gov *governor.ChainGovernor,
	components *p2p.Components,
) supervisor.Runnable {
	bootTime := time.Now()
	ctr := int64(0)

	return func(ctx context.Context) error {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick.C:
				// create a heartbeat
				heartbeat := func() *gossipv1.Heartbeat {
					DefaultRegistry.mu.Lock()
					defer DefaultRegistry.mu.Unlock()
					networks := make([]*gossipv1.Heartbeat_Network, 0, len(DefaultRegistry.networkStats))
					for _, v := range DefaultRegistry.networkStats {
						errCtr := DefaultRegistry.GetErrorCount(vaa.ChainID(v.Id))
						v.ErrorCount = errCtr
						networks = append(networks, v)
					}

					heartbeat := &gossipv1.Heartbeat{
						NodeName:      nodeName,
						Counter:       ctr,
						Timestamp:     time.Now().UnixNano(),
						Networks:      networks,
						Version:       version.Version(),
						GuardianAddr:  ethcrypto.PubkeyToAddress(gk.PublicKey).String(),
						BootTimestamp: bootTime.UnixNano(),
						Features:      features,
					}

					if components.P2PIDInHeartbeat {
						var err error
						heartbeat.P2PNodeId, err = localPeerID.Marshal()
						if err != nil {
							panic(err)
						}
					}

					return heartbeat
				}()

				ourAddr := ethcrypto.PubkeyToAddress(gk.PublicKey)
				if err := gst.SetHeartbeat(ourAddr, localPeerID, heartbeat); err != nil {
					panic(err)
				}
				collectNodeMetrics(ourAddr, localPeerID, heartbeat)
				if gov != nil {
					gov.Heartbeat(heartbeat, io, gk, ourAddr)
				}

				err := io.Send(ctx, &gossipv1.GossipMessage{
					Message: &gossipv1.GossipMessage_SignedHeartbeat{
						SignedHeartbeat: createSignedHeartbeat(gk, heartbeat),
					},
				})
				if err != nil {
					return fmt.Errorf("failed to send heartbeat: %w", err)
				}

				p2pHeartbeatsSent.Inc()
				ctr += 1
			}
		}
	}
}

func HeartbeatProcessorRunnable(
	gst *node_common.GuardianSetState,
	disableVerify bool,
	io p2p.GossipReceiver,
	components *p2p.Components,
) supervisor.Runnable {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		heartbeatChan := make(chan *p2p.FilteredEnvelope[*gossipv1.GossipMessage_SignedHeartbeat], 50)
		err := p2p.SubscribeFilteredWithEnvelope(ctx, io, heartbeatChan)
		if err != nil {
			return fmt.Errorf("failed to subscribe to heartbeats: %w", err)
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case h := <-heartbeatChan:
				s := h.Message.SignedHeartbeat
				gs := gst.Get()
				if gs == nil {
					// No valid guardian set yet - dropping heartbeat
					logger.Debug("skipping heartbeat - no guardian set",
						zap.Any("value", s),
						zap.Stringer("from", h.From))
					break
				}
				if heartbeat, err := processSignedHeartbeat(h.From, s, gs, gst, disableVerify); err != nil {
					logger.Debug("invalid signed heartbeat received",
						zap.Error(err),
						zap.Any("payload", h.Message),
						zap.Any("value", s),
						zap.Stringer("from", h.From))
				} else {
					logger.Debug("valid signed heartbeat received",
						zap.Any("value", heartbeat),
						zap.Stringer("from", h.From))
					func() {
						if len(heartbeat.P2PNodeId) != 0 {
							components.ProtectedHostByGuardianKeyLock.Lock()
							defer components.ProtectedHostByGuardianKeyLock.Unlock()
							var peerId peer.ID
							if err = peerId.Unmarshal(heartbeat.P2PNodeId); err != nil {
								logger.Error("p2p_node_id_in_heartbeat_invalid",
									zap.Any("payload", h.Message),
									zap.Any("value", s),
									zap.Stringer("from", h.From))
							} else {
								guardianAddr := common.BytesToAddress(s.GuardianAddr)
								prevPeerId, ok := components.ProtectedHostByGuardianKey[guardianAddr]
								if ok {
									if prevPeerId != peerId {
										logger.Info("p2p_guardian_peer_changed",
											zap.String("guardian_addr", guardianAddr.String()),
											zap.String("prevPeerId", prevPeerId.String()),
											zap.String("newPeerId", peerId.String()),
										)
										components.ConnMgr.Unprotect(prevPeerId, "heartbeat")
										components.ConnMgr.Protect(peerId, "heartbeat")
										components.ProtectedHostByGuardianKey[guardianAddr] = peerId
									}
								} else {
									components.ConnMgr.Protect(peerId, "heartbeat")
									components.ProtectedHostByGuardianKey[guardianAddr] = peerId
								}
							}
						} else {
							logger.Debug("p2p_node_id_not_in_heartbeat",
								zap.Error(err),
								zap.Any("payload", heartbeat.NodeName))
						}
					}()
				}
			}
		}
	}
}

func createSignedHeartbeat(gk *ecdsa.PrivateKey, heartbeat *gossipv1.Heartbeat) *gossipv1.SignedHeartbeat {
	ourAddr := ethcrypto.PubkeyToAddress(gk.PublicKey)

	b, err := proto.Marshal(heartbeat)
	if err != nil {
		panic(err)
	}

	// Sign the heartbeat using our node's guardian key.
	digest := heartbeatDigest(b)
	sig, err := ethcrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		panic(err)
	}

	return &gossipv1.SignedHeartbeat{
		Heartbeat:    b,
		Signature:    sig,
		GuardianAddr: ourAddr.Bytes(),
	}
}

func processSignedHeartbeat(from peer.ID, s *gossipv1.SignedHeartbeat, gs *node_common.GuardianSet, gst *node_common.GuardianSetState, disableVerify bool) (*gossipv1.Heartbeat, error) {
	envelopeAddr := common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk common.Address
	if !ok {
		if !disableVerify {
			return nil, fmt.Errorf("invalid message: %s not in guardian set", envelopeAddr)
		}
	} else {
		pk = gs.Keys[idx]
	}

	digest := heartbeatDigest(s.Heartbeat)

	// SECURITY: see whitepapers/0009_guardian_key.md
	if len(heartbeatMessagePrefix)+len(s.Heartbeat) < 34 {
		return nil, fmt.Errorf("invalid message: too short")
	}

	pubKey, err := ethcrypto.Ecrecover(digest.Bytes(), s.Signature)
	if err != nil {
		return nil, errors.New("failed to recover public key")
	}

	signerAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	if pk != signerAddr && !disableVerify {
		return nil, fmt.Errorf("invalid signer: %v", signerAddr)
	}

	var h gossipv1.Heartbeat
	err = proto.Unmarshal(s.Heartbeat, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	if time.Until(time.Unix(0, h.Timestamp)).Abs() > heartbeatMaxTimeDifference {
		return nil, fmt.Errorf("heartbeat is too old or too far into the future")
	}

	if h.GuardianAddr != signerAddr.String() {
		return nil, fmt.Errorf("GuardianAddr in heartbeat does not match signerAddr")
	}

	// Store verified heartbeat in global guardian set state.
	if err := gst.SetHeartbeat(signerAddr, from, &h); err != nil {
		return nil, fmt.Errorf("failed to store in guardian set state: %w", err)
	}

	collectNodeMetrics(signerAddr, from, &h)

	return &h, nil
}

func heartbeatDigest(b []byte) common.Hash {
	return ethcrypto.Keccak256Hash(append(heartbeatMessagePrefix, b...))
}
