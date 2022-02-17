package ethereum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/ethereum/abi"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

var (
	ethConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_connection_errors_total",
			Help: "Total number of Ethereum connection errors (either during initial connection or while watching)",
		}, []string{"reason"})

	ethLockupsFound = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_eth_lockups_found_total",
			Help: "Total number of Eth lockups found (pre-confirmation)",
		})
	ethLockupsConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_eth_lockups_confirmed_total",
			Help: "Total number of Eth lockups verified (post-confirmation)",
		})
	ethMessagesOrphaned = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_lockups_orphaned_total",
			Help: "Total number of Eth lockups dropped (orphaned)",
		}, []string{"reason"})
	guardianSetChangesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_eth_guardian_set_changes_confirmed_total",
			Help: "Total number of guardian set changes verified (we only see confirmed ones to begin with)",
		})
	currentEthHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_eth_current_height",
			Help: "Current Ethereum block height",
		})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_eth_query_latency",
			Help: "Latency histogram for Ethereum calls (note that most interactions are streaming queries, NOT calls, and we cannot measure latency for those",
		}, []string{"operation"})
)

type (
	EthBridgeWatcher struct {
		url                    string
		bridge                 eth_common.Address
		minLockupConfirmations uint64

		lockChan chan *common.ChainLock
		setChan  chan *common.GuardianSet

		obsvReqC chan *gossipv1.ObservationRequest

		pending   map[pendingKey]*pendingLock
		pendingMu sync.Mutex

		// 0 is a valid guardian set, so we need a nil value here
		currentGuardianSet *uint32
	}

	pendingKey struct {
		TxHash    eth_common.Hash
		BlockHash eth_common.Hash
	}

	pendingLock struct {
		lock   *common.ChainLock
		height uint64
	}
)

func NewEthBridgeWatcher(url string, bridge eth_common.Address, minConfirmations uint64, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet, obsvReqC chan *gossipv1.ObservationRequest) *EthBridgeWatcher {
	return &EthBridgeWatcher{url: url, bridge: bridge, minLockupConfirmations: minConfirmations, lockChan: lockEvents, setChan: setEvents, pending: map[pendingKey]*pendingLock{}, obsvReqC: obsvReqC}
}

