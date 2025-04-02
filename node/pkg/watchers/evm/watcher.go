package evm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"
	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	ethConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_connection_errors_total",
			Help: "Total number of Ethereum connection errors (either during initial connection or while watching)",
		}, []string{"eth_network", "reason"})

	ethMessagesObserved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_messages_observed_total",
			Help: "Total number of Eth messages observed (pre-confirmation)",
		}, []string{"eth_network"})
	ethMessagesOrphaned = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_messages_orphaned_total",
			Help: "Total number of Eth messages dropped (orphaned)",
		}, []string{"eth_network", "reason"})
	ethMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_messages_confirmed_total",
			Help: "Total number of Eth messages verified (post-confirmation)",
		}, []string{"eth_network"})
	currentEthHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_eth_current_height",
			Help: "Current Ethereum block height",
		}, []string{"eth_network"})
	currentEthSafeHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_eth_current_safe_height",
			Help: "Current Ethereum safe block height",
		}, []string{"eth_network"})
	currentEthFinalizedHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_eth_current_finalized_height",
			Help: "Current Ethereum finalized block height",
		}, []string{"eth_network"})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_eth_query_latency",
			Help: "Latency histogram for Ethereum calls (note that most interactions are streaming queries, NOT calls, and we cannot measure latency for those",
		}, []string{"eth_network", "operation"})
)

type (
	Watcher struct {
		// EVM RPC url.
		url string
		// Address of the EVM contract
		contract eth_common.Address
		// Human-readable name of the EVM network, for logging and monitoring.
		networkName string
		// Readiness component
		readinessSync readiness.Component
		// VAA ChainID of the network monitored by this watcher.
		chainID vaa.ChainID

		// Channel for sending new MesssagePublications. Messages should not be sent
		// to this channel directly. Instead, they should be wrapped by
		// a call to `verifyAndPublish()`.
		msgC chan<- *common.MessagePublication

		// Channel to send guardian set changes to.
		// setC can be set to nil if no guardian set changes are needed.
		//
		// We currently only fetch the guardian set from one primary chain, which should
		// have this flag set to true, and false on all others.
		//
		// The current primary chain is Ethereum (a mostly arbitrary decision because it
		// has the best API - we might want to switch the primary chain to Solana once
		// the governance mechanism lives there),
		setC chan<- *common.GuardianSet

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC <-chan *gossipv1.ObservationRequest

		// Incoming query requests from the network. Pre-filtered to only
		// include requests for our chainID.
		queryReqC <-chan *query.PerChainQueryInternal

		// Outbound query responses to query requests
		queryResponseC chan<- *query.PerChainQueryResponseInternal

		pending   map[pendingKey]*pendingMessage
		pendingMu sync.Mutex

		// 0 is a valid guardian set, so we need a nil value here
		currentGuardianSet *uint32

		// Interface to the chain specific ethereum library.
		ethConn connectors.Connector
		env     common.Environment
		logger  *zap.Logger

		latestBlockNumber          uint64
		latestSafeBlockNumber      uint64
		latestFinalizedBlockNumber uint64
		l1Finalizer                interfaces.L1Finalizer

		ccqConfig          query.PerChainConfig
		ccqMaxBlockNumber  *big.Int
		ccqTimestampCache  *BlocksByTimestamp
		ccqBackfillChannel chan *ccqBackfillRequest
		ccqBatchSize       int64
		ccqBackfillCache   bool
		ccqLogger          *zap.Logger
		// Whether the Transfer Verifier should be initialized for this watcher.
		txVerifierEnabled bool
		// Transfer Verifier instance. If nil, transfer verification is disabled.
		txVerifier txverifier.TransferVerifierInterface
	}

	pendingKey struct {
		TxHash         eth_common.Hash
		BlockHash      eth_common.Hash
		EmitterAddress vaa.Address
		Sequence       uint64
	}

	pendingMessage struct {
		message *common.MessagePublication
		height  uint64
	}
)

const (
	// MaxWaitConfirmations is the maximum number of confirmations to wait before declaring a transaction abandoned.
	MaxWaitConfirmations = 60

	// pruneHeightDelta is the block height difference between the latest block and the oldest block to keep in memory.
	// It is used as a parameter for the Transfer Verifier.
	// Value is arbitrary and can be adjusted if it helps performance.
	PruneHeightDelta = uint64(20)
)

