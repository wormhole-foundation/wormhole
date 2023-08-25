package ccq

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

func FetchCurrentGuardianSet(rpcUrl, coreAddr string) (*common.GuardianSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ethContract := eth_common.HexToAddress(coreAddr)
	rawClient, err := ethRpc.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum")
	}
	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(ethContract, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create caller")
	}
	currentIndex, err := caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}
	gs, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: ctx}, currentIndex)
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}
	return &common.GuardianSet{
		Keys:  gs.Keys,
		Index: currentIndex,
	}, nil
}
