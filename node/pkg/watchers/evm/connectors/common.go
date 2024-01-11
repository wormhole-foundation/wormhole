package connectors

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

type BlockMarshaller struct {
	Number *hexutil.Big
	Hash   common.Hash    `json:"hash"`
	Time   hexutil.Uint64 `json:"timestamp"`

	// L1BlockNumber is the L1 block number in which an Arbitrum batch containing this block was submitted.
	// This field is only populated when connecting to Arbitrum.
	L1BlockNumber *hexutil.Big
}

type NewBlock struct {
	Number        *big.Int
	Hash          common.Hash
	Time          uint64
	L1BlockNumber *big.Int // This is only populated on some chains (Arbitrum)
	Finality      FinalityLevel
}

func (b *NewBlock) Copy(f FinalityLevel) *NewBlock {
	return &NewBlock{
		Number:        b.Number,
		Hash:          b.Hash,
		Time:          b.Time,
		L1BlockNumber: b.L1BlockNumber,
		Finality:      f,
	}
}

// Connector exposes Wormhole-specific interactions with an EVM-based network
type Connector interface {
	NetworkName() string
	ContractAddress() common.Address
	GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error)
	GetGuardianSet(ctx context.Context, index uint32) (ethabi.StructsGuardianSet, error)
	WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethabi.AbiLogMessagePublished) (event.Subscription, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	TimeOfBlockByHash(ctx context.Context, hash common.Hash) (uint64, error)
	ParseLogMessagePublished(log types.Log) (*ethabi.AbiLogMessagePublished, error)
	SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error)
	RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	RawBatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	Client() *ethClient.Client
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

type PollSubscription struct {
	errOnce   sync.Once
	err       chan error    // subscription consumer reads, subscription fulfiller writes. used to propagate errors.
	quit      chan error    // subscription consumer writes, subscription fulfiller reads. used to signal that consumer wants to cancel the subscription.
	unsubDone chan struct{} // subscription consumer reads, subscription fulfiller writes. used to signal that the subscription was successfully canceled
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
		close(sub.err) // TODO FIXME this violates golang guidelines “Only the sender should close a channel, never the receiver. Sending on a closed channel will cause a panic.”
	})
}
