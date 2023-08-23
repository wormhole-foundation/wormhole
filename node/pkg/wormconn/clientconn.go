package wormconn

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/btcsuite/btcutil/bech32"
	wormchain "github.com/wormhole-foundation/wormchain/app"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClienConn represents a connection to a wormhole-chain endpoint, encapsulating
// interactions with the chain.
//
// Once a connection is established, users must call ClientConn.Close to
// terminate the connection and free up resources.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer
// to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ClientConn struct {
	c             *grpc.ClientConn
	encCfg        EncodingConfig
	privateKey    cryptotypes.PrivKey
	senderAddress string
	chainId       string
	mutex         sync.Mutex // Protects the account / sequence number
}

// NewConn creates a new connection to the wormhole-chain instance at `target`.
func NewConn(ctx context.Context, target string, privateKey cryptotypes.PrivKey, chainId string) (*ClientConn, error) {
	c, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	encCfg := MakeEncodingConfig(wormchain.ModuleBasics)

	senderAddress, err := generateSenderAddress(privateKey)
	if err != nil {
		return nil, err
	}

	return &ClientConn{c: c, encCfg: encCfg, privateKey: privateKey, senderAddress: senderAddress, chainId: chainId}, nil
}

func (c *ClientConn) SenderAddress() string {
	return c.senderAddress
}

// Close terminates the connection and frees up resources.
func (c *ClientConn) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.c.Close()
}

func (c *ClientConn) BroadcastTxResponseToString(txResp *sdktx.BroadcastTxResponse) string {
	if txResp == nil {
		return "txResp is nil"
	}

	out, err := c.encCfg.Marshaler.MarshalJSON(txResp)
	if err != nil {
		panic(fmt.Sprintf("failed to format txResp: %s", err))
	}

	return string(out)
}

// generateSenderAddress creates the sender address from the private key. It is based on https://pkg.go.dev/github.com/btcsuite/btcutil/bech32#Encode
func generateSenderAddress(privateKey cryptotypes.PrivKey) (string, error) {
	data, err := hex.DecodeString(privateKey.PubKey().Address().String())
	if err != nil {
		return "", fmt.Errorf("failed to generate public key, failed to hex decode string: %w", err)
	}

	conv, err := bech32.ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key, failed to convert bits: %w", err)
	}

	encoded, err := bech32.Encode("wormhole", conv)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key, bech32 encode failed: %w", err)
	}

	return encoded, nil
}