func NewEthWatcher(
	url string,
	contract eth_common.Address,
	networkName string,
	chainID vaa.ChainID,
	msgC chan<- *common.MessagePublication,
	setC chan<- *common.GuardianSet,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	queryReqC <-chan *query.PerChainQueryInternal,
	queryResponseC chan<- *query.PerChainQueryResponseInternal,
	env common.Environment,
	ccqBackfillCache bool,
	txVerifierEnabled bool,
) *Watcher {
	// Note: the watcher's txVerifier field is not set here because it requires a Connector as an argument.
	// Instead, it will be populated in `Run()`.
	return &Watcher{
		url:                url,
		contract:           contract,
		networkName:        networkName,
		readinessSync:      common.MustConvertChainIdToReadinessSyncing(chainID),
		chainID:            chainID,
		msgC:               msgC,
		setC:               setC,
		obsvReqC:           obsvReqC,
		queryReqC:          queryReqC,
		queryResponseC:     queryResponseC,
		pending:            map[pendingKey]*pendingMessage{},
		env:                env,
		ccqConfig:          query.GetPerChainConfig(chainID),
		ccqMaxBlockNumber:  big.NewInt(0).SetUint64(math.MaxUint64),
		ccqBackfillCache:   ccqBackfillCache,
		ccqBackfillChannel: make(chan *ccqBackfillRequest, 50),
		// Signals that a transfer Verifier should be instantiated in Run()
		txVerifierEnabled: txVerifierEnabled,
	}
}

func (w *Watcher) tokenBridge() eth_common.Address {
	var tb []byte
	switch w.env {
	case common.UnsafeDevNet, common.AccountantMock, common.GoTest:
		tb = sdk.KnownDevnetTokenbridgeEmitters[w.chainID]
	case common.TestNet:
		tb = sdk.KnownTestnetTokenbridgeEmitters[w.chainID]
	case common.MainNet:
		tb = sdk.KnownTokenbridgeEmitters[w.chainID]
	}
	return eth_common.BytesToAddress(tb)
}

func (w *Watcher) wrappedNative() eth_common.Address {
	var wnative string
	switch w.env {
	case common.UnsafeDevNet, common.AccountantMock, common.GoTest:
		wnative = sdk.KnownDevnetWrappedNativeAddresses[w.chainID]
	case common.TestNet:
		wnative = sdk.KnownTestnetWrappedNativeAddresses[w.chainID]
	case common.MainNet:
		wnative = sdk.KnownWrappedNativeAddress[w.chainID]
	}
	return eth_common.HexToAddress(wnative)
}

