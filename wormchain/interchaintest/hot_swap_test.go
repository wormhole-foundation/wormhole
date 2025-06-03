package ictest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"go.uber.org/zap/zaptest"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
	wormholesdk "github.com/wormhole-foundation/wormhole/sdk"
)

func SetupHotSwapChain(t *testing.T, wormchainVersion string, guardians guardians.ValSet, numVals int) ibc.Chain {
	WormchainConfig.Images[0].Version = wormchainVersion

	if wormchainVersion == "local" {
		WormchainConfig.Images[0].Repository = "wormchain"
	}

	// Create chain factory with wormchain
	WormchainConfig.ModifyGenesis = ModifyGenesis(VotingPeriod, MaxDepositPeriod, guardians, numVals, true)

	numFullNodes := 0
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     "wormchain",
			ChainConfig:   WormchainConfig,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains[0]
}

type ValidatorInfo struct {
	Validator  *cosmos.ChainNode
	Bech32Addr string
	AccAddr    sdk.AccAddress
}

type QueryAllGuardianValidatorResponse struct {
	GuardianValidators []wormholetypes.GuardianValidator `json:"guardianValidator"`
}

type QueryGetGuardianValidatorResponse struct {
	GuardianValidator wormholetypes.GuardianValidator `json:"guardianValidator"`
}

