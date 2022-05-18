// This specifies the interface to the chain specific Eth / EVM libraries.
// This interface should be implemented for each chain that has a unique go-ethereum or "go-ethereum-ish" library.

package common

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethEvent "github.com/ethereum/go-ethereum/event"

	ethAbi "github.com/certusone/wormhole/node/pkg/ethereum/abi"

	"go.uber.org/zap"
)

type NewBlock struct {
	Number *big.Int
	Hash   ethCommon.Hash
}

type Ethish interface {
	SetLogger(l *zap.Logger)
	DialContext(ctx context.Context, rawurl string) error
	NewAbiFilterer(address ethCommon.Address) error
	NewAbiCaller(address ethCommon.Address) error
	GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error)
	GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error)
	WatchLogMessagePublished(ctx, timeout context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error)
	TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error)
	TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error)
	ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error)
	SubscribeForBlocks(ctx context.Context, sink chan<- *NewBlock) (ethereum.Subscription, error)
}