func (w *Watcher) Run(parentCtx context.Context) error {
	var err error
	logger := supervisor.Logger(parentCtx)
	w.logger = logger
	w.ccqLogger = logger.With(zap.String("component", "ccqevm"))

	logger.Info("Starting watcher",
		zap.String("watcher_name", "evm"),
		zap.String("url", w.url),
		zap.String("contract", w.contract.String()),
		zap.String("networkName", w.networkName),
		zap.String("chainID", w.chainID.String()),
		zap.String("env", string(w.env)),
		zap.Bool("txVerifier", w.txVerifierEnabled),
	)

	// later on we will spawn multiple go-routines through `RunWithScissors`, i.e. catching panics.
	// If any of them panic, this function will return, causing this child context to be canceled
	// such that the other go-routines can free up resources
	ctx, watcherContextCancelFunc := context.WithCancel(parentCtx)
	defer watcherContextCancelFunc()

	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract.Hex(),
	})

	// Verify that we are connecting to the correct chain.
	if err := w.verifyEvmChainID(ctx, logger, w.url); err != nil {
		return fmt.Errorf("failed to verify evm chain id: %w", err)
	}

	// Connect to the node using the appropriate type of connector.
	{
		var finalizedPollingSupported, safePollingSupported bool
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		w.ethConn, finalizedPollingSupported, safePollingSupported, err = w.createConnector(timeout, w.url)
		cancel()
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf(`failed to create connection to url "%s": %w`, w.url, err)
		}

		// Log the connector details for troubleshooting purposes.
		if finalizedPollingSupported {
			if safePollingSupported {
				w.logger.Info("polling for finalized and safe blocks")
			} else {
				w.logger.Info("polling for finalized blocks, will generate safe blocks")
			}
		} else {
			w.logger.Info("assuming instant finality")
		}

		// Initialize a Transfer Verifier
		if w.txVerifierEnabled {

			// This shouldn't happen as Transfer Verification can
			// only be enabled by passing at least one chainID as a
			// CLI flag to guardiand, but this prevents the code
			// from erroneously setting up a Transfer Verifier or
			// else continuing in state where txVerifierEnabled is
			// true but the actual Transfer Verifier is nil.
			if !slices.Contains(txverifier.SupportedChains(), w.chainID) {
				return errors.New("watcher attempted to create Transfer Verifier but this chainId is not supported")
			}

			var tvErr error
			w.txVerifier, tvErr = txverifier.NewTransferVerifier(
				ctx,
				w.ethConn,
				&txverifier.TVAddresses{
					CoreBridgeAddr:    w.contract,
					TokenBridgeAddr:   w.tokenBridge(),
					WrappedNativeAddr: w.wrappedNative(),
				},
				PruneHeightDelta,
				logger,
			)
			if tvErr != nil {
				return fmt.Errorf("failed to create Transfer Verifier instance: %w", tvErr)
			}
			logger.Info("initialized Transfer Verifier",
				zap.String("watcher_name", "evm"),
				zap.String("url", w.url),
				zap.String("contract", w.contract.String()),
			)
		}
	}

	if w.ccqConfig.TimestampCacheSupported {
		w.ccqTimestampCache = NewBlocksByTimestamp(BTS_MAX_BLOCKS, (w.env == common.UnsafeDevNet))
	}

	errC := make(chan error)

	// Subscribe to new message publications. We don't use a timeout here because the LogPollConnector
	// will keep running. Other connectors will use a timeout internally if appropriate.
	messageC := make(chan *ethabi.AbiLogMessagePublished, 2)
	messageSub, err := w.ethConn.WatchLogMessagePublished(ctx, errC, messageC)
	if err != nil {
		ethConnectionErrors.WithLabelValues(w.networkName, "subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		return fmt.Errorf("failed to subscribe to message publication events: %w", err)
	}
	defer messageSub.Unsubscribe()

	// Fetch initial guardian set
	if err := w.fetchAndUpdateGuardianSet(logger, ctx, w.ethConn); err != nil {
		return fmt.Errorf("failed to request guardian set: %v", err)
	}

	// Poll for guardian set.
	common.RunWithScissors(ctx, errC, "evm_fetch_guardian_set", func(ctx context.Context) error {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := w.fetchAndUpdateGuardianSet(logger, ctx, w.ethConn); err != nil {
					errC <- fmt.Errorf("failed to request guardian set: %v", err)
					return nil
				}
			}
		}
	})

	common.RunWithScissors(ctx, errC, "evm_fetch_objs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case r := <-w.obsvReqC:
				if r.ChainId > math.MaxUint16 {
					logger.Error("chain id for observation request is not a valid uint16",
						zap.Uint32("chainID", r.ChainId),
						zap.String("txID", hex.EncodeToString(r.TxHash)),
					)
					continue
				}
				numObservations, err := w.handleReobservationRequest(
					ctx,
					vaa.ChainID(r.ChainId),
					r.TxHash,
					w.ethConn,
					atomic.LoadUint64(&w.latestFinalizedBlockNumber),
					atomic.LoadUint64(&w.latestSafeBlockNumber),
				)
				if err != nil {
					logger.Error("failed to process observation request",
						zap.Uint32("chainID", r.ChainId),
						zap.String("txID", hex.EncodeToString(r.TxHash)),
						zap.Error(err),
					)
				}

				logger.Info("reobserved transactions",
					zap.Uint32("chainID", r.ChainId),
					zap.String("txID", hex.EncodeToString(r.TxHash)),
					zap.Uint32("numObservations", numObservations),
				)
			}
		}
	})

	if w.ccqConfig.QueriesSupported() {
		w.ccqStart(ctx, errC)
	}

	common.RunWithScissors(ctx, errC, "evm_fetch_messages", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-messageSub.Err():
				ethConnectionErrors.WithLabelValues(w.networkName, "subscription_error").Inc()
				errC <- fmt.Errorf("error while processing message publication subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				return nil
			case ev := <-messageC:
				blockTime, err := w.getBlockTime(ctx, ev.Raw.BlockHash)
				if err != nil {
					ethConnectionErrors.WithLabelValues(w.networkName, "block_by_number_error").Inc()
					if canRetryGetBlockTime(err) {
						go w.waitForBlockTime(ctx, logger, errC, ev)
						continue
					}
					p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					errC <- fmt.Errorf("failed to request timestamp for block %d, hash %s: %w", ev.Raw.BlockNumber, ev.Raw.BlockHash.String(), err)
					return nil
				}

				w.postMessage(ctx, ev, blockTime)
			}
		}
	})

	// Watch headers
	headSink := make(chan *connectors.NewBlock, 100)
	headerSubscription, err := w.ethConn.SubscribeForBlocks(ctx, errC, headSink)
	if err != nil {
		ethConnectionErrors.WithLabelValues(w.networkName, "header_subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}
	defer headerSubscription.Unsubscribe()

	common.RunWithScissors(ctx, errC, "evm_fetch_headers", func(ctx context.Context) error {
		stats := gossipv1.Heartbeat_Network{ContractAddress: w.contract.Hex()}
		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-headerSubscription.Err():
				logger.Error("error while processing header subscription", zap.Error(err))
				ethConnectionErrors.WithLabelValues(w.networkName, "header_subscription_error").Inc()
				errC <- fmt.Errorf("error while processing header subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				return nil
			case ev := <-headSink:
				// These two pointers should have been checked before the event was placed on the channel, but just being safe.
				if ev == nil {
					logger.Error("new header event is nil")
					continue
				}
				if ev.Number == nil {
					logger.Error("new header block number is nil", zap.Stringer("finality", ev.Finality))
					continue
				}

				start := time.Now()
				currentHash := ev.Hash
				logger.Debug("processing new header",
					zap.Stringer("current_block", ev.Number),
					zap.Uint64("block_time", ev.Time),
					zap.Stringer("current_blockhash", currentHash),
					zap.Stringer("finality", ev.Finality),
				)
				readiness.SetReady(w.readinessSync)

				blockNumberU := ev.Number.Uint64()
				if ev.Finality == connectors.Latest {
					atomic.StoreUint64(&w.latestBlockNumber, blockNumberU)
					currentEthHeight.WithLabelValues(w.networkName).Set(float64(blockNumberU))
					stats.Height = int64(blockNumberU) // #nosec G115 -- This conversion is safe indefinitely
					w.updateNetworkStats(&stats)
					w.ccqAddLatestBlock(ev)
					continue
				}

				// The only blocks that get here are safe and finalized.

				if ev.Finality == connectors.Safe {
					atomic.StoreUint64(&w.latestSafeBlockNumber, blockNumberU)
					currentEthSafeHeight.WithLabelValues(w.networkName).Set(float64(blockNumberU))
					stats.SafeHeight = int64(blockNumberU) // #nosec G115 -- This conversion is safe indefinitely
				} else {
					atomic.StoreUint64(&w.latestFinalizedBlockNumber, blockNumberU)
					currentEthFinalizedHeight.WithLabelValues(w.networkName).Set(float64(blockNumberU))
					stats.FinalizedHeight = int64(blockNumberU) // #nosec G115 -- This conversion is safe indefinitely
				}
				w.updateNetworkStats(&stats)

				w.pendingMu.Lock()
				for key, pLock := range w.pending {
					// If this block is safe, only process messages wanting safe.
					// If it's not safe, only process messages wanting finalized.
					if (ev.Finality == connectors.Safe) != (pLock.message.ConsistencyLevel == vaa.ConsistencyLevelSafe) {
						continue
					}

					// Transaction is now ready
					if pLock.height <= blockNumberU {
						msm := time.Now()
						timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
						tx, err := w.ethConn.TransactionReceipt(timeout, eth_common.BytesToHash(pLock.message.TxID))
						queryLatency.WithLabelValues(w.networkName, "transaction_receipt").Observe(time.Since(msm).Seconds())
						cancel()

						// If the node returns an error after waiting expectedConfirmation blocks,
						// it means the chain reorged and the transaction was orphaned. The
						// TransactionReceipt call is using the same websocket connection than the
						// head notifications, so it's guaranteed to be atomic.
						//
						// Check multiple possible error cases - the node seems to return a
						// "not found" error most of the time, but it could conceivably also
						// return a nil tx or rpc.ErrNoResult.
						if tx == nil || errors.Is(err, rpc.ErrNoResult) || (err != nil && err.Error() == "not found") {
							logger.Warn("tx was orphaned",
								zap.String("msgId", pLock.message.MessageIDString()),
								zap.String("txHash", pLock.message.TxIDString()),
								zap.Stringer("blockHash", key.BlockHash),
								zap.Uint64("target_blockNum", pLock.height),
								zap.Stringer("current_blockNum", ev.Number),
								zap.Stringer("finality", ev.Finality),
								zap.Stringer("current_blockHash", currentHash),
								zap.Error(err))
							delete(w.pending, key)
							ethMessagesOrphaned.WithLabelValues(w.networkName, "not_found").Inc()
							continue
						}

						// This should never happen - if we got this far, it means that logs were emitted,
						// which is only possible if the transaction succeeded. We check it anyway just
						// in case the EVM implementation is buggy.
						if tx.Status != 1 {
							logger.Error("transaction receipt with non-success status",
								zap.String("msgId", pLock.message.MessageIDString()),
								zap.String("txHash", pLock.message.TxIDString()),
								zap.Stringer("blockHash", key.BlockHash),
								zap.Uint64("target_blockNum", pLock.height),
								zap.Stringer("current_blockNum", ev.Number),
								zap.Stringer("finality", ev.Finality),
								zap.Stringer("current_blockHash", currentHash),
								zap.Error(err))
							delete(w.pending, key)
							ethMessagesOrphaned.WithLabelValues(w.networkName, "tx_failed").Inc()
							continue
						}

						// Any error other than "not found" is likely transient - we retry next block.
						if err != nil {
							if pLock.height+MaxWaitConfirmations <= blockNumberU {
								// An error from this "transient" case has persisted for more than MaxWaitConfirmations.
								logger.Info("observation timed out",
									zap.String("msgId", pLock.message.MessageIDString()),
									zap.String("txHash", pLock.message.TxIDString()),
									zap.Stringer("blockHash", key.BlockHash),
									zap.Uint64("target_blockNum", pLock.height),
									zap.Stringer("current_blockNum", ev.Number),
									zap.Stringer("finality", ev.Finality),
									zap.Stringer("current_blockHash", currentHash),
								)
								ethMessagesOrphaned.WithLabelValues(w.networkName, "timeout").Inc()
								delete(w.pending, key)
							} else {
								logger.Warn("transaction could not be fetched",
									zap.String("msgId", pLock.message.MessageIDString()),
									zap.String("txHash", pLock.message.TxIDString()),
									zap.Stringer("blockHash", key.BlockHash),
									zap.Uint64("target_blockNum", pLock.height),
									zap.Stringer("current_blockNum", ev.Number),
									zap.Stringer("finality", ev.Finality),
									zap.Stringer("current_blockHash", currentHash),
									zap.Error(err))
							}
							continue
						}

						// It's possible for a transaction to be orphaned and then included in a different block
						// but with the same tx hash. Drop the observation (it will be re-observed and needs to
						// wait for the full confirmation time again).
						if tx.BlockHash != key.BlockHash {
							logger.Info("tx got dropped and mined in a different block; the message should have been reobserved",
								zap.String("msgId", pLock.message.MessageIDString()),
								zap.String("txHash", pLock.message.TxIDString()),
								zap.Stringer("blockHash", key.BlockHash),
								zap.Uint64("target_blockNum", pLock.height),
								zap.Stringer("current_blockNum", ev.Number),
								zap.Stringer("finality", ev.Finality),
								zap.Stringer("current_blockHash", currentHash),
							)
							delete(w.pending, key)
							ethMessagesOrphaned.WithLabelValues(w.networkName, "blockhash_mismatch").Inc()
							continue
						}

						logger.Info("observation confirmed",
							zap.String("msgId", pLock.message.MessageIDString()),
							zap.String("txHash", pLock.message.TxIDString()),
							zap.Stringer("blockHash", key.BlockHash),
							zap.Uint64("target_blockNum", pLock.height),
							zap.Stringer("current_blockNum", ev.Number),
							zap.Stringer("finality", ev.Finality),
							zap.Stringer("current_blockHash", currentHash),
						)
						delete(w.pending, key)

						// Note that `tx` here is actually a receipt
						txHash := eth_common.Hash(pLock.message.TxID)
						pubErr := w.verifyAndPublish(pLock.message, ctx, txHash, tx)
						if pubErr != nil {
							logger.Error("could not publish message: transfer verification failed",
								zap.String("msgId", pLock.message.MessageIDString()),
								zap.String("txHash", txHash.String()),
								zap.Error(pubErr),
							)
						}
					}
				}

				w.pendingMu.Unlock()
				logger.Debug("processed new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("finality", ev.Finality),
					zap.Stringer("current_blockhash", currentHash),
					zap.Duration("took", time.Since(start)),
				)
			}
		}
	})

	// Now that the init is complete, peg readiness. That will also happen when we process a new head, but chains
	// that wait for finality may take a while to receive the first block and we don't want to hold up the init.
	readiness.SetReady(w.readinessSync)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

func (w *Watcher) fetchAndUpdateGuardianSet(
	logger *zap.Logger,
	ctx context.Context,
	ethConn connectors.Connector,
) error {
	msm := time.Now()
	logger.Debug("fetching guardian set")
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	idx, gs, err := fetchCurrentGuardianSet(timeout, ethConn)
	if err != nil {
		ethConnectionErrors.WithLabelValues(w.networkName, "guardian_set_fetch_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		return err
	}

	queryLatency.WithLabelValues(w.networkName, "get_guardian_set").Observe(time.Since(msm).Seconds())

	if w.currentGuardianSet != nil && *(w.currentGuardianSet) == idx {
		return nil
	}

	logger.Info("updated guardian set found", zap.Any("value", gs), zap.Uint32("index", idx))

	w.currentGuardianSet = &idx

	if w.setC != nil {
		w.setC <- common.NewGuardianSet(gs.Keys, idx)
	}

	return nil
}

// Fetch the current guardian set ID and guardian set from the chain.
func fetchCurrentGuardianSet(ctx context.Context, ethConn connectors.Connector) (uint32, *ethabi.StructsGuardianSet, error) {
	currentIndex, err := ethConn.GetCurrentGuardianSetIndex(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	gs, err := ethConn.GetGuardianSet(ctx, currentIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}

	return currentIndex, &gs, nil
}

// getFinality determines if the chain supports "finalized" and "safe". This is hard coded so it requires thought to change something. However, it also reads the RPC
// to make sure the node actually supports the expected values, and returns an error if it doesn't. Note that we do not support using safe mode but not finalized mode.
func (w *Watcher) getFinality(ctx context.Context) (bool, bool, error) {
	finalized, safe, err := GetFinality(w.env, w.chainID)
	if err != nil {
		return false, false, fmt.Errorf("failed to get finality for %s chain %v: %v", w.env, w.chainID, err)
	}

	// If finalized / safe should be supported, read the RPC to make sure they actually are.
	if finalized {
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		c, err := rpc.DialContext(timeout, w.url)
		if err != nil {
			return false, false, fmt.Errorf("failed to connect to endpoint: %w", err)
		}

		type Marshaller struct {
			Number *eth_hexutil.Big
		}
		var m Marshaller

		err = c.CallContext(ctx, &m, "eth_getBlockByNumber", "finalized", false)
		if err != nil || m.Number == nil {
			return false, false, fmt.Errorf("finalized not supported by the node when it should be")
		}

		if safe {
			err = c.CallContext(ctx, &m, "eth_getBlockByNumber", "safe", false)
			if err != nil || m.Number == nil {
				return false, false, fmt.Errorf("safe not supported by the node when it should be")
			}
		}
	}

	return finalized, safe, nil
}

// SetL1Finalizer is used to set the layer one finalizer.
func (w *Watcher) SetL1Finalizer(l1Finalizer interfaces.L1Finalizer) {
	w.l1Finalizer = l1Finalizer
}

// GetLatestFinalizedBlockNumber() implements the L1Finalizer interface and allows other watchers to
// get the latest finalized block number from this watcher.
func (w *Watcher) GetLatestFinalizedBlockNumber() uint64 {
	return atomic.LoadUint64(&w.latestFinalizedBlockNumber)
}

// getLatestSafeBlockNumber() returns the latest safe block seen by this watcher..
func (w *Watcher) getLatestSafeBlockNumber() uint64 {
	return atomic.LoadUint64(&w.latestSafeBlockNumber)
}

func (w *Watcher) updateNetworkStats(stats *gossipv1.Heartbeat_Network) {
	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		Height:          stats.Height,
		SafeHeight:      stats.SafeHeight,
		FinalizedHeight: stats.FinalizedHeight,
		ContractAddress: w.contract.Hex(),
	})
}

// getBlockTime reads the time of a block.
func (w *Watcher) getBlockTime(ctx context.Context, blockHash eth_common.Hash) (uint64, error) {
	msm := time.Now()
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	blockTime, err := w.ethConn.TimeOfBlockByHash(timeout, blockHash)
	cancel()
	queryLatency.WithLabelValues(w.networkName, "block_by_number").Observe(time.Since(msm).Seconds())
	return blockTime, err
}

// postMessage creates a message object from a log event and adds it to the pending list for processing.
func (w *Watcher) postMessage(
	parentCtx context.Context,
	ev *ethabi.AbiLogMessagePublished,
	blockTime uint64,
) {
	msg := &common.MessagePublication{
		TxID:             ev.Raw.TxHash.Bytes(),
		Timestamp:        time.Unix(int64(blockTime), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            ev.Nonce,
		Sequence:         ev.Sequence,
		EmitterChain:     w.chainID,
		EmitterAddress:   PadAddress(ev.Sender),
		Payload:          ev.Payload,
		ConsistencyLevel: ev.ConsistencyLevel,
	}

	ethMessagesObserved.WithLabelValues(w.networkName).Inc()

	if msg.ConsistencyLevel == vaa.ConsistencyLevelPublishImmediately {
		w.logger.Info("found new message publication transaction, publishing it immediately",
			zap.String("msgId", msg.MessageIDString()),
			zap.String("txHash", msg.TxIDString()),
			zap.Uint64("blockNum", ev.Raw.BlockNumber),
			zap.Uint64("latestFinalizedBlock", atomic.LoadUint64(&w.latestFinalizedBlockNumber)),
			zap.Stringer("blockHash", ev.Raw.BlockHash),
			zap.Uint64("blockTime", blockTime),
			zap.Uint32("Nonce", ev.Nonce),
			zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
		)

		verifyCtx, cancel := context.WithCancel(parentCtx)
		defer cancel()

		pubErr := w.verifyAndPublish(msg, verifyCtx, ev.Raw.TxHash, nil)
		if pubErr != nil {
			w.logger.Error("could not publish message: transfer verification failed",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Error(pubErr),
			)
		}
		return
	}

	w.logger.Info("found new message publication transaction",
		zap.String("msgId", msg.MessageIDString()),
		zap.String("txHash", msg.TxIDString()),
		zap.Uint64("blockNum", ev.Raw.BlockNumber),
		zap.Uint64("latestFinalizedBlock", atomic.LoadUint64(&w.latestFinalizedBlockNumber)),
		zap.Stringer("blockHash", ev.Raw.BlockHash),
		zap.Uint64("blockTime", blockTime),
		zap.Uint32("Nonce", ev.Nonce),
		zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
	)

	key := pendingKey{
		TxHash:         eth_common.BytesToHash(msg.TxID),
		BlockHash:      ev.Raw.BlockHash,
		EmitterAddress: msg.EmitterAddress,
		Sequence:       msg.Sequence,
	}

	w.pendingMu.Lock()
	w.pending[key] = &pendingMessage{
		message: msg,
		height:  ev.Raw.BlockNumber,
	}
	w.pendingMu.Unlock()
}

// blockNotFoundErrors is used by `canRetryGetBlockTime`. It is a map of the error returns from `getBlockTime` that can trigger a retry.
var blockNotFoundErrors = map[string]struct{}{
	"not found":                     {},
	"Unknown block":                 {},
	"cannot query unfinalized data": {}, // Seen on Avalanche
}

// canRetryGetBlockTime returns true if the error returned by getBlockTime warrants doing a retry.
func canRetryGetBlockTime(err error) bool {
	_, exists := blockNotFoundErrors[err.Error()]
	return exists
}

// verifyAndPublish validates a MessagePublication to ensure that it's safe. If so, it broadcasts the message. This function
// should be the only location where the watcher's msgC channel is written to.
// Modifies the verificationState field of the message as a side-effect.
// Even if an invalid Transfer is detected, the message will still be published. It is the responsibility of the calling code to handle
// a status of Rejected.
// Note that the result of verification is not returned by this function, but can be accessed directly via the reference to message.
func (w *Watcher) verifyAndPublish(
	// Must be non-nil and have verificationState equal to NotVerified.
	msg *common.MessagePublication,
	ctx context.Context,
	// TODO: in practice it might be possible to read the txHash from the MessagePublication and so this argument might be redundant
	txHash eth_common.Hash,
	// This argument is only used when Transfer Verifier is enabled. If nil, transfer verifier will fetch the receipt.
	// Otherwise, the receipt in the calling context can be passed here to save on RPC requests and parsing.
	receipt *gethTypes.Receipt,
) error {

	if msg == nil {
		return errors.New("verifyAndPublish: message publication cannot be nil")
	}

	if w.txVerifier != nil {
		verifiedMsg, err := verify(ctx, msg, txHash, receipt, w.txVerifier)

		if err != nil {
			return err
		}
		msg = &verifiedMsg
	}

	w.msgC <- msg
	ethMessagesConfirmed.WithLabelValues(w.networkName).Inc()
	if msg.IsReobservation {
		watchers.ReobservationsByChain.WithLabelValues(w.chainID.String(), "std").Inc()
	}
	return nil

}

// waitForBlockTime is a go routine that repeatedly attempts to read the block time for a single log event. It is used when the initial attempt to read
// the block time fails. If it is finally able to read the block time, it posts the event for processing. Otherwise, it will eventually give up.
func (w *Watcher) waitForBlockTime(ctx context.Context, logger *zap.Logger, errC chan error, ev *ethabi.AbiLogMessagePublished) {
	logger.Warn("found new message publication transaction but failed to look up block time, deferring processing",
		zap.String("msgId", msgIdFromLogEvent(w.chainID, ev)),
		zap.Stringer("txHash", ev.Raw.TxHash),
		zap.Uint64("blockNum", ev.Raw.BlockNumber),
		zap.Uint64("latestFinalizedBlock", atomic.LoadUint64(&w.latestFinalizedBlockNumber)),
		zap.Stringer("blockHash", ev.Raw.BlockHash),
		zap.Uint32("Nonce", ev.Nonce),
		zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
	)

	const RetryInterval = 5 * time.Second
	const MaxRetries = 3
	start := time.Now()
	t := time.NewTimer(RetryInterval)
	defer t.Stop()
	retries := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			blockTime, err := w.getBlockTime(ctx, ev.Raw.BlockHash)
			if err == nil {
				logger.Info("retry of block time query succeeded, posting transaction",
					zap.String("msgId", msgIdFromLogEvent(w.chainID, ev)),
					zap.Stringer("txHash", ev.Raw.TxHash),
					zap.Uint64("blockNum", ev.Raw.BlockNumber),
					zap.Stringer("blocHash", ev.Raw.BlockHash),
					zap.Uint64("blockTime", blockTime),
					zap.Uint32("Nonce", ev.Nonce),
					zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
					zap.Stringer("startTime", start),
					zap.Int("retries", retries),
				)

				w.postMessage(ctx, ev, blockTime)
				return
			}

			ethConnectionErrors.WithLabelValues(w.networkName, "block_by_number_error").Inc()
			if !canRetryGetBlockTime(err) {
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				errC <- fmt.Errorf("failed to request timestamp for block %d, hash %s: %w", ev.Raw.BlockNumber, ev.Raw.BlockHash.String(), err)
				return
			}
			if retries >= MaxRetries {
				logger.Error("repeatedly failed to look up block time, giving up",
					zap.String("msgId", msgIdFromLogEvent(w.chainID, ev)),
					zap.Stringer("txHash", ev.Raw.TxHash),
					zap.Uint64("blockNum", ev.Raw.BlockNumber),
					zap.Stringer("blockHash", ev.Raw.BlockHash),
					zap.Uint32("Nonce", ev.Nonce),
					zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
					zap.Stringer("startTime", start),
					zap.Int("retries", retries),
				)

				return
			}

			retries++
			t.Reset(RetryInterval)
		}
	}
}

