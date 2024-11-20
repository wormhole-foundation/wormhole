package ictest

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	types "github.com/wormhole-foundation/wormchain/x/wormhole/types"
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
	var coreModule [32]byte
	copy(coreModule[:], vaa.CoreModule[:])

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

	payload := make([]byte, 40)
	binary.BigEndian.PutUint64(payload[0:8], signedBlocksWindow)
	binary.BigEndian.PutUint64(payload[8:16], minSignedWindow.BigInt().Uint64())
	binary.BigEndian.PutUint64(payload[16:24], downtimeJailDurationSeconds)
	binary.BigEndian.PutUint64(payload[24:32], slashFractionDoubleSignDec.BigInt().Uint64())
	binary.BigEndian.PutUint64(payload[32:40], slashFractionDowntimeDec.BigInt().Uint64())

	gov_msg := types.NewGovernanceMessage(coreModule, byte(vaa.ActionSlashingParamsUpdate), uint16(vaa.ChainIDWormchain),
		payload)

	return gov_msg.MarshalBinary(), nil
}

// querySlashingParams queries the slashing params from the chain
func querySlashingParams(ctx context.Context, wormchain *cosmos.CosmosChain) (params slashingtypes.Params, err error) {
	// query the slashing params
	res, _, err := wormchain.GetNode().ExecQuery(ctx, "slashing", "params")
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

// TestSlashingParamsUpdateVaa tests the execution of a slashing params update VAA
// and verifies that the governance module correctly updates the slashing params
func TestSlashingParamsUpdateVaa(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Base setup
	guardians := guardians.CreateValSet(t, 2)
	chains := CreateLocalChain(t, *guardians)
	ic, ctx, _, _, _, _ := BuildInterchain(t, chains)
	require.NotNil(t, ic)
	require.NotNil(t, ctx)

	wormchain := chains[0].(*cosmos.CosmosChain)

	// Create a governance VAA
	payloadBytes, err := createSlashingParamsUpdate(200, "0.1", 300, "0.2", "0.3")
	require.NoError(t, err)

	v := helpers.GenerateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payloadBytes)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = wormchain.FullNodes[0].ExecTx(ctx, "faucet", "wormhole", "execute-governance-vaa", vHex)
	require.NoError(t, err)

	// query the slashing params
	slashingParams, err := querySlashingParams(ctx, wormchain)
	require.NoError(t, err)

	// validate the slashing params changed
	require.Equal(t, int64(200), slashingParams.SignedBlocksWindow)
	require.Equal(t, "0.100000000000000000", slashingParams.MinSignedPerWindow.String())
	require.Equal(t, 300*time.Second, slashingParams.DowntimeJailDuration)
	require.Equal(t, "0.200000000000000000", slashingParams.SlashFractionDoubleSign.String())
	require.Equal(t, "0.300000000000000000", slashingParams.SlashFractionDowntime.String())
}
