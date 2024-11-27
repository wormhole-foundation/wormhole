package ictest

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"
	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
)

const CUSTOM_IBC_VERSION string = "ibc-wormhole-v1"

func createChains(t *testing.T, wormchainVersion string, guardians guardians.ValSet) []ibc.Chain {
	numWormchainVals := len(guardians.Vals)
	wormchainConfig.Images[0].Version = wormchainVersion

	// Create chain factory with wormchain
	wormchainConfig.ModifyGenesis = ModifyGenesis(votingPeriod, maxDepositPeriod, guardians)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		
		{
			ChainName:     "wormchain",
			ChainConfig:   wormchainConfig,
			NumValidators: &numWormchainVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "osmosis",
			Version: "v15.1.2",
			ChainConfig: ibc.ChainConfig{
				ChainID:        "osmosis-1002",
				GasPrices:      "1.0uosmo",
			},
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}
func buildInterchain(t *testing.T, chains []ibc.Chain) (context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	wormOsmoPath := "wormosmo"
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(t, client, network)
	ic.AddRelayer(r, "relayer")

	ic.AddLink(interchaintest.InterchainLink{
		Chain1:  chains[1], // Osmosis
		Chain2:  chains[0], // Wormchain
		Relayer: r,
		Path:    wormOsmoPath,
	})

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		SkipPathCreation:  true,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer
	err = r.StartRelayer(ctx, eRep, wormOsmoPath)
	require.NoError(t, err)

	//interchaintest.TempDir(sui)
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				t.Logf("an error occured while stopping the relayer: %s", err)
			}
		},
	)

	return ctx, r, eRep, client
}

func TestCwWormholeHappyPath(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := createChains(t, "v2.24.2", *guardians)
	ctx, _, _,  _ := buildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Instantiate the cw_wormhole contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	wormchainCoreContractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	
	contractAddr := wormchainCoreContractInfo.Address

	t.Logf("wormchain core contract address: %s", contractAddr)
	
}
