package interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/docker/docker/client"
	"github.com/icza/dyno"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos/wasm"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	interchaintestrelayer "github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	IBCRelayerImage   = "ghcr.io/cosmos/relayer"
	IBCRelayerVersion = "v2.5.0"

	WormchainName         = "wormchain"
	WormchainRemoteRepo   = "ghcr.io/strangelove-ventures/heighliner/wormchain"
	WormchainLocalRepo    = WormchainName
	WormchainLocalVersion = "local"
	WormchainDenom        = "uworm"

	WormchainBechPrefix = "wormhole"

	VotingPeriod     = "10s"
	MaxDepositPeriod = "10s"
	MinDepositAount  = "1000000"
	CoinType         = "118"

	WormchainImage = ibc.DockerImage{
		Repository: WormchainName,
		Version:    WormchainLocalVersion,
		UidGid:     "1025:1025",
	}

	WormchainConfig = ibc.ChainConfig{
		Type:           "cosmos",
		Name:           WormchainName,
		ChainID:        fmt.Sprintf("%s-1", WormchainName),
		Images:         []ibc.DockerImage{WormchainImage},
		Bin:            WormchainName + "d",
		Bech32Prefix:   WormchainBechPrefix,
		Denom:          WormchainDenom,
		CoinType:       CoinType,
		GasPrices:      fmt.Sprintf("0.0%s", WormchainDenom),
		Gas:            "auto",
		GasAdjustment:  1.8,
		TrustingPeriod: "112h",
		NoHostMount:    false,
		EncodingConfig: WormchainEncoding(),
	}

	numFull = 1
)

// WormchainEncoding returns the encoding config for the chain
func WormchainEncoding() *testutil.TestEncodingConfig {
	cfg := wasm.WasmEncoding()

	// register custom types
	wormholetypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return cfg
}

func CreateChains(t *testing.T, wormchainVersion string, guardians guardians.ValSet) []ibc.Chain {
	numWormchainVals := len(guardians.Vals)
	wormchainConfig.Images[0].Version = wormchainVersion

	// Create chain factory with wormchain
	wormchainConfig.ModifyGenesis = ModifyGenesis(votingPeriod, maxDepositPeriod, guardians)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     WormchainName,
			Version:       config.Images[0].Version,
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
		{
			Name:    "gaia",
			Version: "v10.0.1",
			ChainConfig: ibc.ChainConfig{
				Bech32Prefix: "cosmos",
				GasPrices:    "0.0uatom",
			},
		},
		{
			Name:    "osmosis",
			Version: "v15.1.2",
			ChainConfig: ibc.ChainConfig{
				Bech32Prefix:   "osmo",
				ChainID:        "osmosis-1002", // hardcoded handling in osmosis binary for osmosis-1, so need to override to something different.
				GasPrices:      "1.0uosmo",
				EncodingConfig: wasm.WasmEncoding(),
			},
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

func BuildInterchain(t *testing.T, chains []ibc.Chain) (*interchaintest.Interchain, context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client, string) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	wormGaiaPath := "wormgaia"
	wormOsmoPath := "wormosmo"
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		interchaintestrelayer.CustomDockerImage(IBCRelayerImage, IBCRelayerVersion, "100:1000"),
	).Build(t, client, network)

	ic.AddRelayer(r, "relayer")
	ic.AddLink(interchaintest.InterchainLink{
		Chain1:  chains[0], // Wormchain
		Chain2:  chains[1], // Gaia
		Relayer: r,
		Path:    wormGaiaPath,
	})
	ic.AddLink(interchaintest.InterchainLink{
		Chain1:  chains[0], // Wormchain
		Chain2:  chains[2], // Osmosis
		Relayer: r,
		Path:    wormOsmoPath,
	})

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		SkipPathCreation:  false,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer
	err = r.StartRelayer(ctx, eRep, wormGaiaPath, wormOsmoPath)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				t.Logf("an error occured while stopping the relayer: %s", err)
			}
		},
	)

	return ic, ctx, r, eRep, client, network
}

