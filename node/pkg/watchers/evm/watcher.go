package evm

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/finalizers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"
	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
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
		readinessSync readiness.Component
		// VAA ChainID of the network we're connecting to.
		chainID vaa.ChainID

		// Channel to send new messages to.
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

		// waitForConfirmations indicates if we should wait for the number of confirmations specified by the consistencyLevel in the message.
		// On many of the chains, we already wait for finalized blocks so there is no point in waiting any additional blocks after finality.
		// Therefore this parameter defaults to false. This feature can / should be enabled on chains where we don't wait for finality.
		waitForConfirmations bool

		// maxWaitConfirmations is the maximum number of confirmations to wait before declaring a transaction abandoned. If we are honoring
		// the consistency level (waitForConfirmations is set to true), then we wait maxWaitConfirmations plus the consistency level. This
		// parameter defaults to 60, which should be plenty long enough for most chains. If not, this parameter can be set.
		maxWaitConfirmations uint64

		// Interface to the chain specific ethereum library.
		ethConn       connectors.Connector
		unsafeDevMode bool

		latestFinalizedBlockNumber uint64
		l1Finalizer                interfaces.L1Finalizer

		// These parameters are currently only used for Polygon and should be set via SetRootChainParams()
		rootChainRpc      string
		rootChainContract string
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
	chainID vaa.ChainID,
	msgC chan<- *common.MessagePublication,
	setC chan<- *common.GuardianSet,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	queryReqC <-chan *query.PerChainQueryInternal,
	queryResponseC chan<- *query.PerChainQueryResponseInternal,
	unsafeDevMode bool,
) *Watcher {

	return &Watcher{
		url:                  url,
		contract:             contract,
		networkName:          networkName,
		readinessSync:        common.MustConvertChainIdToReadinessSyncing(chainID),
		waitForConfirmations: false,
		maxWaitConfirmations: 60,
		chainID:              chainID,
		msgC:                 msgC,
		setC:                 setC,
		obsvReqC:             obsvReqC,
		queryReqC:            queryReqC,
		queryResponseC:       queryResponseC,
		pending:              map[pendingKey]*pendingMessage{},
		unsafeDevMode:        unsafeDevMode,
	}
}