func TestValidatorHotswap(t *testing.T) {
	// Base setup
	numGuardians := 2
	numVals := 3
	guardians := guardians.CreateValSet(t, numGuardians)
	chain := SetupHotSwapChain(t, "local", *guardians, numVals)

	ic := interchaintest.NewInterchain().AddChain(chain)
	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	wormchain := chain.(*cosmos.CosmosChain)

	// ============================

	// Query active guardian validators (returns both keys & sdk acc address)
	res, _, err := wormchain.Validators[0].ExecQuery(ctx, "wormhole", "list-guardian-validator")
	require.NoError(t, err)

	// Validate response
	var guardianValidators QueryAllGuardianValidatorResponse
	err = json.Unmarshal(res, &guardianValidators)
	require.NoError(t, err)
	require.Equal(t, numGuardians, len(guardianValidators.GuardianValidators))

	// ============================

	// NOTE:
	//
	// wormchain.Validators & the guardan query do not guarantee order, so we need to map the validators to match the order
	// of the guardian set reference.

	// First guardian key refs - will swap from using first validator to last validator, then back again
	firstGuardianKey := guardianValidators.GuardianValidators[0].GuardianKey
	firstGuardianPrivKey := guardians.Vals[0].Priv
	if !bytes.Equal(firstGuardianKey, guardians.Vals[0].Addr) {
		firstGuardianPrivKey = guardians.Vals[1].Priv
	}

	// Guardian validatore sdk addresses
	firstGuardianValAddr := sdk.AccAddress(guardianValidators.GuardianValidators[0].ValidatorAddr)
	secondGuardianValAddr := sdk.AccAddress(guardianValidators.GuardianValidators[1].ValidatorAddr)

	// Map validators to guardian set order
	var validators [3]ValidatorInfo
	for _, val := range wormchain.Validators {
		valBech32Addr, err := val.AccountKeyBech32(ctx, "validator")
		require.NoError(t, err)

		valInfo := ValidatorInfo{
			Validator:  val,
			Bech32Addr: valBech32Addr,
			AccAddr:    helpers.MustAccAddressFromBech32(valBech32Addr, "wormhole"),
		}

		if strings.Contains(valInfo.AccAddr.String(), firstGuardianValAddr.String()) {
			validators[0] = valInfo
		} else if strings.Contains(valInfo.AccAddr.String(), secondGuardianValAddr.String()) {
			validators[1] = valInfo
		} else {
			validators[2] = valInfo
		}
	}

	// Ensure all validators are mapped
	require.NotNil(t, validators[0])
	require.NotNil(t, validators[1])
	require.NotNil(t, validators[2])

	// References to first & last validator
	firstVal := validators[0]
	newVal := validators[2]

	// ============================

	// Ensure chain can produce blocks with the last validator shut down,
	// as it is not in the active set
	newVal.Validator.StopContainer(ctx)
	err = testutil.WaitForBlocks(ctx, 10, wormchain)
	require.NoError(t, err)
	newVal.Validator.StartContainer(ctx)

	// ============================

	// Query the first guardian's validator
	guardianKey := hex.EncodeToString(firstGuardianKey)
	res, _, err = newVal.Validator.ExecQuery(ctx, "wormhole", "show-guardian-validator", guardianKey)
	require.NoError(t, err)

	// Ensure the first guardian's validator is set to the first validator
	var valResponse wormholetypes.QueryGetGuardianValidatorResponse
	err = json.Unmarshal(res, &valResponse)
	require.NoError(t, err)
	require.Equal(t, firstGuardianKey, valResponse.GuardianValidator.GuardianKey)
	require.Equal(t, firstVal.AccAddr.Bytes(), valResponse.GuardianValidator.ValidatorAddr)

	// ============================

	// Use first validator to allow list the last validator (as it is not in active set)
	_, err = firstVal.Validator.ExecTx(ctx, "validator", "wormhole", "create-allowed-address", newVal.Bech32Addr, "newVal")
	require.NoError(t, err)

	// Migrate first guardian to use last validator
	addrHash := crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, newVal.AccAddr)
	sig, err := crypto.Sign(addrHash[:], firstGuardianPrivKey)
	require.NoErrorf(t, err, "failed to sign wormchain address: %v", err)
	_, err = newVal.Validator.ExecTx(ctx, "validator", "wormhole", "register-account-as-guardian", hex.EncodeToString(sig))
	require.NoError(t, err)

	// Query the first guardian's validator
	res, _, err = newVal.Validator.ExecQuery(ctx, "wormhole", "show-guardian-validator", guardianKey)
	require.NoError(t, err)

	// Ensure the first guardian's validator is set to the last validator
	err = json.Unmarshal(res, &valResponse)
	require.NoError(t, err)
	require.Equal(t, firstGuardianKey, valResponse.GuardianValidator.GuardianKey)
	require.Equal(t, newVal.AccAddr.Bytes(), valResponse.GuardianValidator.ValidatorAddr)

	// Wait 10 blocks to ensure blocks are being produced
	err = testutil.WaitForBlocks(ctx, 10, wormchain)
	require.NoError(t, err)

	// ============================

	// Use last validator to allow list the first validator (as it is not in active set *anymore)
	_, err = newVal.Validator.ExecTx(ctx, "validator", "wormhole", "create-allowed-address", firstVal.Bech32Addr, "firstVal")
	require.NoError(t, err)

	// Migrate first guardian back to use first validator
	addrHash = crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, firstVal.AccAddr)
	sig, err = crypto.Sign(addrHash[:], firstGuardianPrivKey)
	require.NoErrorf(t, err, "failed to sign wormchain address: %v", err)
	_, err = firstVal.Validator.ExecTx(ctx, "validator", "wormhole", "register-account-as-guardian", hex.EncodeToString(sig))
	require.NoError(t, err)

	// Query the first guardian's validator
	res, _, err = firstVal.Validator.ExecQuery(ctx, "wormhole", "show-guardian-validator", guardianKey)
	require.NoError(t, err)

	// Ensure the first guardian's validator is set to the first validator
	err = json.Unmarshal(res, &valResponse)
	require.NoError(t, err)
	require.Equal(t, firstGuardianKey, valResponse.GuardianValidator.GuardianKey)
	require.Equal(t, firstVal.AccAddr.Bytes(), valResponse.GuardianValidator.ValidatorAddr)

	// Wait 10 blocks to ensure blocks are being produced
	err = testutil.WaitForBlocks(ctx, 10, wormchain)
	require.NoError(t, err)
}
