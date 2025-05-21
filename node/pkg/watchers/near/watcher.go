package near

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var (

	// how long to initially wait between observing a transaction and attempting to process the transaction.
	// To successfully process the transaction, all receipts need to be finalized, which typically only occurs two blocks later or so.
	// transaction processing will be retried with exponential backoff, i.e. transaction may stay in the queue for ca. initialTxProcDelay^(txProcRetry+2) time.
	initialTxProcDelay = time.Second * 3

	blockPollInterval = time.Millisecond * 200

	// this value should be set to the max. amount of transactions in a block such that they can all be processed in parallel.
	workerCountTxProcessing int = 100

	// this value should be set to be greater than the amount of chunks in a NEAR block,
	// such that they can all be fetched in parallel.
	// We're currently seeing ~10 chunks/block, so setting this to 20 conservatively.
	workerChunkFetching int = 20
	queueSize           int = 10_000 // size of the queues for chunk processing as well as transaction processing

	// if watcher falls behind this many blocks, start over. This should be set proportional to `queueSize`
	// such that all transactions from `maxFallBehindBlocks` can easily fit into the queue
	maxFallBehindBlocks uint = 200

	metricsInterval = time.Second * 10 // how often you want health metrics reported

	txProcRetry uint = 4 // how often to retry processing a transaction

	// the maximum span of gaps in the NEAR blockchain we want to support
	// lower values yields better performance, but can lead to missed observations if NEAR has larger gaps.
	// During testing, gaps on NEAR were at most 1 block long.
	nearBlockchainMaxGaps = 5
)

type (
	transactionProcessingJob struct {
		txHash          string
		senderAccountId string
		creationTime    time.Time
		retryCounter    uint
		delay           time.Duration
		isReobservation bool

		// set during processing
		hasWormholeMsg bool // set during processing; whether this transaction emitted a Wormhole message
	}

	Watcher struct {
		mainnet         bool
		wormholeAccount string // name of the Wormhole Account on the NEAR blockchain
		nearRPC         string

		// external channels
		msgC          chan<- *common.MessagePublication   // validated (SECURITY: and only validated!) observations go into this channel
		obsvReqC      <-chan *gossipv1.ObservationRequest // observation requests are coming from this channel
		readinessSync readiness.Component

		// internal queues
		transactionProcessingQueueCounter atomic.Int64
		transactionProcessingQueue        chan *transactionProcessingJob
		chunkProcessingQueue              chan nearapi.ChunkHeader

		// events channels
		eventChanTxProcessedDuration chan time.Duration
		eventChan                    chan eventType

		// Error channel
		errC chan error

		// sub-components
		finalizer Finalizer
		nearAPI   nearapi.NearApi
	}
)

// NewWatcher creates a new Near appid watcher
func NewWatcher(
	nearRPC string,
	wormholeContract string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	mainnet bool,
) *Watcher {
	return &Watcher{
		mainnet:                      mainnet,
		wormholeAccount:              wormholeContract,
		nearRPC:                      nearRPC,
		msgC:                         msgC,
		obsvReqC:                     obsvReqC,
		readinessSync:                common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDNear),
		transactionProcessingQueue:   make(chan *transactionProcessingJob, queueSize),
		chunkProcessingQueue:         make(chan nearapi.ChunkHeader, queueSize),
		eventChanTxProcessedDuration: make(chan time.Duration, 10),
		eventChan:                    make(chan eventType, 10),
	}
}

func newTransactionProcessingJob(txHash string, senderAccountId string, isReobservation bool) *transactionProcessingJob {
	return &transactionProcessingJob{
		txHash,
		senderAccountId,
		time.Now(),
		0,
		initialTxProcDelay,
		isReobservation,
		false,
	}
}

func (e *Watcher) runBlockPoll(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	// As we start, get the height of the latest finalized block. We won't be processing any blocks before that.
	finalBlock, err := e.nearAPI.GetFinalBlock(ctx)
	if err != nil || finalBlock.Header.Height == 0 {
		logger.Error("failed to start NEAR block poll", zap.String("error_type", "startup_fail"), zap.String("log_msg_type", "startup_error"))
		return err
	}

	highestFinalBlockHeightObserved := finalBlock.Header.Height - 1 // minues one because we still want to process this block, just no blocks before it

	timer := time.NewTimer(time.Nanosecond) // this is just for the first iteration.

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-timer.C:
			highestFinalBlockHeightObserved, err = e.ReadFinalChunksSince(logger, ctx, highestFinalBlockHeightObserved, e.chunkProcessingQueue)
			if err != nil {
				logger.Warn("NEAR poll error", zap.String("log_msg_type", "block_poll_error"), zap.String("error", err.Error()))
			}

			if highestFinalBlockHeightObserved > math.MaxInt64 {
				logger.Error("failed to start NEAR block poll", zap.String("error_type", "startup_fail"), zap.String("log_msg_type", "startup_error"))
				return fmt.Errorf("the latest finalised NEAR block heigh is not a valid int64: %d", highestFinalBlockHeightObserved)
			}

			p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
				Height:          int64(highestFinalBlockHeightObserved),
				ContractAddress: e.wormholeAccount,
			})
			readiness.SetReady(e.readinessSync)

			timer.Reset(blockPollInterval)
		}
	}
}

