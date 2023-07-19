package p2p

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/certusone/wormhole/node/pkg/accountant"
	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	p2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"go.uber.org/zap"
)

const LOCAL_P2P_PORTRANGE_START = 11000

type G struct {
	// arguments passed to p2p.New
	obsvC                  chan *gossipv1.SignedObservation
	obsvReqC               chan *gossipv1.ObservationRequest
	obsvReqSendC           chan *gossipv1.ObservationRequest
	sendC                  chan []byte
	signedInC              chan *gossipv1.SignedVAAWithQuorum
	priv                   p2pcrypto.PrivKey
	gk                     *ecdsa.PrivateKey
	gst                    *node_common.GuardianSetState
	networkID              string
	bootstrapPeers         string
	nodeName               string
	disableHeartbeatVerify bool
	rootCtxCancel          context.CancelFunc
	gov                    *governor.ChainGovernor
	acct                   *accountant.Accountant
	signedGovCfg           chan *gossipv1.SignedChainGovernorConfig
	signedGovSt            chan *gossipv1.SignedChainGovernorStatus
	components             *Components
}

func NewG(t *testing.T, nodeName string) *G {
	t.Helper()

	cs := 20
	p2ppriv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	if err != nil {
		panic(err)
	}

	guardianpriv, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	g := &G{
		obsvC:                  make(chan *gossipv1.SignedObservation, cs),
		obsvReqC:               make(chan *gossipv1.ObservationRequest, cs),
		obsvReqSendC:           make(chan *gossipv1.ObservationRequest, cs),
		sendC:                  make(chan []byte, cs),
		signedInC:              make(chan *gossipv1.SignedVAAWithQuorum, cs),
		priv:                   p2ppriv,
		gk:                     guardianpriv,
		gst:                    node_common.NewGuardianSetState(nil),
		nodeName:               nodeName,
		disableHeartbeatVerify: false,
		rootCtxCancel:          nil,
		gov:                    nil,
		signedGovCfg:           make(chan *gossipv1.SignedChainGovernorConfig, cs),
		signedGovSt:            make(chan *gossipv1.SignedChainGovernorStatus, cs),
		components:             DefaultComponents(),
	}

	// Consume all output channels
	go func() {
		name := g.nodeName
		t.Logf("[%s] consuming\n", name)
		select {
		case <-g.obsvC:
		case <-g.obsvReqC:
		case <-g.signedInC:
		case <-g.signedGovCfg:
		case <-g.signedGovSt:
		case <-g.sendC:
		}
	}()

	return g
}

// TestWatermark runs 4 different guardians one of which does not send its P2PID in the signed part of the heartbeat.
// The expectation is that hosts that send this information will become "protected" by the Connection Manager.
func TestWatermark(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create 4 nodes
	var guardianset = &node_common.GuardianSet{}
	var gs [4]*G
	for i := range gs {
		gs[i] = NewG(t, fmt.Sprintf("n%d", i))
		gs[i].components.Port = uint(LOCAL_P2P_PORTRANGE_START + i)
		gs[i].networkID = "/wormhole/localdev"

		guardianset.Keys = append(guardianset.Keys, crypto.PubkeyToAddress(gs[i].gk.PublicKey))

		id, err := p2ppeer.IDFromPublicKey(gs[0].priv.GetPublic())
		require.NoError(t, err)

		gs[i].bootstrapPeers = fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic/p2p/%s", LOCAL_P2P_PORTRANGE_START, id.String())
		gs[i].gst.Set(guardianset)

		gs[i].components.ConnMgr, _ = connmgr.NewConnManager(2, 3, connmgr.WithGracePeriod(2*time.Second))
	}

	// The 4th guardian does not put its libp2p key in the heartbeat
	gs[3].components.P2PIDInHeartbeat = false

	// Start the nodes
	for _, g := range gs {
		startGuardian(t, ctx, g)
	}

	// Wait ~20s to let the nodes gossip.
	time.Sleep(20 * time.Second)

	// It's expected to have the 3 first nodes protected on every node
	for guardianIndex, guardian := range gs {

		// expectedProtectedPeers is expected to be 2 for all nodes except the last one where 3 is expected
		func() {
			guardian.components.ProtectedHostByGuardianKeyLock.Lock()
			defer guardian.components.ProtectedHostByGuardianKeyLock.Unlock()
			expectedProtectedPeers := 2
			if guardianIndex == 3 {
				expectedProtectedPeers = 3
			}
			assert.Equal(t, expectedProtectedPeers, len(guardian.components.ProtectedHostByGuardianKey))
		}()

		// check that nodes {0, 1, 2} are protected on all other nodes and that nodes {3} are not protected.
		for otherGuardianIndex, otherGuardian := range gs {
			g1addr, err := p2ppeer.IDFromPublicKey(otherGuardian.priv.GetPublic())
			require.NoError(t, err)
			isProtected := guardian.components.ConnMgr.IsProtected(g1addr, "heartbeat")

			// A node cannot be protected on itself as one's own heartbeats are dropped
			if guardianIndex == otherGuardianIndex {
				continue
			}
			assert.Falsef(t, isProtected && otherGuardianIndex == 3, "node at index 3 should not be protected on node %d but was", guardianIndex)
			assert.Falsef(t, !isProtected && otherGuardianIndex != 3, "node at index %d should be protected on node %d but is not", otherGuardianIndex, guardianIndex)
		}
	}
}

func startGuardian(t *testing.T, ctx context.Context, g *G) {
	t.Helper()
	supervisor.New(ctx, zap.L(),
		Run(g.obsvC,
			g.obsvReqC,
			g.obsvReqSendC,
			g.sendC,
			g.signedInC,
			g.priv,
			g.gk,
			g.gst,
			g.networkID,
			g.bootstrapPeers,
			g.nodeName,
			g.disableHeartbeatVerify,
			g.rootCtxCancel,
			g.acct,
			g.gov,
			g.signedGovCfg,
			g.signedGovSt,
			g.components,
			nil, // ibc feature string
			nil, // signed query request channel
		))
}
