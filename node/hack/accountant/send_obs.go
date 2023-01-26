// This tool can be used to send various observations to the accounting smart contract.
// It is meant for testing purposes only.

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	wormchainURL := string("localhost:9090")
	wormchainKeyPath := string("./dev.wormchain.key")
	contract := "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465"
	guardianKeyPath := string("./dev.guardian.key")

	wormchainKey, err := wormconn.LoadWormchainPrivKey(wormchainKeyPath, "test0000")
	if err != nil {
		logger.Fatal("failed to load devnet wormchain private key", zap.Error(err))
	}

	wormchainConn, err := wormconn.NewConn(ctx, wormchainURL, wormchainKey)
	if err != nil {
		logger.Fatal("failed to connect to wormchain", zap.Error(err))
	}

	logger.Info("Connected to wormchain",
		zap.String("wormchainURL", wormchainURL),
		zap.String("wormchainKeyPath", wormchainKeyPath),
		zap.String("senderAddress", wormchainConn.SenderAddress()),
	)

	logger.Info("Loading guardian key", zap.String("guardianKeyPath", guardianKeyPath))
	gk, err := loadGuardianKey(guardianKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	sequence := uint64(time.Now().Unix())
	timestamp := time.Now()

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", timestamp, sequence, true, false, "Submit should succeed") {
		return
	}

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", timestamp, sequence, true, false, "Already commited should succeed") {
		return
	}

	sequence += 10
	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c17", timestamp, sequence, false, true, "Bad emitter address should fail") {
		return
	}

	sequence += 10
	if !testBatch(ctx, logger, gk, wormchainConn, contract, timestamp, sequence) {
		return
	}

	sequence += 10
	if !testBatchWithcommitted(ctx, logger, gk, wormchainConn, contract, timestamp, sequence) {
		return
	}

	sequence += 10
	if !testBatchWithDigestError(ctx, logger, gk, wormchainConn, contract, timestamp, sequence) {
		return
	}

	logger.Info("Success! All tests passed!")
}

func testSubmit(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	emitterAddressStr string,
	timestamp time.Time,
	sequence uint64,
	expectedResult bool,
	errorExpected bool,
	tag string,
) bool {
	EmitterAddress, _ := vaa.StringToAddress(emitterAddressStr)
	TxHash, _ := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	Payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")

	msg := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        timestamp,
		Nonce:            uint32(0),
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}

	msgs := []*common.MessagePublication{&msg}
	txResp, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.String("test", tag), zap.Error(err))
		return false
	}

	responses, err := accountant.GetObservationResponses(txResp)
	if err != nil {
		logger.Error("acct: failed to get responses", zap.Error(err))
		return false
	}

	if len(responses) != len(msgs) {
		logger.Error("acct: number of responses does not match number of messages", zap.Int("numMsgs", len(msgs)), zap.Int("numResp", len(responses)), zap.Error(err))
		return false
	}

	msgId := msgs[0].MessageIDString()
	status, exists := responses[msgId]
	if !exists {
		logger.Info("test failed: did not receive an observation response for message", zap.String("test", tag), zap.String("msgId", msgId))
		return false
	}

	committed := status.Type == "committed"

	if committed != expectedResult {
		logger.Info("test failed", zap.String("test", tag), zap.Uint64("seqNo", sequence), zap.Bool("committed", committed),
			zap.String("response", wormchainConn.BroadcastTxResponseToString(txResp)))
		return false
	}

	logger.Info("test succeeded", zap.String("test", tag))
	return true
}

func testBatch(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	timestamp time.Time,
	sequence uint64,
) bool {
	EmitterAddress, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	TxHash, _ := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	Payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")

	nonce := uint32(123456)

	msgs := []*common.MessagePublication{}

	msg1 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        timestamp,
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg1)

	nonce = nonce + 1
	sequence = sequence + 1
	msg2 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        time.Now(),
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg2)

	txResp, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.Error(err))
		return false
	}

	responses, err := accountant.GetObservationResponses(txResp)
	if err != nil {
		logger.Error("acct: failed to get responses", zap.Error(err))
		return false
	}

	if len(responses) != len(msgs) {
		logger.Error("acct: number of responses does not match number of messages", zap.Int("numMsgs", len(msgs)), zap.Int("numResp", len(responses)), zap.Error(err))
		return false
	}

	for idx, msg := range msgs {
		msgId := msg.MessageIDString()
		status, exists := responses[msgId]
		if !exists {
			logger.Error("acct: did not receive an observation response for message", zap.Int("idx", idx), zap.String("msgId", msgId))
			return false
		}

		if status.Type != "committed" {
			logger.Error("acct: unexpected response on observation", zap.Int("idx", idx), zap.String("msgId", msgId), zap.String("status", status.Type), zap.String("text", status.Data))
			return false
		}
	}

	logger.Info("test of batch passed.")
	return true
}

