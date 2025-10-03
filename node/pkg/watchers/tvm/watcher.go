package tvm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"

	"go.uber.org/zap"
)

type Watcher struct {
	chainID         vaa.ChainID
	tonConfigURL    string
	CurrentHeight   uint32
	contractAddress *address.Address
	LastProcessedLT uint64                              // Last processed Logical Time (LT) of a transaction
	msgChan         chan<- *common.MessagePublication   // The following is the channel for emitting observations
	obsvReqC        <-chan *gossipv1.ObservationRequest // The following is the channel for receiving re-observation requests
	readinessSync   readiness.Component                 // Used to report the health of the watcher
	Subscriber      *TxSubscriber
}

func NewWatcher(
	chainID vaa.ChainID,
	tonConfigURL string,
	lastLT uint64,
	contractAddress *address.Address,
	msgChan chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) *Watcher {
	return &Watcher{
		chainID:         chainID,
		tonConfigURL:    tonConfigURL,
		LastProcessedLT: lastLT,
		msgChan:         msgChan,
		obsvReqC:        obsvReqC,
		contractAddress: contractAddress,
		readinessSync:   common.MustConvertChainIdToReadinessSyncing(chainID),
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	var err error

	logger := supervisor.Logger(ctx)

	p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: w.contractAddress.String(),
	})

	logger.Info("Starting watcher",
		zap.String("watcher_name", "ton"),
		zap.String("networkID", w.chainID.String()),
	)

	outChan := make(chan *tlb.Transaction)

	w.Subscriber, err = NewTxSubscriber(w.contractAddress, w.LastProcessedLT, w.tonConfigURL, outChan, logger)
	if err != nil {
		return fmt.Errorf("failed to create tx subscriber: %w", err)
	}

	if w.LastProcessedLT == 0 {
		w.LastProcessedLT, err = w.GetCoreAccountLastLT(ctx)
		if err != nil {
			return fmt.Errorf("failed to get last LT: %w", err)
		}
	}

	errC := make(chan error)

	go func() {
		err = w.Subscriber.Work(ctx)
		if err != nil {
			logger.Error("failed to start subscriber", zap.Error(err))
			p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
			errC <- err //nolint:channelcheck // The watcher will exit anyway
		}
	}()

	//Timer for the get_block_height go routine
	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	readiness.SetReady(w.readinessSync)

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	common.RunWithScissors(ctx, errC, "ton_core_events", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("coreEvents context done")
				return ctx.Err()
			case tx := <-w.Subscriber.outChan:
				logger.Info("TON transaction received",
					zap.String("chainID", w.chainID.String()),
					zap.String("component", "TxSubscriber"),
					zap.String("address", string(tx.AccountAddr)),
					zap.String("tx_hash", hex.EncodeToString(tx.Hash)),
					zap.Uint64("lt", tx.LT),
					zap.Uint32("now", tx.Now),
				)
				err = w.inspectBody(logger, tx, false)
				if err != nil {
					p2p.DefaultRegistry.AddErrorCount(w.chainID, 1)
					errC <- err //nolint:channelcheck // The watcher will exit anyway
					return err
				}
			}
		}
	})

	common.RunWithScissors(ctx, errC, "ton_block_height", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("ton_block_height context done")
				return ctx.Err()

			case <-timer.C:
				height, err := w.GetLastMasterchainBlockSeqno(ctx)
				if err != nil {
					logger.Error("Failed to get latest seqno", zap.Error(err))
				} else {
					// currentHeight.Set(float64(height))
					logger.Debug("ton_getLatestSeqno", zap.Int64("result", int64(height)))

					p2p.DefaultRegistry.SetNetworkStats(w.chainID, &gossipv1.Heartbeat_Network{
						Height:          int64(height),
						ContractAddress: w.contractAddress.String(),
					})
					w.CurrentHeight = height
				}

				readiness.SetReady(w.readinessSync)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "ton_fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("ton_fetch_obvs_req context done")
				return ctx.Err()
			case r := <-w.obsvReqC:
				if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != vaa.ChainIDTON {
					panic("invalid chain ID")
				}

				txData, err := w.GetTransactionByReobserveRequest(ctx, r.TxHash)
				if err != nil {
					logger.Error("Failed to get transaction by reobserve", zap.Error(err))
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTON, 1)
					return fmt.Errorf("failed to get transaction by reobserve: %w", err)
				}

				err = w.inspectBody(logger, txData, true)
				if err != nil {
					logger.Info("ton_fetch_obvs_req skipping event data in result", zap.Error(err))
				}
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errC:
		return err
	}
}
