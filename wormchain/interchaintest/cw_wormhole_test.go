package ictest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/cw_wormhole"
)

func createWormchainChains(t *testing.T, wormchainVersion string, guardians guardians.ValSet) []ibc.Chain {
	numWormchainVals := len(guardians.Vals)
	wormchainConfig.Images[0].Version = wormchainVersion

	// Create chain factory with wormchain
	wormchainConfig.ModifyGenesis = ModifyGenesis(votingPeriod, maxDepositPeriod, guardians, true)

	numFullNodes := 0

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     "wormchain",
			ChainConfig:   wormchainConfig,
			NumValidators: &numWormchainVals,
			NumFullNodes:  &numFullNodes,
		},
		// {
		// 	Name:    "osmosis",
		// 	Version: "v15.1.2",
		// 	ChainConfig: ibc.ChainConfig{
		// 		ChainID:        "osmosis-1002", // hardcoded handling in osmosis binary for osmosis-1, so need to override to something different.
		// 		GasPrices:      "1.0uosmo",
		// 		EncodingConfig: wasm.WasmEncoding(),
		// 	},
		// },
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

func buildMultipleChainsInterchain(t *testing.T, chains []ibc.Chain) (context.Context, *client.Client) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	// rep := testreporter.NewNopReporter()
	// eRep := rep.RelayerExecReporter(t)

	// wormOsmoPath := "wormosmo"
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)
	// r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t),
	// 	relayer.StartupFlags("-b", "100"),
	// 	relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "v2.5.2", "100:1000")).Build(
	// 	t, client, network)
	// ic.AddRelayer(r, "relayer")

	// ic.AddLink(interchaintest.InterchainLink{
	// 	Chain1:  chains[1], // Osmosis
	// 	Chain2:  chains[0], // Wormchain
	// 	Relayer: r,
	// 	Path:    wormOsmoPath,
	// })

	err := ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer
	// err = r.StartRelayer(ctx, eRep, wormOsmoPath)
	// require.NoError(t, err)

	// t.Cleanup(
	// 	func() {
	// 		// err := r.StopRelayer(ctx, eRep)
	// 		if err != nil {
	// 			t.Logf("an error occured while stopping the relayer: %s", err)
	// 		}
	// 	},
	// )

	return ctx, client
}

func TestCwWormholeHappyPath(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)

	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)

	wormchain := chains[0].(*cosmos.CosmosChain)

	// Instantiate the cw_wormhole contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	wormchainCoreContractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := wormchainCoreContractInfo.Address

	// Query the contract to check that the guardian set is correct
	var guardianSetResp cw_wormhole.GuardianSetQueryResponse
	err := wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		GuardianSetInfo: &cw_wormhole.QueryMsg_GuardianSetInfo{},
	}, &guardianSetResp)
	require.NoError(t, err)
	require.Equal(t, numVals, len(guardianSetResp.Data.Addresses), "guardian set should have the correct number of guardians")
	// Check that all the guardians from the query are the ones in the running valset
	for _, val := range guardians.Vals {
		found := false
		for _, guardian := range guardianSetResp.Data.Addresses {
			decoded, err := base64.StdEncoding.DecodeString(string(guardian.Bytes))
			require.NoError(t, err)
			guardianDecodedBytes := []byte(decoded)
			if bytes.Equal(val.Addr, guardianDecodedBytes) {
				found = true
				break
			}
		}
		require.True(t, found, "guardian not found in guardian set")
	}

	// Check that the core contract fee is set to 0uworm
	var stateResp cw_wormhole.GetStateQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		GetState: &cw_wormhole.QueryMsg_GetState{},
	}, &stateResp)
	require.NoError(t, err)
	require.Equal(t, "uworm", stateResp.Data.Fee.Denom, "core contract fee should be in uworm")
	require.Equal(t, cw_wormhole.Uint128("0"), stateResp.Data.Fee.Amount, "core contract fee should be 0")

	// Check that hex addresse are able to be queried
	var hexAddressResp cw_wormhole.QueryAddressHexQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{

		QueryAddressHex: &cw_wormhole.QueryMsg_QueryAddressHex{
			Address: "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465",
		},
	}, &hexAddressResp)
	require.NoError(t, err)
	require.IsType(t, "", hexAddressResp.Data.Hex, "hex address should be a string")

	// Check that the core contract can properly verify a VAA
	guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)
	vaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, []byte("test"))
	vaaBz, err := vaa.Marshal()
	require.NoError(t, err)
	encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
	vaaBinary := cw_wormhole.Binary(encodedVaa)

	currentWormchainBlock, err := wormchain.Height(ctx)
	require.NoError(t, err)

	var parsedVaaResponse cw_wormhole.VerifyVAAQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		VerifyVaa: &cw_wormhole.QueryMsg_VerifyVAA{
			BlockTime: int(currentWormchainBlock),
			Vaa:       vaaBinary,
		},
	}, &parsedVaaResponse)
	require.NoError(t, err)
	require.NotNil(t, parsedVaaResponse.Data, "VAA should be verified")
	require.Equal(t, "test", string(parsedVaaResponse.Data.Payload), "VAA payload should be what we passed in")

	// Attempt to submit the VAA properly
	// submitVaa := helpers.GenerateEmptyVAA(
	// 	t,
	// 	guardians,
	// 	wormhole_vaa.GatewayModuleStr,
	// 	wormhole_vaa.ActionCancelUpgrade,
	// 	wormhole_vaa.ChainIDWormchain,
	// )
	submitVaa := "010000000001003f3179d5bb17b6f2ecc13741ca3f78d922043e99e09975e3904332d2418bb3f16d7ac93ca8401f8bed1cf9827bc806ecf7c5a283340f033bf472724abf1d274f0000000000000000000001000000000000000000000000000000000000000000000000000000000000ffff000000000000000000010000000000000000000000000000000000000000000000000000000005f5e10001000000000000000000000000000000000000000000000000000000757573640003000000000000000000000000f7f7dde848e7450a029cd0a9bd9bdae4b5147db3000300000000000000000000000000000000000000000000000000000000000f4240"
	submitVaa = base64.StdEncoding.EncodeToString([]byte(submitVaa))
	executeVAAPayload, err := json.Marshal(cw_wormhole.ExecuteMsg{
		SubmitVaa: &cw_wormhole.ExecuteMsg_SubmitVAA{
			Vaa: cw_wormhole.Binary(submitVaa),
		},
	})
	require.NoError(t, err)
	_, err = wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
	require.NoError(t, err)
}
