package finalizers

import (
	"context"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	arbitrumAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/arbitrumabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"

	"go.uber.org/zap"
)

// ArbitrumFinalizer implements the finality check for Arbitrum.
// Arbitrum blocks should not be considered finalized until they show up on Ethereum.
// To determine when a block is final, we have to query the NodeInterface precompiled contract on Arbitrum.

// To build the ABI for the NodeInterface precomile, do the following:
// - Download this file to ethereum/contracts/NodeInterface.sol and build the contracts.
//     https://developer.offchainlabs.com/assets/files/NodeInterface-1413a19adf5bfcf97f5a5d3207d1b452.sol
// - Edit ethereum/build/contracts/NodeInterface.json and delete all but the bracketed part of “abi:”.
// - cd thirdparty/abigen and do the following to build the abigen tool:
//     go build -mod=readonly -o abigen github.com/ethereum/go-ethereum/cmd/abigen
// - cd to the wormhole directory and do the following:
//   - mkdir node/pkg/watchers/evm/connectors/arbitrumabi
//   - third_party/abigen/abigen --abi ethereum/build/contracts/NodeInterface.json --pkg abi_arbitrum --out node/pkg/watchers/evm/connectors/arbitrumabi/abi.go

type ArbitrumFinalizer struct {
	logger    *zap.Logger
	connector connectors.Connector
	caller    *arbitrumAbi.AbiArbitrumCaller
}

const (
	arbitrumNodeInterfacePrecompileAddr = "0x00000000000000000000000000000000000000C8"
)

func NewArbitrumFinalizer(logger *zap.Logger, connector connectors.Connector, client *ethClient.Client) *ArbitrumFinalizer {
	caller, err := arbitrumAbi.NewAbiArbitrumCaller(ethCommon.HexToAddress(arbitrumNodeInterfacePrecompileAddr), client)
	if err != nil {
		panic(fmt.Errorf("failed to create Arbitrum finalizer: %w", err))
	}

	return &ArbitrumFinalizer{
		logger:    logger,
		connector: connector,
		caller:    caller,
	}
}

func (a *ArbitrumFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	_, err := a.caller.FindBatchContainingBlock(&ethBind.CallOpts{Context: ctx}, block.Number.Uint64())
	if err != nil {
		// "requested block 430842 is after latest on-chain block 430820 published in batch 4686"
		if strings.ContainsAny(err.Error(), "is after latest on-chain block") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
