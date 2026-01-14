package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	p2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Test helpers from watermark_test.go
const LOCAL_P2P_PORTRANGE_START_TSS = 15000

type mockTssGossiper struct {
	messages chan *gossipv1.TSSGossipMessage
	mu       sync.Mutex
}

func newMockTssGossiper() *mockTssGossiper {
	return &mockTssGossiper{
		messages: make(chan *gossipv1.TSSGossipMessage, 10),
	}
}

func (m *mockTssGossiper) Inform(msg *gossipv1.TSSGossipMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages <- msg
	return nil
}

func (m *mockTssGossiper) Outbound() <-chan *gossipv1.TSSGossipMessage {
	return nil
}

func (m *mockTssGossiper) getMessage(t *testing.T) *gossipv1.TSSGossipMessage {
	select {
	case msg := <-m.messages:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for tss message")
		return nil
	}
}

func (m *mockTssGossiper) assertNoMessage(t *testing.T) {
	select {
	case msg := <-m.messages:
		t.Fatalf("received unexpected tss message: %v", msg)
	case <-time.After(2 * time.Second):
		// OK
	}
}

func TestTssGossipListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a bootstrap node (g0) and a listener node (g1)
	var guardianset = &common.GuardianSet{}
	var gs [2]*G
	for i := range gs {

		gs[i] = NewG(t, fmt.Sprintf("n%d", i))
		gs[i].components.Port = uint(LOCAL_P2P_PORTRANGE_START_TSS + i) // #nosec G115 -- This is safe as the inputs are constants
		gs[i].networkID = "/wormhole/localdev"
		guardianset.Keys = append(guardianset.Keys, crypto.PubkeyToAddress(gs[i].guardianSigner.PublicKey(ctx)))
		// id, err := p2ppeer.IDFromPublicKey(gs[i].priv.GetPublic())
		// require.NoError(t, err)
		// protectedPeers = append(protectedPeers, id.String()) // Protect all nodes

		// Set bootstrap to the first node

		gs[i].components.ConnMgr, _ = connmgr.NewConnManager(2, 3, connmgr.WithGracePeriod(2*time.Second))

	}

	id0, err := p2ppeer.IDFromPublicKey(gs[0].priv.GetPublic())
	require.NoError(t, err)
	bootstrapPeer := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic/p2p/%s", LOCAL_P2P_PORTRANGE_START_TSS, id0.String())

	for i := range gs {
		gs[i].bootstrapPeers = bootstrapPeer
		gs[i].gst.Set(guardianset)
	}

	// g0 is just a bootstrap node, no tss gossiper
	startGuardian(t, ctx, gs[0], []string{})

	// g1 is the listener with a mock tss gossiper
	mockGossiper := newMockTssGossiper()
	gs[1].tssGossiper = mockGossiper
	startGuardian(t, ctx, gs[1], []string{})

	// Wait for nodes to connect
	time.Sleep(5 * time.Second)

	// Create a publisher host
	p2ppriv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	publisher, err := NewHost(zap.NewNop(), ctx, gs[0].networkID, bootstrapPeer, DefaultComponents(), p2ppriv)
	require.NoError(t, err)
	defer publisher.Close()

	// Connect publisher to bootstrap
	bootstrapMultiAddr, err := multiaddr.NewMultiaddr(bootstrapPeer)
	require.NoError(t, err)
	bootstrapAddrInfo, err := peer.AddrInfoFromP2pAddr(bootstrapMultiAddr)
	require.NoError(t, err)
	err = publisher.Connect(ctx, *bootstrapAddrInfo)
	require.NoError(t, err)

	ps, err := pubsub.NewGossipSub(ctx, publisher)
	require.NoError(t, err)

	tssTopicName := fmt.Sprintf("%s/%s", gs[0].networkID, "tss")
	topic, err := ps.Join(tssTopicName)
	require.NoError(t, err)
	// Give pubsub time to propagate subscriptions
	time.Sleep(2 * time.Second)

	t.Run("ValidMessage", func(t *testing.T) {
		tssMsg := &gossipv1.TSSGossipMessage{
			Message:      []byte("hello"),
			Signature:    []byte("sig"),
			GuardianAddr: []byte("addr"),
		}
		gossipMsg := &gossipv1.GossipMessage{
			Message: &gossipv1.GossipMessage_TssGossipMessage{
				TssGossipMessage: tssMsg,
			},
		}
		msgBytes, err := proto.Marshal(gossipMsg)
		require.NoError(t, err)

		err = topic.Publish(ctx, msgBytes)
		require.NoError(t, err)

		received := mockGossiper.getMessage(t)
		assert.True(t, proto.Equal(tssMsg, received))
	})

	t.Run("InvalidMessage", func(t *testing.T) {
		err = topic.Publish(ctx, []byte("this is not a protobuf"))
		require.NoError(t, err)
		mockGossiper.assertNoMessage(t)
	})

	t.Run("UnknownMessageType", func(t *testing.T) {
		gossipMsg := &gossipv1.GossipMessage{
			Message: &gossipv1.GossipMessage_SignedHeartbeat{
				SignedHeartbeat: &gossipv1.SignedHeartbeat{},
			},
		}
		msgBytes, err := proto.Marshal(gossipMsg)
		require.NoError(t, err)

		err = topic.Publish(ctx, msgBytes)
		require.NoError(t, err)
		mockGossiper.assertNoMessage(t)
	})
}
