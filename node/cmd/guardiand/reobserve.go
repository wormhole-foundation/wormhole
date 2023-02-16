package guardiand

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Multiplex observation requests to the appropriate chain
func reobservationRequestsHandler(
	gst *node_common.GuardianSetState,
	receiver p2p.GossipReceiver,
	chainObsvReqC map[vaa.ChainID]chan *gossipv1.ObservationRequest,
	localObsvReqC <-chan *gossipv1.ObservationRequest,
) supervisor.Runnable {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		// Due to the automatic re-observation requests sent out by the processor we may end
		// up getting multiple requests to re-observe the same tx. Keep a cache of the
		// requests received in the last 11 minutes so that we don't end up repeatedly
		// re-observing the same transactions.
		type cachedRequest struct {
			chainId vaa.ChainID
			txHash  string
		}

		obsvReqProducer, obsvReqConsumer := p2p.MeteredBufferedChannelPair[*p2p.FilteredEnvelope[*gossipv1.GossipMessage_SignedObservationRequest]](ctx, 100, "reobervation_requests")
		err := p2p.SubscribeFilteredWithEnvelope(ctx, receiver, obsvReqProducer)
		if err != nil {
			return err
		}

		cache := make(map[cachedRequest]time.Time)

		processObservationRequest := func(req *gossipv1.ObservationRequest) {
			r := cachedRequest{
				chainId: vaa.ChainID(req.ChainId),
				txHash:  hex.EncodeToString(req.TxHash),
			}

			if _, ok := cache[r]; ok {
				// We've recently seen a re-observation request for this tx
				// so skip this one.
				logger.Info("skipping duplicate re-observation request",
					zap.Stringer("chain", r.chainId),
					zap.String("tx_hash", r.txHash),
				)
				return
			}

			if channel, ok := chainObsvReqC[r.chainId]; ok {
				select {
				case channel <- req:
					cache[r] = time.Now()

				default:
					logger.Warn("failed to send reobservation request to watcher",
						zap.Stringer("chain_id", r.chainId),
						zap.String("tx_hash", r.txHash))
				}
			} else {
				logger.Error("unknown chain ID for reobservation request",
					zap.Uint16("chain_id", uint16(r.chainId)),
					zap.String("tx_hash", r.txHash))
			}
		}

		ticker := time.NewTicker(7 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				for r, t := range cache {
					if time.Since(t) > 11*time.Minute {
						delete(cache, r)
					}
				}
			case m := <-obsvReqConsumer:
				s := m.Message.SignedObservationRequest
				gs := gst.Get()
				if gs == nil {
					logger.Debug("dropping SignedObservationRequest - no guardian set",
						zap.Any("value", s),
						zap.Stringer("from", m.From))
					break
				}
				req, err := processSignedObservationRequest(s, gs)
				if err != nil {
					// TODO metrics p2pMessagesReceived.WithLabelValues("invalid_signed_observation_request").Inc()
					logger.Debug("invalid signed observation request received",
						zap.Error(err),
						zap.Any("payload", m.Message),
						zap.Any("value", s),
						zap.Stringer("from", m.From))
				} else {
					logger.Info("valid signed observation request received",
						zap.Any("value", s),
						zap.Stringer("from", m.From))
				}
				processObservationRequest(req)
			case req := <-localObsvReqC:
				processObservationRequest(req)
			}
		}
	}
}

// Sign and publish reobservation request to the network
func reobservationsRequestSender(
	gk *ecdsa.PrivateKey,
	sender p2p.GossipSender,
	c <-chan *gossipv1.ObservationRequest,
) supervisor.Runnable {
	return func(ctx context.Context) error {
		for {
			logger := supervisor.Logger(ctx)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-c:
				b, err := proto.Marshal(msg)
				if err != nil {
					panic(err)
				}

				// Sign the observation request using our node's guardian key.
				digest := signedObservationRequestDigest(b)
				sig, err := ethcrypto.Sign(digest.Bytes(), gk)
				if err != nil {
					panic(err)
				}

				sReq := &gossipv1.SignedObservationRequest{
					ObservationRequest: b,
					Signature:          sig,
					GuardianAddr:       ethcrypto.PubkeyToAddress(gk.PublicKey).Bytes(),
				}

				envelope := &gossipv1.GossipMessage{
					Message: &gossipv1.GossipMessage_SignedObservationRequest{
						SignedObservationRequest: sReq}}

				err = sender.Send(ctx, envelope)
				if err != nil {
					logger.Error("failed to publish observation request", zap.Error(err))
				} else {
					logger.Info("published signed observation request", zap.Any("signed_observation_request", sReq))
				}
			}
		}
	}
}

var signedObservationRequestPrefix = []byte("signed_observation_request|")

func processSignedObservationRequest(s *gossipv1.SignedObservationRequest, gs *node_common.GuardianSet) (*gossipv1.ObservationRequest, error) {
	envelopeAddr := common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk common.Address
	if !ok {
		return nil, fmt.Errorf("invalid message: %s not in guardian set", envelopeAddr)
	} else {
		pk = gs.Keys[idx]
	}

	// SECURITY: see whitepapers/0009_guardian_key.md
	if len(signedObservationRequestPrefix)+len(s.ObservationRequest) < 34 {
		return nil, fmt.Errorf("invalid observation request: too short")
	}

	digest := signedObservationRequestDigest(s.ObservationRequest)

	pubKey, err := ethcrypto.Ecrecover(digest.Bytes(), s.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key")
	}

	signerAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	if pk != signerAddr {
		return nil, fmt.Errorf("invalid signer: %v", signerAddr)
	}

	var h gossipv1.ObservationRequest
	err = proto.Unmarshal(s.ObservationRequest, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal observation request: %w", err)
	}

	// TODO: implement per-guardian rate limiting

	return &h, nil
}

func signedObservationRequestDigest(b []byte) common.Hash {
	return ethcrypto.Keccak256Hash(append(signedObservationRequestPrefix, b...))
}
