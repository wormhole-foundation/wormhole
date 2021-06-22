package publicrpc

import (
	"math/rand"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
)

// track the number of active connections
var (
	currentPublicHeartbeatStreamsOpen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_publicrpc_rawheartbeat_connections",
			Help: "Current number of clients consuming gRPC raw heartbeat streams",
		})
)

func init() {
	prometheus.MustRegister(currentPublicHeartbeatStreamsOpen)
}

// multiplexing to distribute heartbeat messages to all the open connections
type PublicRawHeartbeatConnections struct {
	mu     sync.RWMutex
	subs   map[int]chan<- *publicrpcv1.Heartbeat
	logger *zap.Logger
}

func HeartbeatStreamMultiplexer(logger *zap.Logger) *PublicRawHeartbeatConnections {
	ps := &PublicRawHeartbeatConnections{
		subs:   map[int]chan<- *publicrpcv1.Heartbeat{},
		logger: logger.Named("heartbeatmultiplexer"),
	}
	return ps
}

// getUniqueClientId loops to generate & test integers for existence as key of map. returns an int that is not a key in map.
func (ps *PublicRawHeartbeatConnections) getUniqueClientId() int {
	clientId := rand.Intn(1e6)
	found := false
	for found {
		clientId = rand.Intn(1e6)
		_, found = ps.subs[clientId]
	}
	return clientId
}

// subscribeHeartbeats adds a channel to the subscriber map, keyed by arbitary clientId
func (ps *PublicRawHeartbeatConnections) subscribeHeartbeats(ch chan *publicrpcv1.Heartbeat) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	clientId := ps.getUniqueClientId()
	ps.logger.Info("subscribeHeartbeats for client", zap.Int("client", clientId))
	ps.subs[clientId] = ch
	currentPublicHeartbeatStreamsOpen.Set(float64(len(ps.subs)))
	return clientId
}

// PublishHeartbeat sends a message to all channels in the subscription map
func (ps *PublicRawHeartbeatConnections) PublishHeartbeat(msg *publicrpcv1.Heartbeat) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for client, ch := range ps.subs {
		select {
		case ch <- msg:
			ps.logger.Debug("published message to client", zap.Int("client", client))
		default:
			ps.logger.Debug("buffer overrrun when attempting to publish message", zap.Int("client", client))
		}
	}
}

// unsubscribeHeartbeats removes the client's channel from the subscription map
func (ps *PublicRawHeartbeatConnections) unsubscribeHeartbeats(clientId int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.logger.Debug("unsubscribeHeartbeats for client", zap.Int("clientId", clientId))
	delete(ps.subs, clientId)
	currentPublicHeartbeatStreamsOpen.Set(float64(len(ps.subs)))
}
