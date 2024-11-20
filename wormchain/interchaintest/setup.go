package ictest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/icza/dyno"

	interchaintest "github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos/wasm"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	// pathWormchainGaia   = "wormchain-gaia" // Replace with 2nd cosmos chain supporting wormchain
	// genesisWalletAmount = int64(10_000_000)
	votingPeriod     = "10s"
	maxDepositPeriod = "10s"
	coinType         = "118"
	wormchainConfig  = ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "wormchain",
		ChainID: "wormchain-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/strangelove-ventures/heighliner/wormchain",
				UidGid:     "1025:1025",
			},
		},
		Bin:            "wormchaind",
		Bech32Prefix:   "wormhole",
		Denom:          "uworm",
		CoinType:       coinType,
		GasPrices:      "0.00uworm",
		GasAdjustment:  1.8,
		TrustingPeriod: "112h",
		NoHostMount:    false,
		EncodingConfig: wormchainEncoding(),
	}
	numFullNodes = 1
)

// wormchainEncoding registers the Wormchain specific module codecs so that the associated types and msgs
// will be supported when writing to the blocksdb sqlite database.
func wormchainEncoding() *simappparams.EncodingConfig {
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
			ChainName:     "wormchain",
			ChainConfig:   wormchainConfig,
			NumValidators: &numWormchainVals,
			NumFullNodes:  &numFullNodes,
		},
		{Name: "gaia", Version: "v10.0.1", ChainConfig: ibc.ChainConfig{
			GasPrices: "0.0uatom",
		}},
		{
			Name:    "osmosis",
			Version: "v15.1.2",
			ChainConfig: ibc.ChainConfig{
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

func BuildInterchain(t *testing.T, chains []ibc.Chain) (context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client) {
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
	r := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
		t, client, network)
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

	return ctx, r, eRep, client
}

// Modify the genesis file:
// * Goverance - i.e. voting period
// * Get generated val set
// * Get faucet address
// * Set Guardian Set List using new val set
// * Set Guardian Validator List using new val set
// * Allow list the faucet address
func ModifyGenesis(votingPeriod string, maxDepositPeriod string, guardians guardians.ValSet) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		numVals := len(guardians.Vals)
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}

		// Modify gov
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "voting_params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "deposit_params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set max deposit period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "deposit_params", "min_deposit", 0, "denom"); err != nil {
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

		// Get relayer address
		relayerAddress, err := dyno.Get(g, "app_state", "auth", "accounts", numVals+1, "address")
		if err != nil {
			return nil, fmt.Errorf("failed to get relayer address: %w", err)
		}

		// Set guardian set list and validators
		guardianSetList := []GuardianSet{}
		guardianSet := GuardianSet{
			Index: 0,
			Keys:  [][]byte{},
		}
		guardianValidators := []GuardianValidator{}
		for i := 0; i < numVals; i++ {
			guardianSet.Keys = append(guardianSet.Keys, guardians.Vals[i].Addr)
			guardianValidators = append(guardianValidators, GuardianValidator{
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

		allowedAddresses := []ValidatorAllowedAddress{}
		allowedAddresses = append(allowedAddresses, ValidatorAllowedAddress{
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

// Replace these with reference to x/wormchain/types
type GuardianSet struct {
	Index          uint32   `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Keys           [][]byte `protobuf:"bytes,2,rep,name=keys,proto3" json:"keys,omitempty"`
	ExpirationTime uint64   `protobuf:"varint,3,opt,name=expirationTime,proto3" json:"expirationTime,omitempty"`
}

type ValidatorAllowedAddress struct {
	// the validator/guardian that controls this entry
	ValidatorAddress string `protobuf:"bytes,1,opt,name=validator_address,json=validatorAddress,proto3" json:"validator_address,omitempty"`
	// the allowlisted account
	AllowedAddress string `protobuf:"bytes,2,opt,name=allowed_address,json=allowedAddress,proto3" json:"allowed_address,omitempty"`
	// human readable name
	Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
}

type GuardianValidator struct {
	GuardianKey   []byte `protobuf:"bytes,1,opt,name=guardianKey,proto3" json:"guardianKey,omitempty"`
	ValidatorAddr []byte `protobuf:"bytes,2,opt,name=validatorAddr,proto3" json:"validatorAddr,omitempty"`
}