// Modify the genesis file:
// * Goverance - i.e. voting period
// * Get generated val set
// * Get faucet address
// * Set Guardian Set List using new val set
// * Set Guardian Validator List using new val set
// * Allow list the faucet address
func ModifyGenesis(votingPeriod string, maxDepositPeriod string, guardians guardians.ValSet, numVals int, skipRelayers bool) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}

		isSdk47 := true

		for _, env := range chainConfig.Env {
			if env == "ICT_ABOVE_SDK_47=false" {
				isSdk47 = false
			}
		}

		votingParams := "params"
		depositParams := "params"

		if !isSdk47 {
			votingParams = "voting_params"
			depositParams = "deposit_params"
		}

		// Modify gov
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", votingParams, "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", depositParams, "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set max deposit period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", depositParams, "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set min deposit in genesis json: %w", err)
		}

		// Get validators
		var validators [][]byte
		for i := 0; i < numVals; i++ {
			validatorBech32, err := dyno.Get(g, "app_state", "genutil", "gen_txs", i, "body", "messages", 0, "delegator_address")
			if err != nil {
				return nil, fmt.Errorf("failed to get validator pub key: %w", err)
			}
			validatorAccAddr := helpers.MustAccAddressFromBech32(validatorBech32.(string), chainConfig.Bech32Prefix).Bytes()
			validators = append(validators, validatorAccAddr)
		}

		// Get faucet address
		faucetAddress, err := dyno.Get(g, "app_state", "auth", "accounts", numVals, "address")
		if err != nil {
			return nil, fmt.Errorf("failed to get faucet address: %w", err)
		}

		var relayerAddress interface{}
		if !skipRelayers {
			// Get relayer address
			relayerAddress, err = dyno.Get(g, "app_state", "auth", "accounts", numVals+1, "address")
			if err != nil {
				return nil, fmt.Errorf("failed to get relayer address: %w", err)
			}
		}

		// Set guardian set list and validators
		guardianSetList := []wormholetypes.GuardianSet{}
		guardianSet := wormholetypes.GuardianSet{
			Index: 0,
			Keys:  [][]byte{},
		}
		guardianValidators := []GuardianValidator{}
		for i := 0; i < numVals; i++ {
			guardianSet.Keys = append(guardianSet.Keys, guardians.Vals[i].Addr)
			guardianValidators = append(guardianValidators, wormholetypes.GuardianValidator{
				GuardianKey:   guardians.Vals[i].Addr,
				ValidatorAddr: validators[i],
			})
		}
		guardianSetList = append(guardianSetList, guardianSet)
		if err := dyno.Set(g, guardianSetList, "app_state", "wormhole", "guardianSetList"); err != nil {
			return nil, fmt.Errorf("failed to set guardian set list: %w", err)
		}
		if err := dyno.Set(g, guardianValidators, "app_state", "wormhole", "guardianValidatorList"); err != nil {
			return nil, fmt.Errorf("failed to set guardian validator list: %w", err)
		}

		allowedAddresses := []wormholetypes.ValidatorAllowedAddress{}
		allowedAddresses = append(allowedAddresses, wormholetypes.ValidatorAllowedAddress{
			ValidatorAddress: sdk.MustBech32ifyAddressBytes(chainConfig.Bech32Prefix, validators[0]),
			AllowedAddress:   faucetAddress.(string),
			Name:             "Faucet",
		})
		allowedAddresses = append(allowedAddresses, ValidatorAllowedAddress{
			ValidatorAddress: sdk.MustBech32ifyAddressBytes(chainConfig.Bech32Prefix, validators[0]),
			AllowedAddress:   relayerAddress.(string),
			Name:             "Relayer",
		})
		if err := dyno.Set(g, allowedAddresses, "app_state", "wormhole", "allowedAddresses"); err != nil {
			return nil, fmt.Errorf("failed to set guardian validator list: %w", err)
		}

		config := wormholetypes.Config{
			GuardianSetExpiration: 86400,
			GovernanceEmitter:     vaa.GovernanceEmitter[:],
			GovernanceChain:       uint32(vaa.GovernanceChain),
			ChainId:               uint32(vaa.ChainIDWormchain),
		}
		if err := dyno.Set(g, config, "app_state", "wormhole", "config"); err != nil {
			return nil, fmt.Errorf("failed to set guardian validator list: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		fmt.Println("Genesis: ", string(out))
		return out, nil
	}
}
