package ictest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Params is the slashing params response from the chain
type Params struct {
	SignedBlocksWindow      string `json:"signed_blocks_window"`
	MinSignedPerWindow      string `json:"min_signed_per_window"`
	DowntimeJailDuration    string `json:"downtime_jail_duration"`
	SlashFractionDoubleSign string `json:"slash_fraction_double_sign"`
	SlashFractionDowntime   string `json:"slash_fraction_downtime"`
}

// createSlashingParamsUpdate creates a slashing params update VAA
// that can be executed on chain via the governance module
func createSlashingParamsUpdate(
	signedBlocksWindow uint64,
	minSignedPerWindow string,
	downtimeJailDurationSeconds uint64,
	slashFractionDoubleSign string,
	slashFractionDowntime string,
) ([]byte, error) {
	minSignedWindow, err := sdk.NewDecFromStr(minSignedPerWindow)
	if err != nil {
		return nil, err
	}

	downtimeJailDurationSeconds = downtimeJailDurationSeconds * uint64(time.Second)

	slashFractionDoubleSignDec, err := sdk.NewDecFromStr(slashFractionDoubleSign)
	if err != nil {
		return nil, err
	}

	slashFractionDowntimeDec, err := sdk.NewDecFromStr(slashFractionDowntime)
	if err != nil {
		return nil, err
	}

	payloadBody := vaa.BodyGatewaySlashingParamsUpdate{
		SignedBlocksWindow:      signedBlocksWindow,
		MinSignedPerWindow:      minSignedWindow.BigInt().Uint64(),
		DowntimeJailDuration:    downtimeJailDurationSeconds,
		SlashFractionDoubleSign: slashFractionDoubleSignDec.BigInt().Uint64(),
		SlashFractionDowntime:   slashFractionDowntimeDec.BigInt().Uint64(),
	}

	return payloadBody.Serialize()
}

// querySlashingParams queries the slashing params from the chain
func querySlashingParams(ctx context.Context, wormchain *cosmos.CosmosChain) (params slashingtypes.Params, err error) {
	// query the slashing params
	res, _, err := wormchain.FullNodes[0].ExecQuery(ctx, "slashing", "params")
	if err != nil {
		return
	}

	var slashingParams Params
	err = json.Unmarshal(res, &slashingParams)
	if err != nil {
		return
	}

	params.SignedBlocksWindow, err = strconv.ParseInt(slashingParams.SignedBlocksWindow, 10, 64)
	if err != nil {
		return
	}

	params.MinSignedPerWindow, err = sdk.NewDecFromStr(slashingParams.MinSignedPerWindow)
	if err != nil {
		return
	}

	params.DowntimeJailDuration, err = time.ParseDuration(slashingParams.DowntimeJailDuration)
	if err != nil {
		return
	}

	params.SlashFractionDoubleSign, err = sdk.NewDecFromStr(slashingParams.SlashFractionDoubleSign)
	if err != nil {
		return
	}

	params.SlashFractionDowntime, err = sdk.NewDecFromStr(slashingParams.SlashFractionDowntime)
	if err != nil {
		return
	}

	return
}

// createAndExecuteVaa creates and executes a governance VAA on the wormchain
func createAndExecuteVaa(ctx context.Context, guardians *guardians.ValSet, wormchain *cosmos.CosmosChain, payloadBytes []byte) error {
	v := helpers.GenerateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payloadBytes)
	vBz, err := v.Marshal()
	if err != nil {
		return err
	}
	vHex := hex.EncodeToString(vBz)

	_, err = wormchain.FullNodes[0].ExecTx(ctx, "faucet", "wormhole", "execute-gateway-governance-vaa", vHex)
	if err != nil {
		return err
	}

	return nil
}

func verifyParams(t *testing.T, ctx context.Context, wormchain *cosmos.CosmosChain) {
	// query the slashing params
	slashingParams, err := querySlashingParams(ctx, wormchain)
	require.NoError(t, err)

	// validate the slashing params did not change
	require.Equal(t, int64(200), slashingParams.SignedBlocksWindow)
	require.Equal(t, "0.100000000000000000", slashingParams.MinSignedPerWindow.String())
	require.Equal(t, 300*time.Second, slashingParams.DowntimeJailDuration)
	require.Equal(t, "0.200000000000000000", slashingParams.SlashFractionDoubleSign.String())
	require.Equal(t, "0.300000000000000000", slashingParams.SlashFractionDowntime.String())
}

// TestSlashingParamsUpdateVaa tests the execution of a slashing params update VAA
// and verifies that the governance module correctly updates the slashing params
func TestSlashingParamsUpdateVaa(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// base setup
	guardians := guardians.CreateValSet(t, 2)
	chain := createSingleNodeCluster(t, *guardians)
	ctx, _ := buildSingleNodeInterchain(t, chain)
	require.NotNil(t, ctx)

	wormchain := chain.(*cosmos.CosmosChain)

	// ------------------------------

	// create a governance VAA -- happy path
	payloadBytes, err := createSlashingParamsUpdate(200, "0.1", 300, "0.2", "0.3")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.NoError(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)

	// ------------------------------

	// create a governance VAA - invalid signed blocks window
	payloadBytes, err = createSlashingParamsUpdate(0, "0.1", 300, "0.2", "0.3")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.Error(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)

	// ------------------------------

	// create a governance VAA - invalid downtime jail duration
	payloadBytes, err = createSlashingParamsUpdate(200, "0.1", 0, "0.2", "0.3")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.Error(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)

	// ------------------------------

	// create a governance VAA - invalid slash fraction double sign
	payloadBytes, err = createSlashingParamsUpdate(200, "0.1", 300, "2.0", "0.3")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.Error(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)

	// ------------------------------

	// create a governance VAA - invalid slash fraction downtime
	payloadBytes, err = createSlashingParamsUpdate(200, "0.1", 300, "0.2", "2.0")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.Error(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)

	// ------------------------------

	// create a governance VAA - negative string values
	payloadBytes, err = createSlashingParamsUpdate(200, "-0.1", 300, "-0.2", "-2.0")
	require.NoError(t, err)

	// create and send
	err = createAndExecuteVaa(ctx, guardians, wormchain, payloadBytes)
	require.Error(t, err)

	// verify the slashing params
	verifyParams(t, ctx, wormchain)
}
