package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignedHeartbeat(t *testing.T) {

	type testCase struct {
		timestamp             int64
		guardianSigner        guardiansigner.GuardianSigner
		heartbeatGuardianAddr string
		fromP2pId             peer.ID
		p2pNodeId             []byte
		expectSuccess         bool
	}

	// define the tests

	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	gAddr := crypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background()))
	fromP2pId, err := peer.Decode("12D3KooWSgMXkhzTbKTeupHYmyG7sFJ5LpVreQcwVnX8RD7LBpy9")
	assert.NoError(t, err)
	p2pNodeId, err := fromP2pId.Marshal()
	assert.NoError(t, err)

	//guardianSigner2, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	//assert.NoError(t, err)
	//gAddr2 := crypto.PubkeyToAddress(guardianSigner2.PublicKey(context.Background()))
	fromP2pId2, err := peer.Decode("12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU")
	assert.NoError(t, err)
	p2pNodeId2, err := fromP2pId2.Marshal()
	assert.NoError(t, err)

	tests := []testCase{
		// happy case
		{
			timestamp:             time.Now().UnixNano(),
			guardianSigner:        guardianSigner,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         true,
		},
		// // guardian signed a heartbeat for another guardian
		// {
		// 	timestamp:             time.Now().UnixNano(),
		// 	guardianSigner:        guardianSigner,
		// 	heartbeatGuardianAddr: gAddr2.String(),
		// 	fromP2pId:             fromP2pId,
		// 	p2pNodeId:             p2pNodeId,
		// 	expectSuccess:         false,
		// },
		// old heartbeat
		{
			timestamp:             time.Now().Add(-time.Hour).UnixNano(),
			guardianSigner:        guardianSigner,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
		// heartbeat from the distant future
		{
			timestamp:             time.Now().Add(time.Hour).UnixNano(),
			guardianSigner:        guardianSigner,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
		// mismatched peer id
		{
			timestamp:             time.Now().UnixNano(),
			guardianSigner:        guardianSigner,
			heartbeatGuardianAddr: gAddr.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId2,
			expectSuccess:         false,
		},
	}
	// run the tests

	testFunc := func(t *testing.T, tc testCase) {

		addr := crypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background()))

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

		s := createSignedHeartbeat(context.Background(), guardianSigner, heartbeat)
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

func TestSignedObservation(t *testing.T) {

	type testCase struct {
		timestamp       int64
		guardianSigner  guardiansigner.GuardianSigner
		guardianAddress []byte
		expectSuccess   bool
	}

	// define the tests

	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	gAddr := crypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background()))
	assert.NoError(t, err)
	assert.NoError(t, err)

	guardianSigner2, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	//gAddr2 := crypto.PubkeyToAddress(guardianSigner2.PublicKey(context.Background()))
	assert.NoError(t, err)
	assert.NoError(t, err)

	tests := []testCase{
		// happy case
		{
			timestamp:       time.Now().UnixNano(),
			guardianSigner:  guardianSigner,
			guardianAddress: gAddr[:],
			expectSuccess:   true,
		},
		// Invalid key signed the data
		{
			timestamp:       time.Now().UnixNano(),
			guardianAddress: gAddr[:],
			guardianSigner:  guardianSigner2,
			expectSuccess:   false,
		},
		// Old timestamp request
		{
			timestamp:       time.Now().Add(-time.Hour).UnixNano(),
			guardianSigner:  guardianSigner,
			guardianAddress: gAddr[:],
			expectSuccess:   false,
		},
		// Old protobuf version handles (for now)
		{
			timestamp:       0,
			guardianSigner:  guardianSigner,
			guardianAddress: gAddr[:],
			expectSuccess:   true,
		},
	}
	// run the tests

	testFunc := func(t *testing.T, tc testCase) {

		req := &gossipv1.ObservationRequest{
			ChainId:   2,
			TxHash:    []byte{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9},
			Timestamp: tc.timestamp,
		}
		b, err := proto.Marshal(req)
		if err != nil {
			panic(err)
		}

		digest := signedObservationRequestDigest(b)
		sig, err := tc.guardianSigner.Sign(context.Background(), digest.Bytes())
		if err != nil {
			panic(err)
		}

		sReq := &gossipv1.SignedObservationRequest{
			ObservationRequest: b,
			Signature:          sig,
			GuardianAddr:       tc.guardianAddress,
		}

		gs := &node_common.GuardianSet{
			Keys:  []common.Address{gAddr},
			Index: 1,
		}

		result, err := processSignedObservationRequest(sReq, gs)

		if tc.expectSuccess {
			assert.NoError(t, err)
			assert.EqualValues(t, req.ChainId, result.ChainId)
			assert.EqualValues(t, req.TxHash, result.TxHash)
			assert.EqualValues(t, req.Timestamp, result.Timestamp)
		} else {
			assert.Error(t, err)
		}
	}

	for _, tc := range tests {
		testFunc(t, tc)
	}
}
