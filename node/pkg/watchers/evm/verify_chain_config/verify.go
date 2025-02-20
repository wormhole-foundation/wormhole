package main

// This is a tool that queries the RPCs to verify that the `EvmChainID` values in the chain config maps are correct as
// compared to the specified well-known public RPC endpoint. This tool should be run whenever either of the chain config maps are updated.

// Usage: go run verify.go [--env `env``]
//        Where `env` may be "mainnet", "testnet" or "both" where the default is "both".

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	envStr = flag.String("env", "both", `Environment to be validated, may be "mainnet", "testnet" or "both", default is "both"`)
)

func main() {
	flag.Parse()

	if *envStr == "both" {
		verifyForEnv(common.MainNet)
		verifyForEnv(common.TestNet)
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

		verifyForEnv(env)
	}
}

type ListEntry struct {
	ChainID vaa.ChainID
	Entry   evm.EnvEntry
}

func verifyForEnv(env common.Environment) {
	m, err := evm.GetChainConfigMap(env)
	if err != nil {
		fmt.Printf("Failed to get chain config map for %snet\n", env)
		os.Exit(1)
	}

	// The map is ordered by hash. Sort it by chain ID.
	orderedList := []ListEntry{}
	for chainId, entry := range m {
		orderedList = append(orderedList, ListEntry{chainId, entry})
	}

	sort.Slice(orderedList, func(i, j int) bool {
		return orderedList[i].ChainID < orderedList[j].ChainID
	})

	ctx := context.Background()

	for _, entry := range orderedList {
		if entry.Entry.PublicRPC == "" {
			fmt.Printf("Skipping %v %v because the rpc is null\n", env, entry.ChainID)
		} else {
			evmChainID, err := evm.QueryEvmChainID(ctx, entry.Entry.PublicRPC)
			if err != nil {
				fmt.Printf("ERROR: Failed to query EVM chain ID for %v %v: %v\n", env, entry.ChainID, err)
				os.Exit(1)
			}

			if evmChainID != entry.Entry.EvmChainID {
				fmt.Printf("ERROR: EVM chain ID mismatch for %v %v: config: %v, actual: %v\n", env, entry.ChainID, entry.Entry.EvmChainID, evmChainID)
				os.Exit(1)
			}

			fmt.Printf("EVM chain ID match for %v %v: value: %v\n", env, entry.ChainID, evmChainID)
		}
	}
}