func (e *EthBridgeWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDEthereum, &gossipv1.Heartbeat_Network{
		ContractAddress: e.bridge.Hex(),
	})

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	c, err := ethclient.DialContext(timeout, e.url)
	if err != nil {
		ethConnectionErrors.WithLabelValues("dial_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
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
		ethConnectionErrors.WithLabelValues("subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
		return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	}

	// Subscribe to guardian set changes
	guardianSetC := make(chan *abi.AbiLogGuardianSetChanged, 2)
	guardianSetEvent, err := f.WatchLogGuardianSetChanged(&bind.WatchOpts{Context: timeout}, guardianSetC)
	if err != nil {
		ethConnectionErrors.WithLabelValues("subscribe_error").Inc()
		return fmt.Errorf("failed to subscribe to guardian set events: %w", err)
	}

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	// Get initial validator set from Ethereum. We could also fetch it from Solana,
	// because both sets are synchronized, we simply made an arbitrary decision to use Ethereum.
	timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	idx, gs, err := FetchCurrentGuardianSet(timeout, e.url, e.bridge)
	if err != nil {
		ethConnectionErrors.WithLabelValues("guardian_set_fetch_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
		return fmt.Errorf("failed requesting guardian set from Ethereum: %w", err)
	}
	logger.Info("initial guardian set fetched", zap.Any("value", gs), zap.Uint32("index", idx))
	e.setChan <- &common.GuardianSet{
		Keys:  gs.Keys,
		Index: idx,
	}

	// Track the current block number so we can compare it to the block number of
	// the message publication for observation requests.
	var currentBlockNumber uint64

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				// This can't happen unless there is a programming error - the caller
				// is expected to send us only requests for our chainID.
				if vaa.ChainID(r.ChainId) != vaa.ChainIDEthereum {
					panic("invalid chain ID")
				}

				tx := eth_common.BytesToHash(r.TxHash)
				logger.Info("received observation request",
					zap.String("tx_hash", tx.Hex()))

				// SECURITY: Load the block number before requesting the transaction to avoid a
				// race condition where requesting the tx succeeds and is then dropped due to a fork,
				// but blockNumberU had already advanced beyond the required threshold.
				//
				// In the primary watcher flow, this is of no concern since we assume the node
				// always sends the head before it sends the logs (implicit synchronization
				// by relying on the same websocket connection).
				blockNumberU := atomic.LoadUint64(&currentBlockNumber)
				if blockNumberU == 0 {
					logger.Error("no block number available, ignoring observation request")
					continue
				}

				timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
				blockNumber, msgs, err := MessageEventsForTransaction(timeout, c, e.bridge, tx)
				cancel()

				if err != nil {
					logger.Error("failed to process observation request")
					continue
				}

				for _, msg := range msgs {
					// SECURITY: In the recovery flow, we already know which transaction to
					// observe, and we can assume that it has reached the expected finality
					// level a long time ago. Therefore, the logic is much simpler than the
					// primary watcher, which has to wait for finality.
					//
					// Instead, we can simply check if the transaction's block number is in
					// the past by more than the expected confirmation number.
					//
					// Ensure that the current block number is at least expectedConfirmations
					// larger than the message observation's block number.
					if blockNumber+e.minLockupConfirmations <= blockNumberU {
						logger.Info("re-observed message publication transaction",
							zap.Stringer("tx", msg.TxHash),
							zap.Uint64("current_block", blockNumberU),
							zap.Uint64("observed_block", blockNumber),
						)
						e.lockChan <- msg
					} else {
						logger.Info("ignoring re-observed message publication transaction",
							zap.Stringer("tx", msg.TxHash),
							zap.Uint64("current_block", blockNumberU),
							zap.Uint64("observed_block", blockNumber),
						)
					}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-tokensLockedSub.Err():
				ethConnectionErrors.WithLabelValues("subscription_error").Inc()
				errC <- fmt.Errorf("error while processing token lockup subscription: %w", e)
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
				return
			case e := <-guardianSetEvent.Err():
				ethConnectionErrors.WithLabelValues("subscription_error").Inc()
				errC <- fmt.Errorf("error while processing guardian set subscription: %w", e)
				return
			case ev := <-tokensLockedC:
				// Request timestamp for block
				msm := time.Now()
				timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
				b, err := c.BlockByNumber(timeout, big.NewInt(int64(ev.Raw.BlockNumber)))
				cancel()
				queryLatency.WithLabelValues("block_by_number").Observe(time.Since(msm).Seconds())

				if err != nil {
					ethConnectionErrors.WithLabelValues("block_by_number_error").Inc()
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
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

				ethLockupsFound.Inc()

				key := pendingKey{
					TxHash:    ev.Raw.TxHash,
					BlockHash: ev.Raw.BlockHash,
				}

				e.pendingMu.Lock()
				e.pending[key] = &pendingLock{
					lock:   lock,
					height: ev.Raw.BlockNumber,
				}
				e.pendingMu.Unlock()
			case ev := <-guardianSetC:
				logger.Info("guardian set has changed, fetching new value",
					zap.Uint32("new_index", ev.NewGuardianIndex))

				guardianSetChangesConfirmed.Inc()

				msm := time.Now()
				timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
				gs, err := caller.GetGuardianSet(&bind.CallOpts{Context: timeout}, ev.NewGuardianIndex)
				cancel()
				queryLatency.WithLabelValues("get_guardian_set").Observe(time.Since(msm).Seconds())
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
		ethConnectionErrors.WithLabelValues("header_subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-headerSubscription.Err():
				ethConnectionErrors.WithLabelValues("header_subscription_error").Inc()
				errC <- fmt.Errorf("error while processing header subscription: %w", e)
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDEthereum, 1)
				return
			case ev := <-headSink:
				start := time.Now()
				currentHash := ev.Hash()
				logger.Info("processing new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("current_blockhash", currentHash))
				currentEthHeight.Set(float64(ev.Number.Int64()))
				readiness.SetReady(common.ReadinessEthSyncing)
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDEthereum, &gossipv1.Heartbeat_Network{
					Height:          ev.Number.Int64(),
					ContractAddress: e.bridge.Hex(),
				})

				e.pendingMu.Lock()

				blockNumberU := ev.Number.Uint64()
				atomic.StoreUint64(&currentBlockNumber, blockNumberU)

				for key, pLock := range e.pending {

					// Transaction was dropped and never picked up again
					if pLock.height+4*e.minLockupConfirmations <= blockNumberU {
						logger.Info("lockup timed out",
							zap.Stringer("tx", pLock.lock.TxHash),
							zap.Stringer("blockhash", key.BlockHash),
							zap.Stringer("current_block", ev.Number),
							zap.Stringer("current_blockhash", currentHash))
						ethMessagesOrphaned.WithLabelValues("timeout").Inc()
						delete(e.pending, key)
						continue
					}

					// Transaction is now ready
					if pLock.height+e.minLockupConfirmations <= ev.Number.Uint64() {
						timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
						tx, err := c.TransactionReceipt(timeout, pLock.lock.TxHash)
						cancel()

						// If the node returns an error after waiting expectedConfirmation blocks,
						// it means the chain reorged and the transaction was orphaned. The
						// TransactionReceipt call is using the same websocket connection than the
						// head notifications, so it's guaranteed to be atomic.
						//
						// Check multiple possible error cases - the node seems to return a
						// "not found" error most of the time, but it could conceivably also
						// return a nil tx or rpc.ErrNoResult.
						if tx == nil || err == rpc.ErrNoResult || (err != nil && err.Error() == "not found") {
							logger.Warn("tx was orphaned",
								zap.Stringer("tx", pLock.lock.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.Error(err))
							delete(e.pending, key)
							ethMessagesOrphaned.WithLabelValues("not_found").Inc()
							continue
						}

						// Any error other than "not found" is likely transient - we retry next block.
						if err != nil {
							logger.Warn("transaction could not be fetched",
								zap.Stringer("tx", pLock.lock.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.Error(err))
							continue
						}

						// It's possible for a transaction to be orphaned and then included in a different block
						// but with the same tx hash. Drop the observation (it will be re-observed and needs to
						// wait for the full confirmation time again).
						if tx.BlockHash != key.BlockHash {
							logger.Info("tx got dropped and mined in a different block; the message should have been reobserved",
								zap.Stringer("tx", pLock.lock.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash))
							delete(e.pending, key)
							ethMessagesOrphaned.WithLabelValues("blockhash_mismatch").Inc()
							continue
						}

						logger.Info("lockup confirmed",
							zap.Stringer("tx", pLock.lock.TxHash),
							zap.Stringer("blockhash", key.BlockHash),
							zap.Stringer("current_block", ev.Number),
							zap.Stringer("current_blockhash", currentHash))
						delete(e.pending, key)
						e.lockChan <- pLock.lock
						ethLockupsConfirmed.Inc()
					}
				}

				e.pendingMu.Unlock()
				logger.Info("processed new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("current_blockhash", currentHash),
					zap.Duration("took", time.Since(start)))
			}
		}
	}()

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
