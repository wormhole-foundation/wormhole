package interchaintest

import (
	"context"
	"fmt"
	"testing"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tokenfactorytypes "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var (
	WormchainName    = "wormchain"
	WormchainVersion = "local"
	WormchainDenom   = "uworm"

	WormchainBechPrefix = "wormhole"

	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"
	MinDepositAount  = "1000000"

	GenesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.params.voting_period",
			Value: VotingPeriod,
		},
		{
			Key:   "app_state.gov.params.max_deposit_period",
			Value: MaxDepositPeriod,
		},
		{
			Key:   "app_state.gov.params.min_deposit.0.denom",
			Value: WormchainDenom,
		},
	}

	WormchainImage = ibc.DockerImage{
		Repository: WormchainName,
		Version:    WormchainVersion,
		UidGid:     "1025:1025",
	}

	WormchainConfig = ibc.ChainConfig{
		Type:                "cosmos",
		Name:                WormchainName,
		ChainID:             "wormhole-1",
		Images:              []ibc.DockerImage{WormchainImage},
		Bin:                 WormchainName + "d",
		Bech32Prefix:        WormchainBechPrefix,
		Denom:               WormchainDenom,
		CoinType:            "118",
		GasPrices:           fmt.Sprintf("0%s", WormchainDenom),
		GasAdjustment:       1.0,
		TrustingPeriod:      "48h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      WormchainEncoding(),
		ModifyGenesis:       cosmos.ModifyGenesis(GenesisKV),
	}
)

// WormchainEncoding returns the encoding config for the chain
func WormchainEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// Add custom encoding overrides here
	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	wormholetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	tokenfactorytypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

// CreateChain generates a new chain with a custom image (useful for upgrades)
func CreateChain(t *testing.T, numVals, numFull int, img ibc.DockerImage) []ibc.Chain {
	cfg := WormchainConfig
	cfg.Images = []ibc.DockerImage{img}
	return CreateChainWithCustomConfig(t, numVals, numFull, cfg)
}

// CreateThisBranchChain generates this branch's chain (ex: from the commit)
func CreateThisBranchChain(t *testing.T, numVals, numFull int) []ibc.Chain {
	return CreateChain(t, numVals, numFull, WormchainImage)
}

func CreateChainWithCustomConfig(t *testing.T, numVals, numFull int, config ibc.ChainConfig) []ibc.Chain {
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          WormchainName,
			ChainName:     WormchainName,
			Version:       config.Images[0].Version,
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

func BuildInitialChain(t *testing.T, chains []ibc.Chain) (*interchaintest.Interchain, context.Context, *client.Client, string) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic = ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	return ic, ctx, client, network
}
