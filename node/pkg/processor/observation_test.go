package processor

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"testing"
	"time"
)

func getVAA() vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	vaa := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	return vaa
}

func TestHandleInboundSignedVAAWithQuorum_NilGuardianSet(t *testing.T) {
	vaa := getVAA()
	marshalVAA, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	ctx := context.Background()
	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(ctx, signedVAAWithQuorum)

	// Check to see if we got an errro, which we should have,
	// because a `gs` is not defined on processor
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	expected_error := "dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet"
	assert.Equal(t, expected_error, firstLog.Message)
}

func TestHandleInboundSignedVAAWithQuorum_BadSigner(t *testing.T) {
	vaa := getVAA()

	// Define some good/bad keys/addrs to build from
	goodGuardianPrivKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	badGuardianPrivKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	goodGuardianAddr := crypto.PubkeyToAddress(goodGuardianPrivKey.PublicKey)

	// Define a good GuardianSet
	guardianSet := common.GuardianSet{
		Keys: []ethcommon.Address{
			goodGuardianAddr,
		},
		Index: 1,
	}

	// Sign with a bad key
	vaa.AddSignature(badGuardianPrivKey, 1)
	marshalVAA, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	ctx := context.Background()
	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.gs = &guardianSet
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(ctx, signedVAAWithQuorum)

	// Check to see if we got an error, which we should have,
	// because the VAA is signed by a bad signer
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	expected_error := "received SignedVAAWithQuorum message with invalid VAA signatures"
	assert.Equal(t, expected_error, firstLog.Message)
}

func TestHandleInboundSignedVAAWithQuorum_GuardianSetNoKeysNoSignature(t *testing.T) {
	vaa := getVAA()

	badGuardianPrivKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	// Define a good GuardianSet
	guardianSet := common.GuardianSet{
		Keys:  []ethcommon.Address{},
		Index: 1,
	}

	// Sign with a bad key
	vaa.AddSignature(badGuardianPrivKey, 1)
	marshalVAA, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	ctx := context.Background()
	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.gs = &guardianSet
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(ctx, signedVAAWithQuorum)

	// Check to see if we got an error, which we should have,
	// because the VAA is not signed and we don't have any GuardianKeys
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	expected_error := "received SignedVAAWithQuorum message with invalid VAA signatures"
	assert.Equal(t, expected_error, firstLog.Message)
}

func TestHandleInboundSignedVAAWithQuorum_GuardianSetNoKeysBadSignature(t *testing.T) {
	vaa := getVAA()

	// Define a good GuardianSet
	guardianSet := common.GuardianSet{
		Keys:  []ethcommon.Address{},
		Index: 1,
	}

	marshalVAA, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	ctx := context.Background()
	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.gs = &guardianSet
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(ctx, signedVAAWithQuorum)

	// Check to see if we got an error, which we should have,
	// because the VAA is signed by a bad signer and we don't have any GuardianKeys
	// This means VerifySignature failed because signer was invalid
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	expected_error := "received SignedVAAWithQuorum message without quorum"
	assert.Equal(t, expected_error, firstLog.Message)
}
