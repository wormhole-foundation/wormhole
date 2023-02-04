package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestSignedHeartbeat(t *testing.T) {

	type testCase struct {
		timestamp             int64
		gk                    *ecdsa.PrivateKey
		heartbeatGuardianAddr string
		expectSuccess         bool
	}

	// define the tests

	gk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	assert.NoError(t, err)
	gAddr := ethcrypto.PubkeyToAddress(gk.PublicKey)

	gk2, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	assert.NoError(t, err)
	gAddr2 := ethcrypto.PubkeyToAddress(gk2.PublicKey)

	tests := []testCase{
		// happy case
		{
			timestamp:             time.Now().UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr.String(),
			expectSuccess:         true,
		},
		// guardian signed a heartbeat for another guardian
		{
			timestamp:             time.Now().UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr2.String(),
			expectSuccess:         false,
		},
		// old heartbeat
		{
			timestamp:             time.Now().Add(-time.Hour).UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr2.String(),
			expectSuccess:         false,
		},
		// heartbeat from the distant future
		{
			timestamp:             time.Now().Add(time.Hour).UnixNano(),
			gk:                    gk,
			heartbeatGuardianAddr: gAddr2.String(),
			expectSuccess:         false,
		},
	}
	// run the tests

	testFunc := func(t *testing.T, tc testCase) {

		addr := ethcrypto.PubkeyToAddress(gk.PublicKey)

		heartbeat := &gossipv1.Heartbeat{
			NodeName:      "someNode",
			Counter:       1,
			Timestamp:     tc.timestamp,
			Networks:      []*gossipv1.Heartbeat_Network{},
			Version:       "0.0.1beta",
			GuardianAddr:  tc.heartbeatGuardianAddr,
			BootTimestamp: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
			Features:      []string{},
		}

		s := createSignedHeartbeat(gk, heartbeat)
		gs := &node_common.GuardianSet{
			Keys:  []common.Address{addr},
			Index: 1,
		}

		gst := node_common.NewGuardianSetState(nil)

		heartbeatResult, err := processSignedHeartbeat("someone", s, gs, gst, false)

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
