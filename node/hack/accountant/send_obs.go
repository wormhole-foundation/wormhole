// This tool can be used to confirm that the CoinkGecko price query still works after the token list is updated.
// Usage: go run check_query.go

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

	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	// data, err := hex.DecodeString("C3AE4256EAA0BA6D01041585F63AE7CAA69D6D33")
	// if err != nil {
	// 	logger.Fatal("failed to hex decode string", zap.Error(err))
	// }

	// conv, err := bech32.ConvertBits(data, 8, 5, true)
	// if err != nil {
	// 	logger.Fatal("failed to convert bits", zap.Error(err))
	// }

	// encoded, err := bech32.Encode("wormhole", conv)
	// if err != nil {
	// 	logger.Fatal("bech32 encode failed", zap.Error(err))
	// }
	// logger.Info("encoded", zap.String("str", encoded))
	// return

	wormchainURL := string("localhost:9090")
	wormchainKeyPath := string("./dev.wormchain.key")
	contract := "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
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

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", timestamp, sequence, false, false, "Submit should succeed") {
		return
	}

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", timestamp, sequence, true, false, "Already commited should succeed") {
		return
	}

	sequence += 1
	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c17", timestamp, sequence, false, true, "Bad emitter address should fail") {
		return
	}
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
	gsIndex := uint32(0)
	guardianIndex := uint32(0)

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

	txResp, err := accountant.SubmitObservationToContract(ctx, logger, gk, gsIndex, guardianIndex, wormchainConn, contract, &msg)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.String("test", tag), zap.Error(err))
		return false
	}

	// out, err := wormchainConn.BroadcastTxResponseToString(txResp)
	// if err != nil {
	// 	logger.Error("acct: failed to parse broadcast response", zap.Error(err))
	// 	return false
	// }

	alreadyCommitted, err := accountant.CheckSubmitObservationResult(txResp)
	if err != nil {
		if !errorExpected {
			logger.Error("acct: unexpected error", zap.String("test", tag), zap.Error(err))
			return false
		}

		logger.Info("test succeeded, expected error returned", zap.String("test", tag), zap.Error(err))
		return true
	}
	if alreadyCommitted != expectedResult {
		out, err := wormchainConn.BroadcastTxResponseToString(txResp)
		if err != nil {
			logger.Error("acct: failed to parse broadcast response", zap.String("test", tag), zap.Error(err))
			return false
		}

		logger.Info("test failed", zap.String("test", tag), zap.Uint64("seqNo", sequence), zap.Bool("alreadyCommitted", alreadyCommitted), zap.String("response", out))
		return false
	}

	logger.Info("test succeeded", zap.String("test", tag))
	return true
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

/*
DEBUG: obs: {
  key: {
    emitter_chain: 2,
    emitter_address: 'AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=',
    sequence: 0
  },
  nonce: 0,
  payload: 'AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==',
  tx_hash: '82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b'
}
*/
