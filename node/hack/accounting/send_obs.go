// This tool can be used to confirm that the CoinkGecko price query still works after the token list is updated.
// Usage: go run check_query.go

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/certusone/wormhole/node/pkg/accounting"
	"github.com/certusone/wormhole/node/pkg/devnet"
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

	wormchainRPC := string("localhost:9090")
	wormchainKeyPath := string("./dev.wormchain.key")
	contract := "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
	guardianKeyPath := string("./dev.guardian.key")

	wormchainKey, err := devnet.LoadWormchainPrivKey(wormchainKeyPath)
	if err != nil {
		logger.Fatal("failed to load devnet wormchain private key", zap.Error(err))
	}

	logger.Info("Connecting to wormchain.", zap.String("wormchainRPC", wormchainRPC))
	wormchainConn, err := wormconn.NewConn(ctx, wormchainRPC, wormchainKey)
	if err != nil {
		logger.Fatal("failed to connect to wormchain", zap.Error(err))
	}

	logger.Info("Connected to wormchain", zap.String("publicKey", wormchainConn.PublicKey()))

	logger.Info("Loading guardian key")
	gk, err := loadGuardianKey(guardianKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}
	logger.Info("Loaded guardian key")

	EmitterChain := uint16(2)
	emitterAddress, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	EmitterAddress := base64.StdEncoding.EncodeToString(emitterAddress.Bytes())
	Sequence := uint64(0)
	Nonce := uint32(0)
	TxHash := "82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b"
	payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")
	Payload := base64.StdEncoding.EncodeToString(payload)
	gsIndex := uint32(0)

	logger.Info("DEBUG", zap.Uint16("EmitterChain", EmitterChain))
	logger.Info("DEBUG", zap.String("EmitterAddress", EmitterAddress))
	logger.Info("DEBUG", zap.Uint64("Sequence", Sequence))
	logger.Info("DEBUG", zap.Uint32("Nonce", Nonce))
	logger.Info("DEBUG", zap.String("Payload", Payload))
	logger.Info("DEBUG", zap.String("TxHash", TxHash))
	obs := []accounting.Observation{
		accounting.Observation{
			Key:     accounting.TransferKey{EmitterChain: uint16(EmitterChain), EmitterAddress: EmitterAddress, Sequence: Sequence},
			Nonce:   Nonce,
			TxHash:  TxHash,
			Payload: Payload,
		},
	}

	err = accounting.SubmitObservationToContract(ctx, logger, gk, gsIndex, wormchainConn, contract, obs)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.Error(err))
		return
	}

	logger.Info("Sent observation request to wormchain")
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
