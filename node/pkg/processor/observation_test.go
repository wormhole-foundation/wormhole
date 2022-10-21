package processor

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	marshalVAA, _ := vaa.Marshal()

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	ctx := context.Background()
	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(ctx, signedVAAWithQuorum)

	// Check to see if we got an error, which we should have,
	// because a `gs` is not defined on processor
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	errorString := "dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet"
	assert.Equal(t, errorString, firstLog.Message)
}

func TestHandleInboundSignedVAAWithQuorum(t *testing.T) {
	goodPrivateKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	goodAddr1 := crypto.PubkeyToAddress(goodPrivateKey1.PublicKey)
	badPrivateKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	tests := []struct {
		label      string
		keyOrder   []*ecdsa.PrivateKey
		indexOrder []uint8
		addrs      []ethcommon.Address
		errString  string
	}{
		{label: "GuardianSetNoKeys", keyOrder: []*ecdsa.PrivateKey{}, indexOrder: []uint8{}, addrs: []ethcommon.Address{},
			errString: "dropping SignedVAAWithQuorum message since we have a guardian set without keys"},
		{label: "VAANoSignatures", keyOrder: []*ecdsa.PrivateKey{}, indexOrder: []uint8{0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA was not signed"},
		{label: "VAAInvalidSignatures", keyOrder: []*ecdsa.PrivateKey{badPrivateKey1}, indexOrder: []uint8{0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
		{label: "DuplicateGoodSignaturesNonMonotonic", keyOrder: []*ecdsa.PrivateKey{goodPrivateKey1, goodPrivateKey1, goodPrivateKey1, goodPrivateKey1}, indexOrder: []uint8{0, 0, 0, 0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
		{label: "DuplicateGoodSignaturesMonotonic", keyOrder: []*ecdsa.PrivateKey{goodPrivateKey1, goodPrivateKey1, goodPrivateKey1, goodPrivateKey1}, indexOrder: []uint8{0, 1, 2, 3}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			vaa := getVAA()

			// Define a GuardianSet from test addrs
			guardianSet := common.GuardianSet{
				Keys:  tc.addrs,
				Index: 1,
			}

			// Sign with the keys at the proper index
			for i, key := range tc.keyOrder {
				vaa.AddSignature(key, tc.indexOrder[i])
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

			// Check to see if we got an error, which we should have
			assert.Equal(t, 1, observedLogs.Len())
			firstLog := observedLogs.All()[0]
			assert.Equal(t, tc.errString, firstLog.Message)
		})
	}
}
