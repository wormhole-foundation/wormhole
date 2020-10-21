package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/abi"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	EthBridgeWatcher struct {
		url              string
		bridge           eth_common.Address
		minConfirmations uint64

		pendingLocks      map[eth_common.Hash]*pendingLock
		pendingLocksGuard sync.Mutex

		lockChan chan *common.ChainLock
		setChan  chan *common.GuardianSet
	}

	pendingLock struct {
		lock   *common.ChainLock
		height uint64
	}
)

func NewEthBridgeWatcher(url string, bridge eth_common.Address, minConfirmations uint64, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet) *EthBridgeWatcher {
	return &EthBridgeWatcher{url: url, bridge: bridge, minConfirmations: minConfirmations, lockChan: lockEvents, setChan: setEvents, pendingLocks: map[eth_common.Hash]*pendingLock{}}
}

func (e *EthBridgeWatcher) Run(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	c, err := ethclient.DialContext(timeout, e.url)
	if err != nil {
		return fmt.Errorf("dialing eth client failed: %w", err)
	}

	f, err := abi.NewAbiFilterer(e.bridge, c)
	if err != nil {
		return fmt.Errorf("could not create wormhole bridge filter: %w", err)
	}

	caller, err := abi.NewAbiCaller(e.bridge, c)
	if err != nil {
		panic(err)
	}

	// Timeout for initializing subscriptions
	timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Subscribe to new token lockups
	tokensLockedC := make(chan *abi.AbiLogTokensLocked, 2)
	tokensLockedSub, err := f.WatchLogTokensLocked(&bind.WatchOpts{Context: timeout}, tokensLockedC, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	}
	defer tokensLockedSub.Unsubscribe()

	// Subscribe to guardian set changes
	guardianSetC := make(chan *abi.AbiLogGuardianSetChanged, 2)
	guardianSetEvent, err := f.WatchLogGuardianSetChanged(&bind.WatchOpts{Context: timeout}, guardianSetC)
	if err != nil {
		return fmt.Errorf("failed to subscribe to guardian set events: %w", err)
	}
	defer tokensLockedSub.Unsubscribe()

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	// Get initial validator set from Ethereum. We could also fetch it from Solana,
	// because both sets are synchronized, we simply made an arbitrary decision to use Ethereum.
	timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	idx, gs, err := FetchCurrentGuardianSet(timeout, e.url, e.bridge)
	if err != nil {
		return fmt.Errorf("failed requesting guardian set from Ethereum: %w", err)
	}
	logger.Info("initial guardian set fetched", zap.Any("value", gs), zap.Uint32("index", idx))
	e.setChan <- &common.GuardianSet{
		Keys:  gs.Keys,
		Index: idx,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-tokensLockedSub.Err():
				errC <- fmt.Errorf("error while processing token lockup subscription: %w", e)
				return
			case e := <-guardianSetEvent.Err():
				errC <- fmt.Errorf("error while processing guardian set subscription: %w", e)
				return
			case ev := <-tokensLockedC:
				// Request timestamp for block
				timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
				b, err := c.BlockByNumber(timeout, big.NewInt(int64(ev.Raw.BlockNumber)))
				cancel()
				if err != nil {
					errC <- fmt.Errorf("failed to request timestamp for block %d: %w", ev.Raw.BlockNumber, err)
					return
				}

				lock := &common.ChainLock{
					TxHash:        ev.Raw.TxHash,
					Timestamp:     time.Unix(int64(b.Time()), 0),
					Nonce:         ev.Nonce,
					SourceAddress: ev.Sender,
					TargetAddress: ev.Recipient,
					SourceChain:   vaa.ChainIDEthereum,
					TargetChain:   vaa.ChainID(ev.TargetChain),
					TokenChain:    vaa.ChainID(ev.TokenChain),
					TokenAddress:  ev.Token,
					TokenDecimals: ev.TokenDecimals,
					Amount:        ev.Amount,
				}

				logger.Info("found new lockup transaction", zap.Stringer("tx", ev.Raw.TxHash),
					zap.Uint64("block", ev.Raw.BlockNumber))
				e.pendingLocksGuard.Lock()
				e.pendingLocks[ev.Raw.TxHash] = &pendingLock{
					lock:   lock,
					height: ev.Raw.BlockNumber,
				}
				e.pendingLocksGuard.Unlock()
			case ev := <-guardianSetC:
				logger.Info("guardian set has changed, fetching new value",
					zap.Uint32("new_index", ev.NewGuardianIndex))

				gs, err := caller.GetGuardianSet(&bind.CallOpts{Context: timeout}, ev.NewGuardianIndex)
				if err != nil {
					// We failed to process the guardian set update and are now out of sync with the chain.
					// Recover by crashing the runnable, which causes the guardian set to be re-fetched.
					errC <- fmt.Errorf("error requesting new guardian set value for %d: %w", ev.NewGuardianIndex, err)
					return
				}

				logger.Info("new guardian set fetched", zap.Any("value", gs), zap.Uint32("index", ev.NewGuardianIndex))
				e.setChan <- &common.GuardianSet{
					Keys:  gs.Keys,
					Index: ev.NewGuardianIndex,
				}
			}
		}
	}()

	// Watch headers
	headSink := make(chan *types.Header, 2)
	headerSubscription, err := c.SubscribeNewHead(ctx, headSink)
	if err != nil {
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}
	defer headerSubscription.Unsubscribe()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-headerSubscription.Err():
				errC <- fmt.Errorf("error while processing header subscription: %w", e)
				return
			case ev := <-headSink:
				start := time.Now()
				logger.Info("processing new header", zap.Stringer("block", ev.Number))
				e.pendingLocksGuard.Lock()

				blockNumberU := ev.Number.Uint64()
				for hash, pLock := range e.pendingLocks {

					// Transaction was dropped and never picked up again
					if pLock.height+4*e.minConfirmations <= blockNumberU {
						logger.Debug("lockup timed out", zap.Stringer("tx", pLock.lock.TxHash),
							zap.Stringer("block", ev.Number))
						delete(e.pendingLocks, hash)
						continue
					}

					// Transaction is now ready
					if pLock.height+e.minConfirmations <= ev.Number.Uint64() {
						logger.Debug("lockup confirmed", zap.Stringer("tx", pLock.lock.TxHash),
							zap.Stringer("block", ev.Number))
						delete(e.pendingLocks, hash)
						e.lockChan <- pLock.lock
					}
				}

				e.pendingLocksGuard.Unlock()
				logger.Info("processed new header", zap.Stringer("block", ev.Number),
					zap.Duration("took", time.Since(start)))
			}
		}
	}()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// Fetch the current guardian set ID and guardian set from the chain.
func FetchCurrentGuardianSet(ctx context.Context, rpcURL string, bridgeContract eth_common.Address) (uint32, *abi.WormholeGuardianSet, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return 0, nil, fmt.Errorf("dialing eth client failed: %w", err)
	}

	caller, err := abi.NewAbiCaller(bridgeContract, c)
	if err != nil {
		panic(err)
	}

	opts := &bind.CallOpts{Context: ctx}

	currentIndex, err := caller.GuardianSetIndex(opts)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	gs, err := caller.GetGuardianSet(opts, currentIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}

	return currentIndex, &gs, nil
}
