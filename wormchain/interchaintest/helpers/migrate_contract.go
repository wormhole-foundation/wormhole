package helpers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createWasmMigrationPayload(code_id uint64, contractAddr string, json_msg string) []byte {
	expected_hash := vaa.CreateMigrateCosmwasmContractHash(code_id, contractAddr, []byte(json_msg))

	var payload bytes.Buffer
	payload.Write(vaa.WasmdModule[:])
	payload.Write([]byte{byte(vaa.ActionMigrateContract)})
	binary.Write(&payload, binary.BigEndian, uint16(vaa.ChainIDWormchain))
	// custom payload
	payload.Write(expected_hash[:])
	return payload.Bytes()
}

func MigrateContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	contractAddr string,
	codeId string,
	message string,
	guardians *guardians.ValSet,
) error {

	node := chain.GetFullNode()

	code_id, err := strconv.ParseUint(codeId, 10, 64)
	require.NoError(t, err)
	payload := createWasmMigrationPayload(code_id, contractAddr, message)
	v := GenerateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "migrate", contractAddr, codeId, message, vHex, "--gas", "auto")
	return err
}
