//go:build integration

package solana

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// TestCloseEventReobservation tests reobservation of a close_posted_message event on Solana devnet.
// The close_posted_message instruction closes an old message account and emits its data as a CPI event,
// allowing guardians to reobserve messages even after the original account has been closed.
// Run with: go test -tags=integration -run TestCloseEventReobservation ./pkg/watchers/solana/
func TestCloseEventReobservation(t *testing.T) {
	// Devnet RPC URL
	const devnetRPC = "https://api.devnet.solana.com"

	// Devnet Wormhole Core Bridge address
	const rawContract = "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5"
	contractAddress, err := solana.PublicKeyFromBase58(rawContract)
	require.NoError(t, err)

	// Transaction signature to reobserve
	// https://explorer.solana.com/tx/29zzwD7aRvyMhUBDLtqJUJR6PcFcFciRmhRwFzSqoXdPXMork1AG9wVdgYGUa7BVr2svNWC65fMp9Uia4Cy4zMXU?cluster=devnet
	const txSignature = "29zzwD7aRvyMhUBDLtqJUJR6PcFcFciRmhRwFzSqoXdPXMork1AG9wVdgYGUa7BVr2svNWC65fMp9Uia4Cy4zMXU"
	signature, err := solana.SignatureFromBase58(txSignature)
	require.NoError(t, err)

	// Expected message values
	expectedTimestamp := time.Unix(1776161416, 0)
	expectedNonce := uint32(24)
	expectedSequence := uint64(23)
	expectedConsistencyLevel := uint8(32)
	expectedPayload, err := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000742bf979105179e44aed27baf37d66ef73cc3d88")
	require.NoError(t, err)
	// Single keccak256 hash of the VAA body (what Wormhole Explorer shows)
	expectedBodyHash := "ee897440b3aed4de69068689e2b385d25793f9658ba179bc680363af4db78ba2"
	// Double keccak256 hash (signing digest used for guardian signatures)
	// This can be confirmed by calling parseAndVerifyVM on the Ethereum Core Bridge
	// https://etherscan.io/address/0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B#readProxyContract
	// with the VAA hex: 0x01000000000100b7e9e4a3336afbffee50e2b6cf3eea2b0ac110a01d083c35d5e5d4d8ee53ed36073017c80c4174c14fad263d9c84a3a5edd9b73360fcc48b90ed698626be1be30069de128800000018000176752c408f4314dafbafc0766d39847940292e7cbb0c57d60c8ddd929183757b0000000000000017200100000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000742bf979105179e44aed27baf37d66ef73cc3d88
	expectedSigningDigest := "d6458c37243d2cd1463a113174af01077f885b1a43bb31daea14b9cbc5b57eec"

	// Convert base58 emitter address to vaa.Address
	emitterPubkey, err := solana.PublicKeyFromBase58("8yQjiwvWM6BYeQ2wrYZgqtrPPDYr3bJVAqC23NSpacev")
	require.NoError(t, err)
	var expectedEmitterAddress vaa.Address
	copy(expectedEmitterAddress[:], emitterPubkey[:])

	// Create a message channel to receive observations
	msgC := make(chan *common.MessagePublication, 10)

	// Create logger
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	// Create the watcher with devnet configuration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s := &SolanaWatcher{
		ctx:         ctx,
		logger:      logger,
		contract:    contractAddress,
		rawContract: rawContract,
		rpcUrl:      devnetRPC,
		rpcClient:   rpc.New(devnetRPC),
		commitment:  rpc.CommitmentFinalized,
		chainID:     vaa.ChainIDSolana,
		msgC:        msgC,
	}

	// Perform reobservation using the Reobserve method
	numObservations, err := s.Reobserve(ctx, vaa.ChainIDSolana, signature[:], devnetRPC)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), numObservations, "expected exactly one observation")

	// Receive the message from the channel
	select {
	case msg := <-msgC:
		// Verify all message fields
		assert.Equal(t, expectedTimestamp, msg.Timestamp, "timestamp mismatch")
		assert.Equal(t, expectedNonce, msg.Nonce, "nonce mismatch")
		assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain, "emitter chain mismatch")
		assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress, "emitter address mismatch")
		assert.Equal(t, expectedSequence, msg.Sequence, "sequence mismatch")
		assert.Equal(t, expectedConsistencyLevel, msg.ConsistencyLevel, "consistency level mismatch")
		assert.Equal(t, expectedPayload, msg.Payload, "payload mismatch")
		assert.True(t, msg.IsReobservation, "expected IsReobservation to be true")
		assert.Equal(t, signature[:], msg.TxID, "TxID should be the Solana transaction signature")
		assert.False(t, msg.Unreliable, "expected Unreliable to be false for reliable message prefix")
		assert.Equal(t, common.NotVerified, msg.VerificationState(), "expected VerificationState to be NotVerified")

		// Verify both hash formats
		v := msg.CreateVAA(0)
		vaaBytes, err := v.Marshal()
		require.NoError(t, err)
		// Body starts after header: 1 byte version + 4 bytes guardian set index + 1 byte sig count = 6 bytes
		body := vaaBytes[6:]
		actualBodyHash := hex.EncodeToString(crypto.Keccak256(body))
		actualSigningDigest := v.HexDigest()

		assert.Equal(t, expectedBodyHash, actualBodyHash, "VAA body hash (single keccak) mismatch")
		assert.Equal(t, expectedSigningDigest, actualSigningDigest, "signing digest (double keccak) mismatch")

		// Log success details
		t.Logf("Successfully reobserved transaction %s", txSignature)
		t.Logf("EmitterAddress: %s", hex.EncodeToString(msg.EmitterAddress[:]))
		t.Logf("Sequence: %d", msg.Sequence)
		t.Logf("Body Hash (single keccak): %s", actualBodyHash)
		t.Logf("Signing Digest (double keccak): %s", actualSigningDigest)

	case <-ctx.Done():
		t.Fatal("timeout waiting for message publication")
	}
}
