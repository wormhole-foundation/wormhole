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

	guardianSigner2, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	gAddr2 := crypto.PubkeyToAddress(guardianSigner2.PublicKey(context.Background()))
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
		// guardian signed a heartbeat for another guardian
		{
			timestamp:             time.Now().UnixNano(),
			guardianSigner:        guardianSigner,
			heartbeatGuardianAddr: gAddr2.String(),
			fromP2pId:             fromP2pId,
			p2pNodeId:             p2pNodeId,
			expectSuccess:         false,
		},
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

	guardianSigner2, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)

	tests := []testCase{
		// happy case
		{
			timestamp:       time.Now().UnixNano(),
			guardianSigner:  guardianSigner,
			guardianAddress: gAddr[:],
			expectSuccess:   true,
		},
		// Timestamp must be initialized (non-zero)
		{
			timestamp:       0,
			guardianSigner:  guardianSigner,
			guardianAddress: gAddr[:],
			expectSuccess:   false,
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
			assert.NoError(t, err)
		}

		digest := signedObservationRequestDigest(b)
		sig, err := tc.guardianSigner.Sign(context.Background(), digest.Bytes())
		if err != nil {
			assert.NoError(t, err)
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

func TestValidateSignedObservationBatch(t *testing.T) {
	// Setup test guardians and peers
	guardianSigner1, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	guardian1Addr := crypto.PubkeyToAddress(guardianSigner1.PublicKey(context.Background()))

	guardianSigner2, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	assert.NoError(t, err)
	guardian2Addr := crypto.PubkeyToAddress(guardianSigner2.PublicKey(context.Background()))

	// Create distinct P2P peer IDs
	peer1, err := peer.Decode("12D3KooWSgMXkhzTbKTeupHYmyG7sFJ5LpVreQcwVnX8RD7LBpy9")
	assert.NoError(t, err)
	peer2, err := peer.Decode("12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU")
	assert.NoError(t, err)
	peer3, err := peer.Decode("12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw")
	assert.NoError(t, err)

	// Helper to create observation batch
	createBatch := func(guardianAddr common.Address, numObservations int) *gossipv1.SignedObservationBatch {
		observations := make([]*gossipv1.Observation, numObservations)
		for i := 0; i < numObservations; i++ {
			observations[i] = &gossipv1.Observation{
				Hash:      []byte{byte(i), 0, 0, 0},
				Signature: []byte{byte(i), 1, 1, 1},
				TxHash:    []byte{byte(i), 2, 2, 2},
				MessageId: "test-msg-id",
			}
		}
		return &gossipv1.SignedObservationBatch{
			Addr:         guardianAddr[:],
			Observations: observations,
		}
	}

	// Helper to register a heartbeat (establishes guardian -> peer mapping)
	registerHeartbeat := func(gst *node_common.GuardianSetState, guardianAddr common.Address, peerID peer.ID) {
		hb := &gossipv1.Heartbeat{
			NodeName:  "test-node",
			Counter:   1,
			Timestamp: time.Now().UnixNano(),
		}
		err := gst.SetHeartbeat(guardianAddr, peerID, hb)
		assert.NoError(t, err)
	}

	type testCase struct {
		name          string
		setupFunc     func() (*node_common.GuardianSet, *node_common.GuardianSetState)
		batch         *gossipv1.SignedObservationBatch
		fromPeer      peer.ID
		expectSuccess bool
		errorContains string
	}

	tests := []testCase{
		{
			name: "happy path - valid batch from registered peer",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				registerHeartbeat(gst, guardian1Addr, peer1)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer1,
			expectSuccess: true,
		},
		{
			name: "guardian not in current guardian set",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				// Guardian set only contains guardian1, but batch is from guardian2
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				registerHeartbeat(gst, guardian2Addr, peer2)
				return gs, gst
			},
			batch:         createBatch(guardian2Addr, 10),
			fromPeer:      peer2,
			expectSuccess: false,
			errorContains: "not in current guardian set",
		},
		{
			name: "no heartbeat received from guardian yet",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				// Don't register any heartbeat - guardian is in set but hasn't sent heartbeat
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer1,
			expectSuccess: false,
			errorContains: "no heartbeat received from guardian",
		},
		{
			name: "P2P peer mismatch",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				// Guardian1 registered with peer1, but batch comes from peer2
				registerHeartbeat(gst, guardian1Addr, peer1)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer2, // Different peer than registered
			expectSuccess: false,
			errorContains: "does not match known peers",
		},
		{
			name: "batch exceeds maximum size",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				registerHeartbeat(gst, guardian1Addr, peer1)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 5000), // Exceeds limit of 4000
			fromPeer:      peer1,
			expectSuccess: false,
			errorContains: "batch exceeds max size",
		},
		{
			name: "empty batch",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				registerHeartbeat(gst, guardian1Addr, peer1)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 0), // Empty
			fromPeer:      peer1,
			expectSuccess: false,
			errorContains: "empty observation batch",
		},
		{
			name: "guardian with multiple registered peers - first peer",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				// Same guardian registered from two different peers
				registerHeartbeat(gst, guardian1Addr, peer1)
				registerHeartbeat(gst, guardian1Addr, peer2)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer1, // First registered peer
			expectSuccess: true,
		},
		{
			name: "guardian with multiple registered peers - second peer",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				// Same guardian registered from two different peers
				registerHeartbeat(gst, guardian1Addr, peer1)
				registerHeartbeat(gst, guardian1Addr, peer2)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer2, // Second registered peer
			expectSuccess: true,
		},
		{
			name: "guardian with multiple registered peers - unregistered peer",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				// Guardian registered from peer1 and peer2, but batch comes from peer3
				registerHeartbeat(gst, guardian1Addr, peer1)
				registerHeartbeat(gst, guardian1Addr, peer2)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 10),
			fromPeer:      peer3, // NOT registered!
			expectSuccess: false,
			errorContains: "does not match known peers",
		},
		{
			name: "batch at exact size limit",
			setupFunc: func() (*node_common.GuardianSet, *node_common.GuardianSetState) {
				gs := node_common.NewGuardianSet([]common.Address{guardian1Addr}, 1)
				gst := node_common.NewGuardianSetState(nil)
				registerHeartbeat(gst, guardian1Addr, peer1)
				return gs, gst
			},
			batch:         createBatch(guardian1Addr, 4000), // Exactly at limit
			fromPeer:      peer1,
			expectSuccess: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs, gst := tc.setupFunc()

			err := validateSignedObservationBatch(tc.fromPeer, tc.batch, gs, gst)

			if tc.expectSuccess {
				assert.NoError(t, err, "expected batch to be accepted")
			} else {
				assert.Error(t, err, "expected batch to be rejected")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			}
		})
	}
}
