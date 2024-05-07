package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

var (
	ccqP2pMessagesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_ccqp2p_broadcast_messages_sent_total",
			Help: "Total number of ccq p2p pubsub broadcast messages sent",
		})
	ccqP2pMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_ccqp2p_broadcast_messages_received_total",
			Help: "Total number of ccq p2p pubsub broadcast messages received",
		}, []string{"type"})
)

type ccqP2p struct {
	logger *zap.Logger

	h             host.Host
	th_req        *pubsub.Topic
	th_resp       *pubsub.Topic
	sub           *pubsub.Subscription
	allowedPeers  map[string]struct{}
	p2pComponents *Components
}

func newCcqRunP2p(
	logger *zap.Logger,
	allowedPeersStr string,
	components *Components,
) *ccqP2p {
	l := logger.With(zap.String("component", "ccqp2p"))
	allowedPeers := make(map[string]struct{})
	for _, peerID := range strings.Split(allowedPeersStr, ",") {
		if peerID != "" {
			l.Info("will allow requests from peer", zap.String("peerID", peerID))
			allowedPeers[peerID] = struct{}{}
		}
	}

	return &ccqP2p{
		logger:        l,
		allowedPeers:  allowedPeers,
		p2pComponents: components,
	}
}

func (ccq *ccqP2p) run(
	ctx context.Context,
	priv crypto.PrivKey,
	gk *ecdsa.PrivateKey,
	p2pNetworkID string,
	bootstrapPeers string,
	port uint,
	signedQueryReqC chan<- *gossipv1.SignedQueryRequest,
	queryResponseReadC <-chan *query.QueryResponsePublication,
	errC chan error,
) error {
	networkID := p2pNetworkID + "/ccq"
	var err error

	components := DefaultComponents()
	if components == nil {
		return fmt.Errorf("components is not initialized")
	}
	components.Port = port

	// Pass the gossip advertize address through to NewHost() if it was defined
	components.GossipAdvertiseAddress = ccq.p2pComponents.GossipAdvertiseAddress

	ccq.h, err = NewHost(ccq.logger, ctx, networkID, bootstrapPeers, components, priv)
	if err != nil {
		return fmt.Errorf("failed to create p2p: %w", err)
	}

	// Build a map of bootstrap peers so we can always allow subscribe requests from them.
	bootstrapPeersMap := map[string]struct{}{}
	bootstrappers, _ := BootstrapAddrs(ccq.logger, bootstrapPeers, ccq.h.ID())
	for _, peer := range bootstrappers {
		bootstrapPeersMap[peer.ID.String()] = struct{}{}
	}

	topic_req := fmt.Sprintf("%s/%s", networkID, "ccq_req")
	topic_resp := fmt.Sprintf("%s/%s", networkID, "ccq_resp")

	ccq.logger.Info("Creating pubsub topics", zap.String("request_topic", topic_req), zap.String("response_topic", topic_resp))
	ps, err := pubsub.NewGossipSub(ctx, ccq.h,
		// We only want to accept subscribes from peers in the allow list.
		pubsub.WithPeerFilter(func(peerID peer.ID, topic string) bool {
			if len(ccq.allowedPeers) == 0 {
				return true
			}
			if _, found := ccq.allowedPeers[peerID.String()]; found {
				return true
			}
			ccq.p2pComponents.ProtectedHostByGuardianKeyLock.Lock()
			defer ccq.p2pComponents.ProtectedHostByGuardianKeyLock.Unlock()
			for _, guardianPeerID := range ccq.p2pComponents.ProtectedHostByGuardianKey {
				if peerID == guardianPeerID {
					return true
				}
			}
			if _, found := bootstrapPeersMap[peerID.String()]; found {
				return true
			}
			ccq.logger.Debug("Dropping subscribe attempt from unknown peer", zap.String("peerID", peerID.String()))
			return false
		}))
	if err != nil {
		return fmt.Errorf("failed to create new gossip sub for req: %w", err)
	}

	// We want to join and subscribe to the request topic. We will receive messages from there, but never write to it.
	ccq.th_req, err = ps.Join(topic_req)
	if err != nil {
		return fmt.Errorf("failed to join topic_req: %w", err)
	}

	// We only want to join the response topic. We will only write to it.
	ccq.th_resp, err = ps.Join(topic_resp)
	if err != nil {
		return fmt.Errorf("failed to join topic_resp: %w", err)
	}

	// We only want to accept messages from peers in the allow list.
	err = ps.RegisterTopicValidator(topic_req, func(ctx context.Context, from peer.ID, msg *pubsub.Message) bool {
		if len(ccq.allowedPeers) == 0 {
			return true
		}
		if _, found := ccq.allowedPeers[msg.GetFrom().String()]; found {
			return true
		}
		ccq.logger.Debug("Dropping message from unknown peer",
			zap.String("fromPeerID", from.String()),
			zap.String("msgPeerID", msg.ReceivedFrom.String()),
			zap.String("msgFrom", msg.GetFrom().String()))
		return false
	})
	if err != nil {
		return fmt.Errorf("failed to register message filter: %w", err)
	}

	// Increase the buffer size to prevent failed delivery to slower subscribers
	ccq.sub, err = ccq.th_req.Subscribe(pubsub.WithBufferSize(1024))
	if err != nil {
		return fmt.Errorf("failed to subscribe topic_req: %w", err)
	}

	common.StartRunnable(ctx, errC, false, "ccqp2p_listener", func(ctx context.Context) error {
		return ccq.listener(ctx, signedQueryReqC)
	})

	common.StartRunnable(ctx, errC, false, "ccqp2p_publisher", func(ctx context.Context) error {
		return ccq.publisher(ctx, gk, queryResponseReadC)
	})

	ccq.logger.Info("Node has been started", zap.String("peer_id", ccq.h.ID().String()), zap.String("addrs", fmt.Sprintf("%v", ccq.h.Addrs())))
	return nil
}

