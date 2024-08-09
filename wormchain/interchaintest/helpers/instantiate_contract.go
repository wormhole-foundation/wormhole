package helpers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createWasmInstantiatePayload(code_id uint64, label string, json_msg string) []byte {
	// governance message with sha3 of arguments to instantiate
	// - code_id (big endian)
	// - label
	// - json_msg
	expected_hash := vaa.CreateInstatiateCosmwasmContractHash(code_id, label, []byte(json_msg))

	var payload bytes.Buffer
	payload.Write(vaa.WasmdModule[:])
	payload.Write([]byte{byte(vaa.ActionInstantiateContract)})
	binary.Write(&payload, binary.BigEndian, uint16(vaa.ChainIDWormchain))
	// custom payload
	payload.Write(expected_hash[:])
	return payload.Bytes()
}

func InstantiateContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	codeId string,
	label string,
	message string,
	guardians *guardians.ValSet,
) (contract string) {

	node := chain.FullNodes[0]

	code_id, err := strconv.ParseUint(codeId, 10, 64)
	require.NoError(t, err)
	payload := createWasmInstantiatePayload(code_id, label, message)
	v := generateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "instantiate", label, codeId, message, vHex, "--gas", "auto")
	require.NoError(t, err)

	stdout, _, err := node.ExecQuery(ctx, "wasm", "list-contract-by-code", codeId)
	require.NoError(t, err)

	contactsRes := QueryContractResponse{}
	err = json.Unmarshal([]byte(stdout), &contactsRes)
	require.NoError(t, err)

	contractAddr := contactsRes.Contracts[len(contactsRes.Contracts)-1]
	return contractAddr
}

type QueryContractResponse struct {
	Contracts []string `json:"contracts"`
}
