package main

// This is a tool that queries the RPCs to verify that the `EvmChainID` values in the chain config maps are correct as
// compared to the specified well-known public RPC endpoint. This tool should be run whenever either of the chain config maps are updated.

// Usage: go run verify.go [--env `env``]
//        Where `env` may be "mainnet", "testnet" or "both" where the default is "both".

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethHexUtil "github.com/ethereum/go-ethereum/common/hexutil"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

var (
	envStr  = flag.String("env", "both", `Environment to be validated, may be "mainnet", "testnet" or "both", default is "both"`)
	chainId = flag.Int("chainId", 0, `An individual chain to be validated, default is all chains`)
)

func main() {
	flag.Parse()

	if *chainId > math.MaxUint16 {
		fmt.Printf("chainId is not a valid Uint16: %d", *chainId)
		os.Exit(1)
	}

	if *envStr == "both" {
		verifyForEnv(common.MainNet, vaa.ChainID(*chainId)) // #nosec G115 -- Conversion is checked above
		verifyForEnv(common.TestNet, vaa.ChainID(*chainId)) // #nosec G115 -- Conversion is checked above
	} else {
		env, err := common.ParseEnvironment(*envStr)
		if err != nil || (env != common.TestNet && env != common.MainNet) {
			if *envStr == "" {
				fmt.Printf("Please specify --env\n")
			} else {
				fmt.Printf("Invalid value for --env, should be `mainnet`, `testnet` or `both`, is `%s`\n", *envStr)
			}
			os.Exit(1)
		}

		verifyForEnv(env, vaa.ChainID(*chainId)) // #nosec G115 -- Conversion is checked above
	}
}

type ListEntry struct {
	ChainID vaa.ChainID
	Entry   evm.EnvEntry
}

func verifyForEnv(env common.Environment, chainID vaa.ChainID) {
	m, err := evm.GetChainConfigMap(env)
	if err != nil {
		fmt.Printf("Failed to get chain config map for %snet\n", env)
		os.Exit(1)
	}

	// Create a slice sorted by ChainID that corresponds to the Chain Config Map.
	orderedList := []ListEntry{}
	for chainId, entry := range m {
		orderedList = append(orderedList, ListEntry{chainId, entry})
	}

	sort.Slice(orderedList, func(i, j int) bool {
		return orderedList[i].ChainID < orderedList[j].ChainID
	})

	ctx := context.Background()

	for _, entry := range orderedList {
		if chainID == vaa.ChainIDUnset || entry.ChainID == chainID {
			if entry.Entry.PublicRPC == "" {
				fmt.Printf("Skipping %v %v because the rpc is null\n", env, entry.ChainID)
			} else {
				fmt.Printf("Verifying %v %v...\n", env, entry.ChainID)

				fmt.Printf("   Verifying EVM chain ID for %v %v ", env, entry.ChainID)
				evmChainID, err := evm.QueryEvmChainID(ctx, entry.Entry.PublicRPC)
				if err != nil {
					fmt.Printf("\u2717\n   ERROR: Failed to query EVM chain ID for %v %v: %v\n", env, entry.ChainID, err)
					os.Exit(1)
				}

				if evmChainID != entry.Entry.EvmChainID {
					fmt.Printf("\u2717\n   ERROR: EVM chain ID mismatch for %v %v: config: %v, actual: %v\n", env, entry.ChainID, entry.Entry.EvmChainID, evmChainID)
					os.Exit(1)
				}

				fmt.Printf("\u2713\n")

				if entry.Entry.Finalized || entry.Entry.Safe {
					fmt.Printf("   Verifying finality values for %v %v ", env, entry.ChainID)
					err := verifyFinality(ctx, entry.Entry.PublicRPC, entry.Entry.Finalized, entry.Entry.Safe)
					if err != nil {
						fmt.Printf("\u2717\n   ERROR: failed to verify finality values for %v %v: %v\n", env, entry.ChainID, err)
						os.Exit(1)
					}
					fmt.Println("\u2713")
				}

				if entry.Entry.ContractAddr != "" {
					fmt.Printf("   Verifying contract address for %v %v ", env, entry.ChainID)
					err := verifyContractAddr(ctx, entry.Entry.PublicRPC, entry.Entry.ContractAddr)
					if err != nil {
						fmt.Printf("\u2717\n   ERROR: failed to verify contract for %v %v: %v\n", env, entry.ChainID, err)
						os.Exit(1)
					}
					fmt.Println("\u2713")
				}
			}
		}
	}
}

func verifyFinality(ctx context.Context, url string, finalized, safe bool) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := rpc.DialContext(timeout, url)
	if err != nil {
		return fmt.Errorf("failed to connect to endpoint: %w", err)
	}

	type Marshaller struct {
		Number *ethHexUtil.Big
	}
	var m Marshaller

	if finalized {
		err = c.CallContext(ctx, &m, "eth_getBlockByNumber", "finalized", false)
		if err != nil || m.Number == nil {
			return fmt.Errorf("finalized not supported by the node when it should be")
		}
	}

	if safe {
		err = c.CallContext(ctx, &m, "eth_getBlockByNumber", "safe", false)
		if err != nil || m.Number == nil {
			return fmt.Errorf("safe not supported by the node when it should be")
		}
	}

	return nil
}

func verifyContractAddr(ctx context.Context, url string, contractAddr string) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rawClient, err := ethRpc.DialContext(timeout, url)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	client := ethClient.NewClient(rawClient)

	caller, err := ethAbi.NewAbiCaller(ethCommon.BytesToAddress(ethCommon.HexToAddress(contractAddr).Bytes()), client)
	if err != nil {
		return fmt.Errorf("failed to create abi caller: %w", err)
	}

	_, err = caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: timeout})
	if err != nil {
		return err
	}

	return nil
}
