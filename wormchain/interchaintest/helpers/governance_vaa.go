package helpers

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func MigrateContractVAA(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	contractAddr string,
	newCodeId uint64,
) {
	node := chain.GetFullNode()
	codeId := fmt.Sprintf("%d", newCodeId)

	// get the migration vaa from the cli- to do this we run the migration command without any vaa data
	stdout, _, err := node.Exec(ctx,
		node.NodeCommand("tx", "wormhole", "migrate", contractAddr, codeId, "{}"), nil)
	require.NoError(t, err)
	fmt.Printf("migrate contract stdout: '%s'\n", stdout)
	fmt.Printf("migrate contract string(stdout): '%s'\n", string(stdout))
	fmt.Printf("migrate contract string(stdout[:]): '%s'\n", string(stdout[:]))

	migrationVAA := stdout

	_, err = node.ExecTx(ctx, keyName, "wormhole", "migrate", contractAddr, codeId, "{}", string(migrationVAA[:len(migrationVAA)-1]), "--gas", "auto")
	require.NoError(t, err)

	fmt.Printf("migrate contract vaa: %s\n", stdout)

}

func SubmitContractUpgradeGovernanceVAA(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	upgradeChainID uint16,
	newContractAddr [32]byte,
	guardians *guardians.ValSet,
) {
	node := chain.GetFullNode()

	payload := vaa.BodyContractUpgrade{
		ChainID:     vaa.ChainID(upgradeChainID),
		NewContract: vaa.Address(newContractAddr),
	}
	fmt.Printf("body contract upgrade payload: %+v\n", payload)

	payloadBz, err := payload.Serialize()
	fmt.Printf("body contract upgrade payload bytes: %s\n", hex.EncodeToString(payloadBz))

	require.NoError(t, err)
	v := generateGovernanceVaa(0, guardians, payloadBz)
	fmt.Printf("governance vaa: %+v\n", v)
	vBz, err := v.Marshal()
	fmt.Printf("governance vaa bytes: %s\n", vBz)
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)
	fmt.Printf("governance vaa hex: %s\n", vHex)

	decodedVaaString, err := hex.DecodeString(vHex)
	fmt.Printf("decoded governance vaa bytes: %s\n", decodedVaaString)
	require.NoError(t, err)

	parsedVAA, err := ParseVAA(decodedVaaString)
	require.NoError(t, err)
	fmt.Printf("parsed governance vaa: %+v\n", parsedVAA)

	action := parsedVAA.Payload[32]
	fmt.Printf("action: %d\n", action)
	fmt.Printf("contract upgrade action %d", vaa.ActionContractUpgrade)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "execute-governance-vaa", vHex, "--gas", "auto")
	require.NoError(t, err)
}

func ParseVAA(data []byte) (*vaa.VAA, error) {
	v, err := vaa.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return v, nil
}
