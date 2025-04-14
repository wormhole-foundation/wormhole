package utils

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

func GetRpcUrl(network common.Environment) string {
	switch network {
	case common.MainNet:
		return "https://rpc.ankr.com/eth"
	case common.TestNet:
		return "https://rpc.ankr.com/eth_goerli"
	case common.UnsafeDevNet:
		return "http://localhost:8545"
	case common.GoTest:
		return "http://eth-devnet:8545"
	case common.AccountantMock:
		return ""
	default:
		return ""
	}
}

func FetchLatestBlockNumber(ctx context.Context, network common.Environment) (*big.Int, error) {
	rawUrl := GetRpcUrl(network)
	if rawUrl == "" {
		return nil, fmt.Errorf("unable to get rpc url")
	}
	return FetchLatestBlockNumberFromUrl(ctx, rawUrl)
}

func FetchLatestBlockNumberFromUrl(ctx context.Context, rawUrl string) (*big.Int, error) {
	rawClient, err := ethRpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to dial eth context: %w", err)
	}
	client := ethClient.NewClient(rawClient)
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch latest header: %w", err)
	}
	return header.Number, nil
}

func FetchCurrentGuardianSet(network common.Environment) (uint32, *ethAbi.StructsGuardianSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	rawUrl := GetRpcUrl(network)
	if rawUrl == "" {
		return 0, nil, fmt.Errorf("unable to get rpc url")
	}
	var ethContract string
	switch network {
	case common.MainNet:
		ethContract = "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
	case common.TestNet:
		ethContract = "0x706abc4E45D419950511e474C7B9Ed348A4a716c"
	case common.UnsafeDevNet:
	case common.GoTest:
		ethContract = "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
	case common.AccountantMock:
	default:
		return 0, nil, fmt.Errorf("unable to fetch guardian set for unknown network %s", network)
	}

	contract := eth_common.HexToAddress(ethContract)
	rawClient, err := ethRpc.DialContext(ctx, rawUrl)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to connect to ethereum")
	}
	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(contract, client)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create caller")
	}
	currentIndex, err := caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: ctx})
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	gs, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: ctx}, currentIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}

	return currentIndex, &gs, nil
}