func (w *Watcher) Run(parentCtx context.Context) error {
	logger := supervisor.Logger(parentCtx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "evm"),
		zap.String("url", w.url),
		zap.String("contract", w.contract.String()),
		zap.String("networkName", w.networkName),
		zap.String("chainID", w.chainID.String()),
		zap.Bool("unsafeDevMode", w.unsafeDevMode),
	)

	// later on we will spawn multiple go-routines through `RunWithScissors`, i.e. catching panics.
	// If any of them panic, this function will return, causing this child context to be canceled
	// such that the other go-routines can free up resources
	ctx, watcherContextCancelFunc := context.WithCancel(parentCtx)
	defer watcherContextCancelFunc()

	useFinalizedBlocks := ((w.chainID == vaa.ChainIDEthereum || w.chainID == vaa.ChainIDSepolia) && (!w.unsafeDevMode))
	if (w.chainID == vaa.ChainIDKarura || w.chainID == vaa.ChainIDAcala) && (!w.unsafeDevMode) {
		ufb, err := w.getAcalaMode(ctx)
		if err != nil {
			return err
		}

		if ufb {
			useFinalizedBlocks = true
		}
	}

	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contract.Hex(),
	})

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	safeBlocksSupported := false

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
	} else if useFinalizedBlocks {
		if (w.chainID == vaa.ChainIDEthereum || w.chainID == vaa.ChainIDSepolia) && !w.unsafeDevMode {
			safeBlocksSupported = true
			logger.Info("using finalized blocks, will publish safe blocks")
		} else {
			logger.Info("using finalized blocks")
		}

		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizers.NewDefaultFinalizer(), 250*time.Millisecond, true, safeBlocksSupported)
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
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizer, 250*time.Millisecond, false, false)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating block poll connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDNeon && !w.unsafeDevMode {
		if w.l1Finalizer == nil {
			return fmt.Errorf("unable to create neon watcher because the l1 finalizer is not set")
		}
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		finalizer := finalizers.NewNeonFinalizer(logger, w.l1Finalizer)
		pollConnector, err := connectors.NewBlockPollConnector(ctx, baseConnector, finalizer, 250*time.Millisecond, false, false)
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
		if w.l1Finalizer == nil {
			return fmt.Errorf("unable to create arbitrum watcher because the l1 finalizer is not set")
		}
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		finalizer := finalizers.NewArbitrumFinalizer(logger, w.l1Finalizer)
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizer, 250*time.Millisecond, false, false)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating arbitrum connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDOptimism && !w.unsafeDevMode {
		// This only supports Bedrock mode
		useFinalizedBlocks = true
		safeBlocksSupported := true
		logger.Info("using finalized blocks, will publish safe blocks")
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizers.NewDefaultFinalizer(), 250*time.Millisecond, useFinalizedBlocks, safeBlocksSupported)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating optimism connector failed: %w", err)
		}
	} else if w.chainID == vaa.ChainIDPolygon && w.usePolygonCheckpointing() {
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("failed to connect to polygon: %w", err)
		}
		w.ethConn, err = connectors.NewPolygonConnector(ctx,
			baseConnector,
			w.rootChainRpc,
			w.rootChainContract,
		)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("failed to create polygon connector: %w", err)
		}
	} else if w.chainID == vaa.ChainIDBase && !w.unsafeDevMode {
		baseConnector, err := connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
		w.ethConn, err = connectors.NewBlockPollConnector(ctx, baseConnector, finalizers.NewDefaultFinalizer(), 250*time.Millisecond, true, true)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("creating base connector failed: %w", err)
		}
	} else {
		w.ethConn, err = connectors.NewEthereumConnector(timeout, w.networkName, w.url, w.contract, logger)
		if err != nil {
			ethConnectionErrors.WithLabelValues(w.networkName, "dial_error").Inc()
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			return fmt.Errorf("dialing eth client failed: %w", err)
		}
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

	// Track the current block numbers so we can compare it to the block number of
	// the message publication for observation requests.
	var currentBlockNumber uint64
	var currentSafeBlockNumber uint64

	common.RunWithScissors(ctx, errC, "evm_fetch_objs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
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
				safeBlockNumberU := atomic.LoadUint64(&currentSafeBlockNumber)

				timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
				blockNumber, msgs, err := MessageEventsForTransaction(timeout, w.ethConn, w.contract, w.chainID, tx)
				cancel()

				if err != nil {
					logger.Error("failed to process observation request",
						zap.Error(err), zap.String("eth_network", w.networkName),
						zap.String("tx_hash", tx.Hex()))
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
						w.msgC <- msg
						continue
					}

					if msg.ConsistencyLevel == vaa.ConsistencyLevelSafe && safeBlocksSupported {
						if safeBlockNumberU == 0 {
							logger.Error("no safe block number available, ignoring observation request",
								zap.String("eth_network", w.networkName))
							continue
						}

						if blockNumber <= safeBlockNumberU {
							logger.Info("re-observed message publication transaction",
								zap.Stringer("tx", msg.TxHash),
								zap.Stringer("emitter_address", msg.EmitterAddress),
								zap.Uint64("sequence", msg.Sequence),
								zap.Uint64("current_safe_block", safeBlockNumberU),
								zap.Uint64("observed_block", blockNumber),
								zap.String("eth_network", w.networkName),
							)
							w.msgC <- msg
						} else {
							logger.Info("ignoring re-observed message publication transaction",
								zap.Stringer("tx", msg.TxHash),
								zap.Stringer("emitter_address", msg.EmitterAddress),
								zap.Uint64("sequence", msg.Sequence),
								zap.Uint64("current_safe_block", safeBlockNumberU),
								zap.Uint64("observed_block", blockNumber),
								zap.String("eth_network", w.networkName),
							)
						}

						continue
					}

					if blockNumberU == 0 {
						logger.Error("no block number available, ignoring observation request",
							zap.String("eth_network", w.networkName))
						continue
					}

					var expectedConfirmations uint64
					if w.waitForConfirmations {
						expectedConfirmations = uint64(msg.ConsistencyLevel)
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
						w.msgC <- msg
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
	})

	common.RunWithScissors(ctx, errC, "evm_fetch_query_req", func(ctx context.Context) error {
		ccqMaxBlockNumber := big.NewInt(0).SetUint64(math.MaxUint64)
		for {
			select {
			case <-ctx.Done():
				return nil
			case queryRequest := <-w.queryReqC:
				// This can't happen unless there is a programming error - the caller
				// is expected to send us only requests for our chainID.
				if queryRequest.Request.ChainId != w.chainID {
					panic("ccqevm: invalid chain ID")
				}

				switch req := queryRequest.Request.Query.(type) {
				case *query.EthCallQueryRequest:
					block := req.BlockId
					logger.Info("received query request",
						zap.String("eth_network", w.networkName),
						zap.String("block", block),
						zap.Int("numRequests", len(req.CallData)),
						zap.String("component", "ccqevm"),
					)

					timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
					// like https://github.com/ethereum/go-ethereum/blob/master/ethclient/ethclient.go#L610

					var blockMethod string
					var callBlockArg interface{}
					// TODO: try making these error and see what happens
					// 1. 66 chars but not 0x hex
					// 2. 64 chars but not hex
					// 3. bad blocks
					// 4. bad 0x lengths
					// 5. strings that aren't "latest", "safe", "finalized"
					// 6. "safe" on a chain that doesn't support safe
					// etc?
					// I would expect this to trip within this scissor (if at all) but maybe this should get more defensive
					if len(block) == 66 || len(block) == 64 {
						blockMethod = "eth_getBlockByHash"
						// looks like a hash which requires the object parameter
						// https://eips.ethereum.org/EIPS/eip-1898
						// https://docs.alchemy.com/reference/eth-call
						hash := eth_common.HexToHash(block)
						callBlockArg = rpc.BlockNumberOrHash{
							BlockHash:        &hash,
							RequireCanonical: true,
						}
					} else {
						blockMethod = "eth_getBlockByNumber"
						callBlockArg = block
					}

					// EvmCallData contains the details of a single query in the batch.
					type EvmCallData struct {
						to                 eth_common.Address
						data               string
						callTransactionArg map[string]interface{}
						callResult         *eth_hexutil.Bytes
						callErr            error
					}

					// We build two slices. The first is the batch submitted to the RPC call. It contains one entry for each query plus one to query the block.
					// The second is the data associated with each request (but not the block request). The index into both is the index into the request call data.
					batch := []rpc.BatchElem{}
					evmCallData := []EvmCallData{}

					// Add each requested query to the batch.
					for _, callData := range req.CallData {
						// like https://github.com/ethereum/go-ethereum/blob/master/ethclient/ethclient.go#L610
						to := eth_common.BytesToAddress(callData.To)
						data := eth_hexutil.Encode(callData.Data)
						ecd := EvmCallData{
							to:   to,
							data: data,
							callTransactionArg: map[string]interface{}{
								"to":   to,
								"data": data,
							},
							callResult: &eth_hexutil.Bytes{},
						}
						evmCallData = append(evmCallData, ecd)

						batch = append(batch, rpc.BatchElem{
							Method: "eth_call",
							Args: []interface{}{
								ecd.callTransactionArg,
								callBlockArg,
							},
							Result: ecd.callResult,
							Error:  ecd.callErr,
						})
					}

					// Add the block query to the batch.
					var blockResult connectors.BlockMarshaller
					var blockError error
					batch = append(batch, rpc.BatchElem{
						Method: blockMethod,
						Args: []interface{}{
							block,
							false, // no full transaction details
						},
						Result: &blockResult,
						Error:  blockError,
					})

					// Query the RPC.
					err := w.ethConn.RawBatchCallContext(timeout, batch)
					cancel()

					if err != nil {
						logger.Error("failed to process query request",
							zap.Error(err), zap.String("eth_network", w.networkName),
							zap.String("block", block),
							zap.Any("batch", batch),
							zap.String("component", "ccqevm"),
						)
						w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
						continue
					}

					if blockError != nil {
						logger.Error("failed to process query block request",
							zap.Error(blockError), zap.String("eth_network", w.networkName),
							zap.String("block", block),
							zap.Any("batch", batch),
							zap.String("component", "ccqevm"),
						)
						w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
						continue
					}

					if blockResult.Number == nil {
						logger.Error("invalid query block result",
							zap.String("eth_network", w.networkName),
							zap.String("block", block),
							zap.Any("batch", batch),
							zap.String("component", "ccqevm"),
						)
						w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
						continue
					}

					if blockResult.Number.ToInt().Cmp(ccqMaxBlockNumber) > 0 {
						logger.Error("block number too large",
							zap.String("eth_network", w.networkName),
							zap.String("block", block),
							zap.Any("batch", batch),
							zap.String("component", "ccqevm"),
						)
						w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
						continue
					}

					resp := query.EthCallQueryResponse{
						BlockNumber: blockResult.Number.ToInt().Uint64(),
						Hash:        blockResult.Hash,
						Time:        time.Unix(int64(blockResult.Time), 0),
						Results:     [][]byte{},
					}

					errFound := false
					for idx := range req.CallData {
						if evmCallData[idx].callErr != nil {
							logger.Error("failed to process query call request",
								zap.Error(evmCallData[idx].callErr), zap.String("eth_network", w.networkName),
								zap.String("block", block),
								zap.Int("errorIdx", idx),
								zap.Any("batch", batch),
								zap.String("component", "ccqevm"),
							)
							w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
							errFound = true
							break
						}

						// Nil or Empty results are not valid
						// eth_call will return empty when the state doesn't exist for a block
						if len(*evmCallData[idx].callResult) == 0 {
							logger.Error("invalid call result",
								zap.String("eth_network", w.networkName),
								zap.String("block", block),
								zap.Int("errorIdx", idx),
								zap.Any("batch", batch),
								zap.String("component", "ccqevm"),
							)
							w.ccqSendQueryResponse(logger, queryRequest, query.QueryRetryNeeded, nil)
							errFound = true
							break
						}

						logger.Info("query result",
							zap.String("eth_network", w.networkName),
							zap.String("block", block),
							zap.String("blockNumber", blockResult.Number.String()),
							zap.String("blockHash", blockResult.Hash.Hex()),
							zap.String("blockTime", blockResult.Time.String()),
							zap.Int("idx", idx),
							zap.String("to", evmCallData[idx].to.Hex()),
							zap.Any("data", evmCallData[idx].data),
							zap.String("result", evmCallData[idx].callResult.String()),
							zap.String("component", "ccqevm"),
						)

						resp.Results = append(resp.Results, *evmCallData[idx].callResult)
					}

					if !errFound {
						w.ccqSendQueryResponse(logger, queryRequest, query.QuerySuccess, &resp)
					}

				default:
					logger.Warn("received unsupported request type",
						zap.Uint8("payload", uint8(queryRequest.Request.Query.Type())),
						zap.String("component", "ccqevm"),
					)
					w.ccqSendQueryResponse(logger, queryRequest, query.QueryFatalError, nil)
				}
			}
		}
	})

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
					return nil
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
						zap.Uint64("blockTime", blockTime),
						zap.Uint64("Sequence", ev.Sequence),
						zap.Uint32("Nonce", ev.Nonce),
						zap.Uint8("ConsistencyLevel", ev.ConsistencyLevel),
						zap.String("eth_network", w.networkName))

					w.msgC <- message
					ethMessagesConfirmed.WithLabelValues(w.networkName).Inc()
					continue
				}

				logger.Info("found new message publication transaction",
					zap.Stringer("tx", ev.Raw.TxHash),
					zap.Uint64("block", ev.Raw.BlockNumber),
					zap.Stringer("blockhash", ev.Raw.BlockHash),
					zap.Uint64("blockTime", blockTime),
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
	})

	// Watch headers
	headSink := make(chan *connectors.NewBlock, 2)
	headerSubscription, err := w.ethConn.SubscribeForBlocks(ctx, errC, headSink)
	if err != nil {
		ethConnectionErrors.WithLabelValues(w.networkName, "header_subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}

	common.RunWithScissors(ctx, errC, "evm_fetch_headers", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-headerSubscription.Err():
				ethConnectionErrors.WithLabelValues(w.networkName, "header_subscription_error").Inc()
				errC <- fmt.Errorf("error while processing header subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
				return nil
			case ev := <-headSink:
				// These two pointers should have been checked before the event was placed on the channel, but just being safe.
				if ev == nil {
					logger.Error("new header event is nil", zap.String("eth_network", w.networkName))
					continue
				}
				if ev.Number == nil {
					logger.Error("new header block number is nil", zap.String("eth_network", w.networkName), zap.Bool("is_safe_block", ev.Safe))
					continue
				}

				start := time.Now()
				currentHash := ev.Hash
				logger.Debug("processing new header",
					zap.Stringer("current_block", ev.Number),
					zap.Stringer("current_blockhash", currentHash),
					zap.Bool("is_safe_block", ev.Safe),
					zap.String("eth_network", w.networkName))
				currentEthHeight.WithLabelValues(w.networkName).Set(float64(ev.Number.Int64()))
				readiness.SetReady(w.readinessSync)
				p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
					Height:          ev.Number.Int64(),
					ContractAddress: w.contract.Hex(),
				})

				w.pendingMu.Lock()

				blockNumberU := ev.Number.Uint64()
				if ev.Safe {
					atomic.StoreUint64(&currentSafeBlockNumber, blockNumberU)
				} else {
					atomic.StoreUint64(&currentBlockNumber, blockNumberU)
					atomic.StoreUint64(&w.latestFinalizedBlockNumber, blockNumberU)
				}

				for key, pLock := range w.pending {
					// If this block is safe, only process messages wanting safe.
					// If it's not safe, only process messages wanting finalized.
					if safeBlocksSupported {
						if ev.Safe != (pLock.message.ConsistencyLevel == vaa.ConsistencyLevelSafe) {
							continue
						}
					}

					var expectedConfirmations uint64
					if w.waitForConfirmations && !ev.Safe {
						expectedConfirmations = uint64(pLock.message.ConsistencyLevel)
					}

					// Transaction was dropped and never picked up again
					if pLock.height+expectedConfirmations+w.maxWaitConfirmations <= blockNumberU {
						logger.Info("observation timed out",
							zap.Stringer("tx", pLock.message.TxHash),
							zap.Stringer("blockhash", key.BlockHash),
							zap.Stringer("emitter_address", key.EmitterAddress),
							zap.Uint64("sequence", key.Sequence),
							zap.Stringer("current_block", ev.Number),
							zap.Bool("is_safe_block", ev.Safe),
							zap.Stringer("current_blockhash", currentHash),
							zap.String("eth_network", w.networkName),
							zap.Uint64("expectedConfirmations", expectedConfirmations),
							zap.Uint64("maxWaitConfirmations", w.maxWaitConfirmations),
						)
						ethMessagesOrphaned.WithLabelValues(w.networkName, "timeout").Inc()
						delete(w.pending, key)
						continue
					}

					// Transaction is now ready
					if pLock.height+expectedConfirmations <= blockNumberU {
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
								zap.Bool("is_safe_block", ev.Safe),
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
								zap.Bool("is_safe_block", ev.Safe),
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
								zap.Bool("is_safe_block", ev.Safe),
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
								zap.Bool("is_safe_block", ev.Safe),
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
							zap.Bool("is_safe_block", ev.Safe),
							zap.Stringer("current_blockhash", currentHash),
							zap.String("eth_network", w.networkName))
						delete(w.pending, key)
						w.msgC <- pLock.message
						ethMessagesConfirmed.WithLabelValues(w.networkName).Inc()
					}
				}

				w.pendingMu.Unlock()
				logger.Debug("processed new header",
					zap.Stringer("current_block", ev.Number),
					zap.Bool("is_safe_block", ev.Safe),
					zap.Stringer("current_blockhash", currentHash),
					zap.Duration("took", time.Since(start)),
					zap.String("eth_network", w.networkName))
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

	logger.Info("updated guardian set found",
		zap.Any("value", gs), zap.Uint32("index", idx),
		zap.String("eth_network", w.networkName))

	w.currentGuardianSet = &idx

	if w.setC != nil {
		w.setC <- &common.GuardianSet{
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

func (w *Watcher) getAcalaMode(ctx context.Context) (useFinalizedBlocks bool, errRet error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := rpc.DialContext(timeout, w.url)
	if err != nil {
		errRet = fmt.Errorf("failed to connect to url %s to check acala mode: %w", w.url, err)
		return
	}

	// First check to see if polling for finalized blocks is suported.
	type Marshaller struct {
		Number *eth_hexutil.Big
	}

	var m Marshaller
	err = c.CallContext(ctx, &m, "eth_getBlockByNumber", "finalized", false)
	if err == nil {
		useFinalizedBlocks = true
		return
	}

	// If finalized blocks are not supported, then we had better be in safe mode!
	var safe bool
	err = c.CallContext(ctx, &safe, "net_isSafeMode")
	if err != nil {
		errRet = fmt.Errorf("check for safe mode for url %s failed: %w", w.url, err)
		return
	}

	if !safe {
		errRet = fmt.Errorf("url %s does not support finalized blocks and is not using safe mode", w.url)
	}

	return
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

// SetRootChainParams is used to enabled checkpointing (currently only for Polygon). It handles
// if the feature is either enabled or disabled, but ensures the configuration is valid.
func (w *Watcher) SetRootChainParams(rootChainRpc string, rootChainContract string) error {
	if (rootChainRpc == "") != (rootChainContract == "") {
		return fmt.Errorf("if either rootChainRpc or rootChainContract are set, they must both be set")
	}

	w.rootChainRpc = rootChainRpc
	w.rootChainContract = rootChainContract
	return nil
}

func (w *Watcher) usePolygonCheckpointing() bool {
	return w.rootChainRpc != "" && w.rootChainContract != ""
}

// SetWaitForConfirmations is used to override whether we should wait for the number of confirmations specified by the consistencyLevel in the message.
func (w *Watcher) SetWaitForConfirmations(waitForConfirmations bool) {
	w.waitForConfirmations = waitForConfirmations
}

// SetMaxWaitConfirmations is used to override the maximum number of confirmations to wait before declaring a transaction abandoned.
func (w *Watcher) SetMaxWaitConfirmations(maxWaitConfirmations uint64) {
	w.maxWaitConfirmations = maxWaitConfirmations
}

// ccqSendQueryResponse sends an error response back to the query handler.
func (w *Watcher) ccqSendQueryResponse(logger *zap.Logger, req *query.PerChainQueryInternal, status query.QueryStatus, resp *query.EthCallQueryResponse) {
	queryResponse := query.CreatePerChainQueryResponseInternal(req.RequestID, req.RequestIdx, req.Request.ChainId, status, resp)
	select {
	case w.queryResponseC <- queryResponse:
		logger.Debug("published query response error to handler", zap.String("component", "ccqevm"))
	default:
		logger.Error("failed to published query response error to handler", zap.String("component", "ccqevm"))
	}
}
