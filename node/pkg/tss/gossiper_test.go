package tss

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
)

func TestWitnessNewVaaV1(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)
	signer, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(privKey)
	require.NoError(t, err)

	gst := node_common.NewGuardianSetState(nil)
	gs := &node_common.GuardianSet{
		Keys:  []eth_common.Address{crypto.PubkeyToAddress(privKey.PublicKey)},
		Index: 1,
	}
	gst.Set(gs)

	s := &signerClient{
		vaaData: vaaHandling{
			isLeader:       true,
			gst:            gst,
			GuardianSigner: signer,
			gossipOutput:   make(chan *gossipv1.TSSGossipMessage, 1),
		},
	}

	v := &vaa.VAA{
		Version:          vaa.VaaVersion1,
		GuardianSetIndex: 1,
		Signatures:       []*vaa.Signature{},
		Timestamp:        time.Now(),
		Nonce:            1,
		Sequence:         1,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   vaa.Address{1},
		Payload:          []byte("payload"),
	}
	v.AddSignature(privKey, 0)

	ctx := context.Background()
	err = s.WitnessNewVaaV1(ctx, v)
	assert.NoError(t, err)

	select {
	case msg := <-s.vaaData.gossipOutput:
		assert.NotNil(t, msg)
		// Verify signature on gossip message
		pubKey, err := crypto.Ecrecover(crypto.Keccak256(msg.Message), msg.Signature)
		require.NoError(t, err)
		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		recoveredAddr := eth_common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])
		assert.Equal(t, addr, recoveredAddr)
	default:
		t.Fatal("expected message in gossipOutput")
	}
}

func TestWitnessNewVaaV1_NotLeader(t *testing.T) {
	s := &signerClient{
		vaaData: vaaHandling{
			isLeader: false,
		},
	}
	v := &vaa.VAA{}
	err := s.WitnessNewVaaV1(context.Background(), v)
	assert.NoError(t, err)
}

func TestInform(t *testing.T) {
	s := &signerClient{
		vaaData: vaaHandling{
			isLeader:       false,
			incomingGossip: make(chan *gossipv1.TSSGossipMessage, 1),
		},
	}
	msg := &gossipv1.TSSGossipMessage{}
	err := s.Inform(msg)
	assert.NoError(t, err)
	select {
	case m := <-s.vaaData.incomingGossip:
		assert.Equal(t, msg, m)
	default:
		t.Fatal("expected message in incomingGossip")
	}
}

func TestInform_Leader(t *testing.T) {
	s := &signerClient{
		vaaData: vaaHandling{
			isLeader: true,
		},
	}
	msg := &gossipv1.TSSGossipMessage{}
	err := s.Inform(msg)
	assert.NoError(t, err)
}

func TestGossipListener(t *testing.T) {
	// Setup keys
	leaderKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	// Setup Guardian Set
	gst := node_common.NewGuardianSetState(nil)
	gs := &node_common.GuardianSet{
		Keys:  []eth_common.Address{crypto.PubkeyToAddress(leaderKey.PublicKey)},
		Index: 1,
	}
	gst.Set(gs)

	// Setup Client
	s := &signerClient{
		conn: &connChans{
			signRequests: make(chan *signer.SignRequest, 1),
		},
		vaaData: vaaHandling{
			isLeader:       false,
			leaderIndex:    0,
			gst:            gst,
			incomingGossip: make(chan *gossipv1.TSSGossipMessage, 1),
		},
	}

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.gossipListener(ctx, zap.NewNop())

	// Create VAA
	v := &vaa.VAA{
		Version:          vaa.VaaVersion1,
		GuardianSetIndex: 1,
		Signatures:       []*vaa.Signature{},
		Timestamp:        time.Now(),
		Nonce:            1,
		Sequence:         1,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   vaa.Address{1},
		Payload:          []byte("payload"),
	}
	v.AddSignature(leaderKey, 0)
	vaaBytes, err := v.Marshal()
	require.NoError(t, err)

	// Create Gossip Message signed by leader
	hash := crypto.Keccak256(vaaBytes)
	sig, err := crypto.Sign(hash, leaderKey)
	require.NoError(t, err)

	gossipMsg := &gossipv1.TSSGossipMessage{
		Message:   vaaBytes,
		Signature: sig,
	}

	// Send message
	s.vaaData.incomingGossip <- gossipMsg

	// Verify SignRequest received
	select {
	case req := <-s.conn.signRequests:
		assert.Equal(t, v.SigningDigest().Bytes(), req.Digest)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for sign request")
	}
}
