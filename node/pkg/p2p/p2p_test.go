package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignedHeartbeat(t *testing.T) {

	type testCase struct {
		timestamp             int64
		gk                    *ecdsa.PrivateKey
		heartbeatGuardianAddr string
		fromP2pId             peer.ID
		p2pNodeId             []byte
		expectSuccess         bool
	}

	// define the tests

	gk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	assert.NoError(t, err)
	gAddr := crypto.PubkeyToAddress(gk.PublicKey)
	fromP2pId, err := peer.Decode("12D3KooWSgMXkhzTbKTeupHYmyG7sFJ5LpVreQcwVnX8RD7LBpy9")
	assert.NoError(t, err)
	p2pNodeId, err := fromP2pId.Marshal()
	assert.NoError(t, err)

	gk2, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	assert.NoError(t, err)
	gAddr2 := crypto.PubkeyToAddress(gk2.PublicKey)
	fromP2pId2, err := peer.Decode("12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU")
	assert.NoError(t, err)
	p2pNodeId2, err := fromP2pId2.Marshal()
	assert.NoError(t, err)

	tests := []testCase{
		// happy case
		{
			timestamp:             time.Now().UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         true,
		},
		// guardian signed a heartbeat for another guardian
		{
			timestamp:             time.Now().UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr2.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
		// old heartbeat
		{
			timestamp:             time.Now().Add(-time.Hour).UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
		// heartbeat from the distant future
		{
			timestamp:             time.Now().Add(time.Hour).UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
		// mismatched peer id
		{
			timestamp:             time.Now().UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId2,
			expectSuccess:         false,
		},
	}
	// run the tests

	testFunc := func(t *testing.T, tc testCase) {

		addr := crypto.PubkeyToAddress(gk.PublicKey)

		heartbeat := &gossipv1.Heartbeat{
			NodeName:      "someNode",
			Counter:       1,
			Timestamp:     tc.timestamp,
			Networks:      []*gossipv1.Heartbeat_Network{},
			Version:       "0.0.1beta",
			GuardianAddr:  tc.heartbeatGuardianAddr,
			BootTimestamp: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
			Features:      []string{},
			P2PNodeId:     tc.p2pNodeId,
		}

		s := createSignedHeartbeat(gk, heartbeat)
		gs := &node_common.GuardianSet{
			Keys:  []common.Address{addr},
			Index: 1,
		}

		gst := node_common.NewGuardianSetState(nil)

		heartbeatResult, err := processSignedHeartbeat(tc.fromP2pId, s, gs, gst, false)

		if tc.expectSuccess {
			assert.NoError(t, err)
			assert.EqualValues(t, heartbeat.GuardianAddr, heartbeatResult.GuardianAddr)
		} else {
			assert.Error(t, err)
		}
	}

	for _, tc := range tests {
		testFunc(t, tc)
	}
}