func testBatchWithcommitted(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	timestamp time.Time,
	sequence uint64,
) bool {
	EmitterAddress, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	TxHash, _ := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	Payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")

	nonce := uint32(123456)

	msgs := []*common.MessagePublication{}

	logger.Info("submitting a single transfer that should work")
	msg1 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        timestamp,
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg1)

	_, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to submit initial observation that should work", zap.Error(err))
		return false
	}

	logger.Info("submitting a second batch where the second one has already been committed")
	msgs = msgs[:0]

	nonce = nonce + 1
	sequence = sequence + 1
	msg2 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        time.Now(),
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg2)

	// Same one we just committed.
	msg3 := msg1
	msgs = append(msgs, &msg3)

	txResp, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.Error(err))
		return false
	}

	responses, err := accountant.GetObservationResponses(txResp)
	if err != nil {
		logger.Error("acct: failed to get responses", zap.Error(err))
		return false
	}

	if len(responses) != len(msgs) {
		logger.Error("acct: number of responses does not match number of messages", zap.Int("numMsgs", len(msgs)), zap.Int("numResp", len(responses)), zap.Error(err))
		return false
	}

	for idx, msg := range msgs {
		msgId := msg.MessageIDString()
		status, exists := responses[msgId]
		if !exists {
			logger.Error("acct: did not receive an observation response for message", zap.Int("idx", idx), zap.String("msgId", msgId))
			return false
		}

		if status.Type != "committed" {
			logger.Error("acct: unexpected response on observation", zap.Int("idx", idx), zap.String("msgId", msgId), zap.String("status", status.Type), zap.String("text", status.Data))
			return false
		}
	}

	logger.Info("test of batch with already committed passed.")
	return true
}

func testBatchWithDigestError(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	timestamp time.Time,
	sequence uint64,
) bool {
	EmitterAddress, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	TxHash, _ := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	Payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")

	nonce := uint32(123456)

	msgs := []*common.MessagePublication{}

	logger.Info("submitting a single transfer that should work")
	msg1 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        timestamp,
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg1)

	_, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to submit initial observation that should work", zap.Error(err))
		return false
	}

	logger.Info("submitting a second batch where the second one has a digest error")
	msgs = msgs[:0]

	nonce = nonce + 1
	sequence = sequence + 1
	msg2 := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        time.Now(),
		Nonce:            nonce,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}
	msgs = append(msgs, &msg2)

	// Same key as the message we committed but change the digest.
	msg3 := msg1
	msg3.Nonce = msg3.Nonce + 1
	msgs = append(msgs, &msg3)

	txResp, err := submit(ctx, logger, gk, wormchainConn, contract, msgs)
	if err != nil {
		logger.Error("acct: failed to submit second observation that should work", zap.Error(err))
		return false
	}

	responses, err := accountant.GetObservationResponses(txResp)
	if err != nil {
		logger.Error("acct: failed to get responses", zap.Error(err))
		return false
	}

	if len(responses) != len(msgs) {
		logger.Error("acct: number of responses does not match number of messages", zap.Int("numMsgs", len(msgs)), zap.Int("numResp", len(responses)), zap.Error(err))
		return false
	}

	msgId := msgs[0].MessageIDString()
	status, exists := responses[msgId]
	if !exists {
		logger.Error("acct: did not receive an observation response for message 0", zap.String("msgId", msgId))
		return false
	}

	if status.Type != "committed" {
		logger.Error("acct: unexpected response on observation for message 0", zap.String("msgId", msgId), zap.String("status", status.Type), zap.String("text", status.Data))
		return false
	}

	msgId = msgs[1].MessageIDString()
	status, exists = responses[msgId]
	if !exists {
		logger.Error("acct: did not receive an observation response for message 1", zap.String("msgId", msgId))
		return false
	}

	if status.Type != "error" {
		logger.Error("acct: unexpected response on observation for message 1", zap.String("status", status.Type), zap.String("text", status.Data))
		return false
	}

	if status.Data != "digest mismatch for processed message" {
		logger.Error("acct: unexpected error text on observation for message 1", zap.String("text", status.Data))
		return false
	}

	logger.Info("test of batch with digest error passed.")
	return true
}

func submit(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	msgs []*common.MessagePublication,
) (*sdktx.BroadcastTxResponse, error) {
	gsIndex := uint32(0)
	guardianIndex := uint32(0)

	return accountant.SubmitObservationsToContract(ctx, logger, gk, gsIndex, guardianIndex, wormchainConn, contract, msgs)
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

// loadGuardianKey loads a serialized guardian key from disk.
func loadGuardianKey(filename string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != GuardianKeyArmoredBlock {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	gk, err := ethCrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	return gk, nil
}
