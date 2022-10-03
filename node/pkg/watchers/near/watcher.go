package near

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	"github.com/certusone/wormhole/node/pkg/watchers/near/timerqueue"
	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	// performance configuration
	workerCountTxProcessing = 10
	workerChunkFetching     = 4      // this value should be set to the amount of chunks in a NEAR block, such that they can all be fetched in parallel
	quequeSize              = 10_000 // size of the queques for chunk processing as well as transaction processing
	maxFallBehindBlocks     = 200    // if watcher falls behind this many blocks, start over
	blockPollInterval       = time.Millisecond * 200
	metricsInterval         = time.Second * 10 // how often you want health metrics reported

	txProcRetry = 4 // how often to retry processing a transaction

	// how long to initially wait between observing a transaction and attempting to process the transaction.
	// To successfully process the transaction, all receipts need to be finalized, which typically only occurs two blocks later or so.
	// transaction processing will be retried with exponential backoff, i.e. transaction may stay in the queque for ca. initialTxProcDelay^(txProcRetry+2) time.
	initialTxProcDelay = time.Second * 3

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

		// set during processing
		hasWormholeMsg         bool   // set during processing; whether this transaction emitted a Wormhole message
		wormholeMsgBlockHeight uint64 // highest block height of a wormhole message in this transaction
	}

	Watcher struct {
		mainnet         bool
		wormholeAccount string // name of the Wormhole Account on the NEAR blockchain
		nearRPC         string

		// external channels
		msgC     chan<- *common.MessagePublication   // validated (SECURITY: and only validated!) observations go into this channel
		obsvReqC <-chan *gossipv1.ObservationRequest // observation requests are coming from this channel

		// internal queques
		transactionProcessingQueue timerqueue.Timerqueue
		chunkProcessingQueue       chan nearapi.ChunkHeader

		// events channels
		eventChanBlockProcessedHeight chan uint64 // whenever a block is processed, post the height here
		eventChanTxProcessedDuration  chan time.Duration
		eventChan                     chan eventType // whenever a messages is confirmed, post true in here

		// sub-components
		finalizer Finalizer
		nearAPI   nearapi.NearAPI
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
		mainnet:                       mainnet,
		wormholeAccount:               wormholeContract,
		nearRPC:                       nearRPC,
		msgC:                          msgC,
		obsvReqC:                      obsvReqC,
		transactionProcessingQueue:    *timerqueue.New(),
		chunkProcessingQueue:          make(chan nearapi.ChunkHeader, quequeSize),
		eventChanBlockProcessedHeight: make(chan uint64, 10),
		eventChanTxProcessedDuration:  make(chan time.Duration, 10),
		eventChan:                     make(chan eventType, 10),
	}
}

func newTransactionProcessingJob(txHash string, senderAccountId string) transactionProcessingJob {
	return transactionProcessingJob{
		txHash,
		senderAccountId,
		time.Now(),
		0,
		initialTxProcDelay,
		false,
		0,
	}
}

func (e *Watcher) runBlockPoll(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	// As we start, get the height of the latest finalized block. We won't be processing any blocks before that.
	finalBlock, err := e.nearAPI.GetFinalBlock(ctx)
	if err != nil || finalBlock.Header.Height == 0 {
		logger.Error("failed to start NEAR block poll", zap.String("log_msg_type", "startup_error"))
		return err
	}

	highestFinalBlockHeightObserved := finalBlock.Header.Height - 1 // minues one because we still want to process this block, just no blocks before it

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	timer := time.NewTimer(time.Nanosecond) // this is just for the first iteration.

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-timer.C:
			highestFinalBlockHeightObserved, err = e.ReadFinalChunksSince(ctx, highestFinalBlockHeightObserved, e.chunkProcessingQueue)
			if err != nil {
				logger.Warn("NEAR poll error", zap.String("log_msg_type", "block_poll_error"), zap.String("error", err.Error()))
			}
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
				logger.Error("near.processChunk failed", zap.String("log_msg_type", "chunk_processing_failed"), zap.String("error", err.Error()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
				continue
			}
			for i := 0; i < len(newJobs); i++ {
				if e.transactionProcessingQueue.Len() > quequeSize {
					logger.Error(
						"NEAR transactionProcessingQueue exceeds max queue size. Skipping transaction.",
						zap.String("log_msg_type", "tx_proc_queue_full"),
						zap.String("chunk_id", chunkHeader.Hash),
					)
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDNear, 1)
					break
				}
				e.transactionProcessingQueue.Schedule(newJobs[i], time.Now().Add(newJobs[i].delay))
			}
		}
	}
}