func (e *Watcher) runChunkFetcher(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil

		case chunkHeader := <-e.chunkProcessingQueue:
			newJobs, err := e.fetchAndParseChunk(logger, ctx, chunkHeader)
			if err != nil {
				logger.Warn("near.processChunk failed", zap.String("log_msg_type", "chunk_processing_failed"), zap.String("error", err.Error()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				continue
			}
			for _, job := range newJobs {
				err := e.schedule(ctx, job, job.delay)
				if err != nil {
					// Debug-level logging here because it could be very noisy (one log entry for *any* transaction on the NEAR blockchain)
					logger.Debug("error scheduling transaction processing job", zap.Error(err))
				}
			}
		}
	}
}

func (e *Watcher) runObsvReqProcessor(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != vaa.ChainIDNear {
				panic("invalid chain ID")
			}

			txHash := base58.Encode(r.TxHash)

			logger.Info("Received obsv request", zap.String("log_msg_type", "obsv_req_received"), zap.String("tx_hash", txHash))

			// TODO e.wormholeContract is not the correct value for senderAccountId. Instead, it should be the account id of the transaction sender.
			// This value is used by NEAR to determine which shard to query. An incorrect value here is not a security risk but could lead to reobservation requests failing.
			// Guardians currently run nodes for all shards and the API seems to be returning the correct results independent of the set senderAccountId but this could change in the future.
			// Fixing this would require adding the transaction sender account ID to the observation request.
			job := newTransactionProcessingJob(txHash, e.wormholeAccount, true)
			err := e.schedule(ctx, job, time.Nanosecond)
			if err != nil {
				// Error-level logging here because this is after an re-observation request already, which should be infrequent
				logger.Error("error scheduling transaction processing job", zap.Error(err))
			}
		}
	}
}

func (e *Watcher) runTxProcessor(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case job := <-e.transactionProcessingQueue:
			err := e.processTx(logger, ctx, job)
			if err != nil {
				// transaction processing unsuccessful. Retry if retry_counter not exceeded.
				if job.retryCounter < txProcRetry {
					// Log and retry with exponential backoff
					logger.Debug(
						"near.processTx",
						zap.String("log_msg_type", "tx_processing_retry"),
						zap.String("tx_hash", job.txHash),
						zap.String("error", err.Error()),
					)
					job.retryCounter++
					job.delay *= 2
					err := e.schedule(ctx, job, job.delay)
					if err != nil {
						// Debug-level logging here because it could be very noisy (one log entry for *any* transaction on the NEAR blockchain)
						logger.Debug("error scheduling transaction processing job", zap.Error(err))
					}
				} else {
					// Warn and do not retry
					logger.Warn(
						"near.processTx",
						zap.String("log_msg_type", "tx_processing_retries_exceeded"),
						zap.String("tx_hash", job.txHash),
						zap.String("error", err.Error()),
					)
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				}
			}

			if job.hasWormholeMsg {
				// report how long it took to process this transaction
				e.eventChanTxProcessedDuration <- time.Since(job.creationTime)
			}
		}

	}
}

func (e *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "near"),
		zap.Bool("mainnet", e.mainnet),
		zap.String("wormholeAccount", e.wormholeAccount),
		zap.String("nearRPC", e.nearRPC),
	)

	e.errC = make(chan error)

	e.nearAPI = nearapi.NewNearApiImpl(nearapi.NewHttpNearRpc(e.nearRPC))
	e.finalizer = newFinalizer(e.eventChan, e.nearAPI, e.mainnet)

	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
		ContractAddress: e.wormholeAccount,
	})

	logger.Info("Near watcher connecting to RPC node ", zap.String("url", e.nearRPC))

	// start metrics reporter
	common.RunWithScissors(ctx, e.errC, "metrics", e.runMetrics)
	// start one poller
	common.RunWithScissors(ctx, e.errC, "blockPoll", e.runBlockPoll)
	// start one obsvReqC runner
	common.RunWithScissors(ctx, e.errC, "obsvReqProcessor", e.runObsvReqProcessor)
	// start `workerCount` many chunkFetcher runners
	for i := 0; i < workerChunkFetching; i++ {
		common.RunWithScissors(ctx, e.errC, fmt.Sprintf("chunk_fetcher_%d", i), e.runChunkFetcher)
	}
	// start `workerCount` many transactionProcessing runners
	for i := 0; i < workerCountTxProcessing; i++ {
		common.RunWithScissors(ctx, e.errC, fmt.Sprintf("txProcessor_%d", i), e.runTxProcessor)
	}

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-e.errC:
		return err
	}
}

// schedule pushes a job to workers after delay. It is context aware and will not execute the job if the context
// is cancelled before delay has passed and the job is picked up by a worker.
func (e *Watcher) schedule(ctx context.Context, job *transactionProcessingJob, delay time.Duration) error {
	if int(e.transactionProcessingQueueCounter.Load())+len(e.transactionProcessingQueue) > queueSize {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
		return fmt.Errorf("NEAR transactionProcessingQueue exceeds max queue size. Skipping transaction.")
	}

	common.RunWithScissors(ctx, e.errC, "scheduledThread",
		func(ctx context.Context) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()

			e.transactionProcessingQueueCounter.Add(1)
			defer e.transactionProcessingQueueCounter.Add(-1)

			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				// Don't block on processing if the context is cancelled
				select {
				case <-ctx.Done():
					return nil
				case e.transactionProcessingQueue <- job:
				}
			}
			return nil
		})
	return nil
}