func (ccq *ccqP2p) close() {
	ccq.logger.Info("entering close")

	if err := ccq.th_req.Close(); err != nil && !errors.Is(err, context.Canceled) {
		ccq.logger.Error("Error closing the topic_req", zap.Error(err))
	}
	if err := ccq.th_resp.Close(); err != nil && !errors.Is(err, context.Canceled) {
		ccq.logger.Error("Error closing the topic_req", zap.Error(err))
	}

	ccq.sub.Cancel()

	if err := ccq.h.Close(); err != nil {
		ccq.logger.Error("error closing the host", zap.Error(err))
	}
}

func (ccq *ccqP2p) listener(ctx context.Context, signedQueryReqC chan<- *gossipv1.SignedQueryRequest) error {
	for {
		envelope, err := ccq.sub.Next(ctx) // Note: sub.Next(ctx) will return an error once ctx is canceled
		if err != nil {
			return fmt.Errorf("failed to receive pubsub message: %w", err)
		}

		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			ccq.logger.Info("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
			ccqP2pMessagesReceived.WithLabelValues("invalid").Inc()
			continue
		}

		ccq.logger.Info("received message", //TODO: Change to Debug
			zap.Any("payload", msg.Message),
			zap.Binary("raw", envelope.Data),
			zap.String("from", envelope.GetFrom().String()))

		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryRequest:
			if err := query.PostSignedQueryRequest(signedQueryReqC, m.SignedQueryRequest); err != nil {
				ccq.logger.Warn("failed to handle query request", zap.Error(err))
			}
		default:
			ccqP2pMessagesReceived.WithLabelValues("unknown").Inc()
			ccq.logger.Warn("received unknown message type (running outdated software?)",
				zap.Any("payload", msg.Message),
				zap.Binary("raw", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
		}
	}
}

func (ccq *ccqP2p) publisher(ctx context.Context, gk *ecdsa.PrivateKey, queryResponseReadC <-chan *query.QueryResponsePublication) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-queryResponseReadC:
			msgBytes, err := msg.Marshal()
			if err != nil {
				ccq.logger.Error("failed to marshal query response", zap.Error(err))
				continue
			}
			digest := query.GetQueryResponseDigestFromBytes(msgBytes)
			sig, err := ethcrypto.Sign(digest.Bytes(), gk)
			if err != nil {
				panic(err)
			}
			envelope := &gossipv1.GossipMessage{
				Message: &gossipv1.GossipMessage_SignedQueryResponse{
					SignedQueryResponse: &gossipv1.SignedQueryResponse{
						QueryResponse: msgBytes,
						Signature:     sig,
					},
				},
			}
			b, err := proto.Marshal(envelope)
			if err != nil {
				panic(err)
			}
			err = ccq.th_resp.Publish(ctx, b)
			if err != nil {
				ccq.logger.Error("failed to publish query response",
					zap.String("requestSignature", msg.Signature()),
					zap.Any("query_response", msg),
					zap.Any("signature", sig),
					zap.Error(err),
				)
			} else {
				ccqP2pMessagesSent.Inc()
				ccq.logger.Info("published signed query response", //TODO: Change to Debug
					zap.String("requestSignature", msg.Signature()),
					zap.Any("query_response", msg),
					zap.Any("signature", sig),
				)
			}
		}
	}
}
