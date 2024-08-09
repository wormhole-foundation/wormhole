package helpers

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/crypto/sha3"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createWasmStoreCodePayload(wasmBytes []byte) []byte {
	// governance message with sha3 of wasmBytes as the payload
	var hashWasm [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(wasmBytes)
	keccak.Sum(hashWasm[:0])

	gov_msg := types.NewGovernanceMessage(vaa.WasmdModule, byte(vaa.ActionStoreCode), uint16(vaa.ChainIDWormchain),
		hashWasm[:])
	return gov_msg.MarshalBinary()
}

func createContractUpgradePayload(payload vaa.BodyContractUpgrade) ([]byte, error) {
	var coreModule [32]byte
	copy(coreModule[:], vaa.CoreModule[:])

	marshalledPayload, err := payload.Serialize()
	if err != nil {
		return nil, err
	}

	gov_msg := types.NewGovernanceMessage(coreModule, byte(vaa.ActionContractUpgrade), uint16(vaa.ChainIDWormchain),
		marshalledPayload)

	return gov_msg.MarshalBinary(), nil
}

func createIbcReceiverUpdateChannelPayload(payload vaa.BodyIbcUpdateChannelChain) ([]byte, error) {
	marshalledPayload, err := payload.Serialize(vaa.IbcReceiverModuleStr)
	if err != nil {
		return nil, err
	}

	gov_msg := types.NewGovernanceMessage(vaa.IbcReceiverModule, byte(vaa.IbcReceiverActionUpdateChannelChain), uint16(vaa.ChainIDWormchain),
		marshalledPayload)

	return gov_msg.MarshalBinary(), nil
}

// func UpgradeCoreContract(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, keyName string, payload vaa.BodyContractUpgrade, guardians *guardians.ValSet) {
// 	node := chain.FullNodes[0]

// }

// wormchaind tx wormhole store [wasm file] [vaa-hex] [flags]
func StoreContract(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, keyName string, fileLoc string, guardians *guardians.ValSet) (codeId string) {
	node := chain.FullNodes[0]

	_, file := filepath.Split(fileLoc)
	err := node.CopyFile(ctx, fileLoc, file)
	require.NoError(t, err, fmt.Errorf("writing contract file to docker volume: %w", err))

	content, err := os.ReadFile(fileLoc)
	require.NoError(t, err)

	// gzip the wasm file
	if IsWasm(content) {
		content, err = GzipIt(content)
		require.NoError(t, err)
	}

	payload := createWasmStoreCodePayload(content)
	v := generateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "store", path.Join(node.HomeDir(), file), vHex, "--gas", "auto")
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, node.Chain)
	require.NoError(t, err)

	stdoutBz, _, err := node.ExecQuery(ctx, "wasm", "list-code", "--reverse")
	require.NoError(t, err)

	res := CodeInfosResponse{}
	err = json.Unmarshal(stdoutBz, &res)
	require.NoError(t, err)

	return res.CodeInfos[0].CodeID
}

// IsWasm checks if the file contents are of wasm binary
func IsWasm(input []byte) bool {
	wasmIdent := []byte("\x00\x61\x73\x6D")
	return bytes.Equal(input[:4], wasmIdent)
}

// GzipIt compresses the input ([]byte)
func GzipIt(input []byte) ([]byte, error) {
	// Create gzip writer.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(input)
	if err != nil {
		return nil, err
	}
	err = w.Close() // You must close this first to flush the bytes to the buffer.
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

type CodeInfo struct {
	CodeID string `json:"code_id"`
}
type CodeInfosResponse struct {
	CodeInfos []CodeInfo `json:"code_infos"`
}