func (e *Watcher) runObsvReqProcessor(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			if vaa.ChainID(r.ChainId) != vaa.ChainIDNear {
				panic("invalid chain ID")
			}

			txHash := base58.Encode(r.TxHash)

			logger.Info("Received obsv request", zap.String("log_msg_type", "obsv_req_received"), zap.String("tx_hash", txHash))

			// TODO !!!! IMPORTANT !!!! e.wormholeContract is not the correct value for senderAccountId.
			// This may work for now, but eventually we could end up hitting the wrong shard, leading to errors.
			// we also need to know the block height. I think if we don't know it we could just set it to 0, but need to think more about that.
			job := newTransactionProcessingJob(txHash, e.wormholeAccount)
			e.transactionProcessingQueue.Schedule(job, time.Now())
		}
	}
}

func (e *Watcher) runTxProcessor(ctx context.Context) error {
	logger := supervisor.Logger(ctx)
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	ticker := time.NewTicker(time.Microsecond * 100)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			j, _, err := e.transactionProcessingQueue.PopFirstIfReady()
			if err != nil {
				continue
			}

			job := j.(transactionProcessingJob)

			err = e.processTx(logger, ctx, &job)
			if err != nil {
				// transaction processing unsuccessful. Retry if retry_counter not exceeded.

				if job.retryCounter < txProcRetry {
					// Warn and retry with exponential backoff
					logger.Warn(
						"near.processTx",
						zap.String("log_msg_type", "tx_processing_retry"),
						zap.String("tx_hash", job.txHash),
						zap.String("error", err.Error()),
					)
					job.retryCounter++
					job.delay *= 2
					e.transactionProcessingQueue.Schedule(job, time.Now().Add(job.delay))
				} else {
					// Error and do not retry
					logger.Error(
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

			// tell everyone about successful processing
			e.eventChanBlockProcessedHeight <- job.wormholeMsgBlockHeight
		}
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	// TODO should be later?
	readiness.SetReady(common.ReadinessNearSyncing)

	logger := supervisor.Logger(ctx)

	e.nearAPI = nearapi.NewNearAPI(e.nearRPC)
	e.finalizer = newFinalizer(e.eventChan, e.nearAPI, e.mainnet)

	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDNear, &gossipv1.Heartbeat_Network{
		ContractAddress: e.wormholeAccount,
	})

	logger.Info("Near watcher connecting to RPC node ", zap.String("url", e.nearRPC))

	// start metrics reporter
	err := supervisor.Run(ctx, "metrics", e.runMetrics)
	if err != nil {
		return err
	}
	// start one poller
	err = supervisor.Run(ctx, "blockPoll", e.runBlockPoll)
	if err != nil {
		return err
	}
	// start one obsvReqC runner
	err = supervisor.Run(ctx, "obsvReqProcessor", e.runObsvReqProcessor)
	if err != nil {
		return err
	}
	// start `workerCount` many transactionProcessing runners
	for i := 0; i < workerChunkFetching; i++ {
		err := supervisor.Run(ctx, fmt.Sprintf("chunk_fetcher_%d", i), e.runChunkFetcher)
		if err != nil {
			return err
		}
	}
	// start `workerCount` many transactionProcessing runners
	for i := 0; i < workerCountTxProcessing; i++ {
		err := supervisor.Run(ctx, fmt.Sprintf("txProcessor_%d", i), e.runTxProcessor)
		if err != nil {
			return err
		}
	}

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	<-ctx.Done()
	return ctx.Err()
}