// msgIdFromLogEvent formats the message ID (chain/emitterAddress/seqNo) from a log event.
func msgIdFromLogEvent(chainID vaa.ChainID, ev *ethabi.AbiLogMessagePublished) string {
	return fmt.Sprintf("%v/%v/%v", uint16(chainID), PadAddress(ev.Sender), ev.Sequence)
}

// createConnector determines the type of connector needed for a chain and creates the appropriate one.
func (w *Watcher) createConnector(ctx context.Context, url string) (ethConn connectors.Connector, finalizedPollingSupported, safePollingSupported bool, err error) {
	finalizedPollingSupported, safePollingSupported, err = w.getFinality(ctx)
	if err != nil {
		err = fmt.Errorf("failed to determine finality: %w", err)
		return
	}

	baseConnector, err := connectors.NewEthereumBaseConnector(ctx, w.networkName, url, w.contract, w.logger)
	if err != nil {
		err = fmt.Errorf("dialing eth client failed: %w", err)
		return
	}

	// We support two types of pollers, the batch poller and the instant finality poller. Instantiate the right one.
	if finalizedPollingSupported {
		ethConn = connectors.NewBatchPollConnector(ctx, w.logger, baseConnector, safePollingSupported, 1000*time.Millisecond)
	} else {
		ethConn = connectors.NewInstantFinalityConnector(baseConnector, w.logger)
	}
	return
}
