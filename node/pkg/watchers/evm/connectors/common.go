package connectors

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type NewBlock struct {
	Number        *big.Int
	Hash          common.Hash
	L1BlockNumber *big.Int // This is only populated on some chains (Arbitrum)
	Safe          bool
}

// Connector exposes Wormhole-specific interactions with an EVM-based network
type Connector interface {
	NetworkName() string
	ContractAddress() common.Address
	GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error)
	GetGuardianSet(ctx context.Context, index uint32) (ethabi.StructsGuardianSet, error)
	WatchLogMessagePublished(ctx context.Context, sink chan<- *ethabi.AbiLogMessagePublished) (event.Subscription, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	TimeOfBlockByHash(ctx context.Context, hash common.Hash) (uint64, error)
	ParseLogMessagePublished(log types.Log) (*ethabi.AbiLogMessagePublished, error)
	SubscribeForBlocks(ctx context.Context, sink chan<- *NewBlock) (ethereum.Subscription, error)
	RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type PollSubscription struct {
	errOnce   sync.Once
	err       chan error
	quit      chan error
	unsubDone chan struct{}
}

func NewPollSubscription() *PollSubscription {
	return &PollSubscription{
		err:       make(chan error, 1),
		quit:      make(chan error, 1),
		unsubDone: make(chan struct{}, 1),
	}
}

var ErrUnsubscribed = errors.New("unsubscribed")

func (sub *PollSubscription) Err() <-chan error {
	return sub.err
}

func (sub *PollSubscription) Unsubscribe() {
	sub.errOnce.Do(func() {
		select {
		case sub.quit <- ErrUnsubscribed:
			<-sub.unsubDone
		case <-sub.unsubDone:
		}
		close(sub.err)
	})
}
