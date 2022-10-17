package evm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/finalizers"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
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
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_eth_query_latency",
			Help: "Latency histogram for Ethereum calls (note that most interactions are streaming queries, NOT calls, and we cannot measure latency for those",
		}, []string{"eth_network", "operation"})
)

type (
	Watcher struct {
		// Ethereum RPC url
		url string
		// Address of the Eth contract
		contract eth_common.Address
		// Human-readable name of the Eth network, for logging and monitoring.
		networkName string
		// Readiness component
		readiness readiness.Component
		// VAA ChainID of the network we're connecting to.
		chainID vaa.ChainID

		// Channel to send new messages to.
		msgChan chan *common.MessagePublication

		// Channel to send guardian set changes to.
		// setChan can be set to nil if no guardian set changes are needed.
		//
		// We currently only fetch the guardian set from one primary chain, which should
		// have this flag set to true, and false on all others.
		//
		// The current primary chain is Ethereum (a mostly arbitrary decision because it
		// has the best API - we might want to switch the primary chain to Solana once
		// the governance mechanism lives there),
		setChan chan *common.GuardianSet

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC chan *gossipv1.ObservationRequest

		pending   map[pendingKey]*pendingMessage
		pendingMu sync.Mutex

		// 0 is a valid guardian set, so we need a nil value here
		currentGuardianSet *uint32

		// Minimum number of confirmations to accept, regardless of what the contract specifies.
		minConfirmations uint64

		// Interface to the chain specific ethereum library.
		ethConn             connectors.Connector
		shouldCheckSafeMode bool
		unsafeDevMode       bool
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

func NewEthWatcher(
	url string,
	contract eth_common.Address,
	networkName string,
	readiness readiness.Component,
	chainID vaa.ChainID,
	messageEvents chan *common.MessagePublication,
	setEvents chan *common.GuardianSet,
	minConfirmations uint64,
	obsvReqC chan *gossipv1.ObservationRequest,
	unsafeDevMode bool) *Watcher {

	return &Watcher{
		url:                 url,
		contract:            contract,
		networkName:         networkName,
		readiness:           readiness,
		minConfirmations:    minConfirmations,
		chainID:             chainID,
		msgChan:             messageEvents,
		setChan:             setEvents,
		obsvReqC:            obsvReqC,
		pending:             map[pendingKey]*pendingMessage{},
		shouldCheckSafeMode: (chainID == vaa.ChainIDKarura || chainID == vaa.ChainIDAcala) && (!unsafeDevMode),
		unsafeDevMode:       unsafeDevMode,
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	if w.shouldCheckSafeMode && !w.useFinalizedBlocks() {
		if err := w.checkForSafeMode(ctx); err != nil {
			return err
		}
	}

	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract.Hex(),
	})

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var err error
	if w.chainID == vaa.ChainIDCelo && !w.unsafeDevMode {
		// When we are running in mainnet or testnet, we need to use the Celo ethereum library rather than go-ethereum.
		// However, in devnet, we currently run the standard ETH node for Celo, so we need to use the standard go-ethereum.
		w.ethConn, err = connectors.NewCeloConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
	} else if w.useFinalizedBlocks() {
		logger.Info("using finalized blocks")
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizers.NewDefaultFinalizer(), 250*time.Millisecond, true)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating block poll connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDMoonbeam && !w.unsafeDevMode {
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		finalizer := finalizers.NewMoonbeamFinalizer(logger, baseConnector)
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizer, 250*time.Millisecond, false)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating block poll connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDNeon {
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		pollConnector, err := connectors.NewBlockPollConnector(ctx, baseConnector, finalizers.NewDefaultFinalizer(), 250*time.Millisecond, false)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating block poll connector failed: %w", err)
		}
		w.ethConn, err = connectors.NewLogPollConnector(ctx, pollConnector, baseConnector.Client())
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating poll connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDArbitrum && !w.unsafeDevMode {
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		finalizer := finalizers.NewArbitrumFinalizer(logger, baseConnector, baseConnector.Client())
		pollConnector, err := connectors.NewBlockPollConnector(ctx, baseConnector, finalizer, 250*time.Millisecond, false)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating block poll connector failed: %w", err)
		}
		w.ethConn, err = connectors.NewArbitrumConnector(ctx, pollConnector)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating arbitrum connector failed: %w", err)
		}
	} else {
		w.ethConn, err = connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
	}

	// Subscribe to new message publications. We don't use a timeout here because the LogPollConnector
	// will keep running. Other connectors will use a timeout internally if appropriate.
	messageC := make(chan *ethabi.AbiLogMessagePublished, 2)
	messageSub, err := w.ethConn.WatchLogMessagePublished(ctx, messageC)
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

	errC := make(chan error)

	// Poll for guardian set.
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := w.fetchAndUpdateGuardianSet(logger, ctx, w.ethConn); err != nil {
					errC <- fmt.Errorf("failed to request guardian set: %v", err)
					return
				}
			}
		}
	}()

	// Track the current block number so we can compare it to the block number of
	// the message publication for observation requests.
	var currentBlockNumber uint64

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-w.obsvReqC:
				// This can't happen unless there is a programming error - the caller
				// is expected to send us only requests for our chainID.
				if vaa.ChainID(r.ChainId) != w.chainID {
					panic("invalid chain ID")
				}

				tx := eth_common.BytesToHash(r.TxHash)
				logger.Info("received observation request",
					zap.String("eth_network", w.networkName),
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
					logger.Error("no block number available, ignoring observation request",
						zap.String("eth_network", w.networkName))
					continue
				}

				timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
				blockNumber, msgs, err := MessageEventsForTransaction(timeout, w.ethConn, w.contract, w.chainID, tx)
				cancel()

				if err != nil {
					logger.Error("failed to process observation request",
						zap.Error(err), zap.String("eth_network", w.networkName))
					continue
				}

				for _, msg := range msgs {
					if msg.ConsistencyLevel == vaa.ConsistencyLevelPublishImmediately {
						logger.Info("re-observed message publication transaction, publishing it immediately",
							zap.Stringer("tx", msg.TxHash),
							zap.Stringer("emitter_address", msg.EmitterAddress),
							zap.Uint64("sequence", msg.Sequence),
							zap.Uint64("current_block", blockNumberU),
							zap.Uint64("observed_block", blockNumber),
							zap.String("eth_network", w.networkName),
						)
						w.msgChan <- msg
						continue
					}

					expectedConfirmations := uint64(msg.ConsistencyLevel)
					if expectedConfirmations < w.minConfirmations {
						expectedConfirmations = w.minConfirmations
					}

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
					if blockNumber+expectedConfirmations <= blockNumberU {
						logger.Info("re-observed message publication transaction",
							zap.Stringer("tx", msg.TxHash),
							zap.Stringer("emitter_address", msg.EmitterAddress),
							zap.Uint64("sequence", msg.Sequence),
							zap.Uint64("current_block", blockNumberU),
							zap.Uint64("observed_block", blockNumber),
							zap.String("eth_network", w.networkName),
						)
						w.msgChan <- msg
					} else {
						logger.Info("ignoring re-observed message publication transaction",
							zap.Stringer("tx", msg.TxHash),
							zap.Stringer("emitter_address", msg.EmitterAddress),
							zap.Uint64("sequence", msg.Sequence),
							zap.Uint64("current_block", blockNumberU),
							zap.Uint64("observed_block", blockNumber),
							zap.Uint64("expected_confirmations", expectedConfirmations),
							zap.String("eth_network", w.networkName),
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
			case err := <-messageSub.Err():
				ethConnectionErrors.WithLabelValues(w.networkName, "subscription_error").Inc()
				errC <- fmt.Errorf("error while processing message publication subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				return
			case ev := <-messageC:
				// Request timestamp for block
				msm := time.Now()
				timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
				blockTime, err := w.ethConn.TimeOfBlockByHash(timeout, ev.Raw.BlockHash)
				cancel()
				queryLatency.WithLabelValues(w.networkName, "block_by_number").Observe(time.Since(msm).Seconds())

				if err != nil {
					ethConnectionErrors.WithLabelValues(w.networkName, "block_by_number_error").Inc()
					p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					errC <- fmt.Errorf("failed to request timestamp for block %d, hash %s: %w",
						ev.Raw.BlockNumber, ev.Raw.BlockHash.String(), err)
					return
				}

				message := &common.MessagePublication{
					TxHash:           ev.Raw.TxHash,
					Timestamp:        time.Unix(int64(blockTime), 0),
					Nonce:            ev.Nonce,
					Sequence:         ev.Sequence,
					EmitterChain:     w.chainID,
					EmitterAddress:   PadAddress(ev.Sender),
					Payload:          ev.Payload,
					ConsistencyLevel: ev.ConsistencyLevel,
				}

				ethMessagesObserved.WithLabelValues(w.networkName).Inc()

				if message.ConsistencyLevel == vaa.ConsistencyLevelPublishImmediately {
					logger.Info("found new message publication transaction, publishing it immediately",
						zap.Stringer("tx", ev.Raw.TxHash),
						zap.Uint64("block", ev.Raw.BlockNumber),
						zap.Stringer("blockhash", ev.Raw.BlockHash),
						zap.Uint64("Sequence", ev.Sequence),
						zap.Uint32("Nonce", ev.Nonce),
						zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
						zap.String("eth_network", w.networkName))

					w.msgChan <- message
					ethMessagesConfirmed.WithLabelValues(w.networkName).Inc()
					continue
				}

				logger.Info("found new message publication transaction",
					zap.Stringer("tx", ev.Raw.TxHash),
					zap.Uint64("block", ev.Raw.BlockNumber),
					zap.Stringer("blockhash", ev.Raw.BlockHash),
					zap.Uint64("Sequence", ev.Sequence),
					zap.Uint32("Nonce", ev.Nonce),
					zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
					zap.String("eth_network", w.networkName))

				key := pendingKey{
					TxHash:         message.TxHash,
					BlockHash:      ev.Raw.BlockHash,
					EmitterAddress: message.EmitterAddress,
					Sequence:       message.Sequence,
				}

				w.pendingMu.Lock()
				w.pending[key] = &pendingMessage{
					message: message,
					height:  ev.Raw.BlockNumber,
				}
				w.pendingMu.Unlock()
			}
		}
	}()

	// Watch headers
	headSink := make(chan *connectors.NewBlock, 2)
	headerSubscription, err := w.ethConn.SubscribeForBlocks(ctx, headSink)
	if err != nil {
		ethConnectionErrors.WithLabelValues(w.networkName, "header_subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-headerSubscription.Err():
				ethConnectionErrors.WithLabelValues(w.networkName, "header_subscription_error").Inc()
				errC <- fmt.Errorf("error while processing header subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				return
			case ev := <-headSink:
				// These two pointers should have been checked before the event was placed on the channel, but just being safe.
				if ev == nil {
					logger.Error("new header event is nil", zap.String("eth_network", w.networkName))
					continue
				}
				if ev.Number == nil {
					logger.Error("new header block number is nil", zap.String("eth_network", w.networkName))
					continue
				}

				start := time.Now()
				currentHash := ev.Hash
				logger.Info("processing new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("current_blockhash", currentHash),
					zap.String("eth_network", w.networkName))
				currentEthHeight.WithLabelValues(w.networkName).Set(float64(ev.Number.Int64()))
				readiness.SetReady(w.readiness)
				p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
					Height:          ev.Number.Int64(),
					ContractAddress: w.contract.Hex(),
				})

				w.pendingMu.Lock()

				blockNumberU := ev.Number.Uint64()
				atomic.StoreUint64(&currentBlockNumber, blockNumberU)

				for key, pLock := range w.pending {
					expectedConfirmations := uint64(pLock.message.ConsistencyLevel)
					if expectedConfirmations < w.minConfirmations {
						expectedConfirmations = w.minConfirmations
					}

					// Transaction was dropped and never picked up again
					if pLock.height+4*uint64(expectedConfirmations) <= blockNumberU {
						logger.Info("observation timed out",
							zap.Stringer("tx", pLock.message.TxHash),
							zap.Stringer("blockhash", key.BlockHash),
							zap.Stringer("emitter_address", key.EmitterAddress),
							zap.Uint64("sequence", key.Sequence),
							zap.Stringer("current_block", ev.Number),
							zap.Stringer("current_blockhash", currentHash),
							zap.String("eth_network", w.networkName),
						)
						ethMessagesOrphaned.WithLabelValues(w.networkName, "timeout").Inc()
						delete(w.pending, key)
						continue
					}

					// Transaction is now ready
					if pLock.height+uint64(expectedConfirmations) <= blockNumberU {
						timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
						tx, err := w.ethConn.TransactionReceipt(timeout, pLock.message.TxHash)
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
								zap.Stringer("tx", pLock.message.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("emitter_address", key.EmitterAddress),
								zap.Uint64("sequence", key.Sequence),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.String("eth_network", w.networkName),
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
								zap.Stringer("tx", pLock.message.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("emitter_address", key.EmitterAddress),
								zap.Uint64("sequence", key.Sequence),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.String("eth_network", w.networkName),
								zap.Error(err))
							delete(w.pending, key)
							ethMessagesOrphaned.WithLabelValues(w.networkName, "tx_failed").Inc()
							continue
						}

						// Any error other than "not found" is likely transient - we retry next block.
						if err != nil {
							logger.Warn("transaction could not be fetched",
								zap.Stringer("tx", pLock.message.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("emitter_address", key.EmitterAddress),
								zap.Uint64("sequence", key.Sequence),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.String("eth_network", w.networkName),
								zap.Error(err))
							continue
						}

						// It's possible for a transaction to be orphaned and then included in a different block
						// but with the same tx hash. Drop the observation (it will be re-observed and needs to
						// wait for the full confirmation time again).
						if tx.BlockHash != key.BlockHash {
							logger.Info("tx got dropped and mined in a different block; the message should have been reobserved",
								zap.Stringer("tx", pLock.message.TxHash),
								zap.Stringer("blockhash", key.BlockHash),
								zap.Stringer("emitter_address", key.EmitterAddress),
								zap.Uint64("sequence", key.Sequence),
								zap.Stringer("current_block", ev.Number),
								zap.Stringer("current_blockhash", currentHash),
								zap.String("eth_network", w.networkName))
							delete(w.pending, key)
							ethMessagesOrphaned.WithLabelValues(w.networkName, "blockhash_mismatch").Inc()
							continue
						}

						logger.Info("observation confirmed",
							zap.Stringer("tx", pLock.message.TxHash),
							zap.Stringer("blockhash", key.BlockHash),
							zap.Stringer("emitter_address", key.EmitterAddress),
							zap.Uint64("sequence", key.Sequence),
							zap.Stringer("current_block", ev.Number),
							zap.Stringer("current_blockhash", currentHash),
							zap.String("eth_network", w.networkName))
						delete(w.pending, key)
						w.msgChan <- pLock.message
						ethMessagesConfirmed.WithLabelValues(w.networkName).Inc()
					}
				}

				w.pendingMu.Unlock()
				logger.Info("processed new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("current_blockhash", currentHash),
					zap.Duration("took", time.Since(start)),
					zap.String("eth_network", w.networkName))
			}
		}
	}()

	// Now that the init is complete, peg readiness. That will also happen when we process a new head, but chains
	// that wait for finality may take a while to receive the first block and we don't want to hold up the init.
	readiness.SetReady(w.readiness)

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
	logger.Info("fetching guardian set")
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

	logger.Info("updated guardian set found",
		zap.Any("value", gs), zap.Uint32("index", idx),
		zap.String("eth_network", w.networkName))

	w.currentGuardianSet = &idx

	if w.setChan != nil {
		w.setChan <- &common.GuardianSet{
			Keys:  gs.Keys,
			Index: idx,
		}
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

func (w *Watcher) checkForSafeMode(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := rpc.DialContext(timeout, w.url)
	if err != nil {
		return fmt.Errorf("failed to connect to url %s to check for safe mode: %w", w.url, err)
	}

	var safe bool
	err = c.CallContext(ctx, &safe, "net_isSafeMode")
	if err != nil {
		return fmt.Errorf("check for safe mode for url %s failed: %w", w.url, err)
	}

	if !safe {
		return fmt.Errorf("url %s is not using safe mode", w.url)
	}

	return nil
}

func (w *Watcher) useFinalizedBlocks() bool {
	if w.unsafeDevMode {
		return false
	}
	return w.chainID == vaa.ChainIDEthereum || w.chainID == vaa.ChainIDKarura || w.chainID == vaa.ChainIDAcala
}
