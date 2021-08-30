package ethereum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math/big"
	"sync"
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
		}, []string{"eth_network", "reason"})

	ethMessagesObserved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eth_messages_observed_total",
			Help: "Total number of Eth messages observed (pre-confirmation)",
		}, []string{"eth_network"})
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
		// Address of the Eth contract contract
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

		pending   map[eth_common.Hash]*pendingMessage
		pendingMu sync.Mutex

		// 0 is a valid guardian set, so we need a nil value here
		currentGuardianSet *uint32
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
	setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{
		url:         url,
		contract:    contract,
		networkName: networkName,
		readiness:   readiness,
		chainID:     chainID,
		msgChan:     messageEvents,
		setChan:     setEvents,
		pending:     map[eth_common.Hash]*pendingMessage{}}
}

func (e *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.contract.Hex(),
	})

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	c, err := ethclient.DialContext(timeout, e.url)
	if err != nil {
		ethConnectionErrors.WithLabelValues(e.networkName, "dial_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return fmt.Errorf("dialing eth client failed: %w", err)
	}

	f, err := abi.NewAbiFilterer(e.contract, c)
	if err != nil {
		return fmt.Errorf("could not create wormhole contract filter: %w", err)
	}

	caller, err := abi.NewAbiCaller(e.contract, c)
	if err != nil {
		panic(err)
	}

	// Timeout for initializing subscriptions
	timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Subscribe to new message publications
	messageC := make(chan *abi.AbiLogMessagePublished, 2)
	messageSub, err := f.WatchLogMessagePublished(&bind.WatchOpts{Context: timeout}, messageC, nil)
	if err != nil {
		ethConnectionErrors.WithLabelValues(e.networkName, "subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return fmt.Errorf("failed to subscribe to message publication events: %w", err)
	}

	// Fetch initial guardian set
	if err := e.fetchAndUpdateGuardianSet(logger, ctx, caller); err != nil {
		return fmt.Errorf("failed to request guardian set: %v", err)
	}

	// Poll for guardian set.
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := e.fetchAndUpdateGuardianSet(logger, ctx, caller); err != nil {
					logger.Error("failed updating guardian set",
						zap.Error(err), zap.String("eth_network", e.networkName))
				}
			}
		}
	}()

	errC := make(chan error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-messageSub.Err():
				ethConnectionErrors.WithLabelValues(e.networkName, "subscription_error").Inc()
				errC <- fmt.Errorf("error while processing message publication subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				return
			case ev := <-messageC:
				// Request timestamp for block
				msm := time.Now()
				timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
				b, err := c.BlockByNumber(timeout, big.NewInt(int64(ev.Raw.BlockNumber)))
				cancel()
				queryLatency.WithLabelValues(e.networkName, "block_by_number").Observe(time.Since(msm).Seconds())

				if err != nil {
					ethConnectionErrors.WithLabelValues(e.networkName, "block_by_number_error").Inc()
					p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
					errC <- fmt.Errorf("failed to request timestamp for block %d: %w", ev.Raw.BlockNumber, err)
					return
				}

				messsage := &common.MessagePublication{
					TxHash:           ev.Raw.TxHash,
					Timestamp:        time.Unix(int64(b.Time()), 0),
					Nonce:            ev.Nonce,
					Sequence:         ev.Sequence,
					EmitterChain:     e.chainID,
					EmitterAddress:   PadAddress(ev.Sender),
					Payload:          ev.Payload,
					ConsistencyLevel: ev.ConsistencyLevel,
				}

				logger.Info("found new message publication transaction", zap.Stringer("tx", ev.Raw.TxHash),
					zap.Uint64("block", ev.Raw.BlockNumber), zap.String("eth_network", e.networkName))

				ethMessagesObserved.WithLabelValues(e.networkName).Inc()

				e.pendingMu.Lock()
				e.pending[ev.Raw.TxHash] = &pendingMessage{
					message: messsage,
					height:  ev.Raw.BlockNumber,
				}
				e.pendingMu.Unlock()
			}
		}
	}()

	// Watch headers
	headSink := make(chan *types.Header, 2)
	headerSubscription, err := c.SubscribeNewHead(ctx, headSink)
	if err != nil {
		ethConnectionErrors.WithLabelValues(e.networkName, "header_subscribe_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-headerSubscription.Err():
				ethConnectionErrors.WithLabelValues(e.networkName, "header_subscription_error").Inc()
				errC <- fmt.Errorf("error while processing header subscription: %w", err)
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				return
			case ev := <-headSink:
				start := time.Now()
				logger.Info("processing new header", zap.Stringer("block", ev.Number),
					zap.String("eth_network", e.networkName))
				currentEthHeight.WithLabelValues(e.networkName).Set(float64(ev.Number.Int64()))
				readiness.SetReady(e.readiness)
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:        ev.Number.Int64(),
					ContractAddress: e.contract.Hex(),
				})

				e.pendingMu.Lock()

				blockNumberU := ev.Number.Uint64()
				for hash, pLock := range e.pending {

					// Transaction was dropped and never picked up again
					if pLock.height+4*uint64(pLock.message.ConsistencyLevel) <= blockNumberU {
						logger.Debug("observation timed out", zap.Stringer("tx", pLock.message.TxHash),
							zap.Stringer("block", ev.Number), zap.String("eth_network", e.networkName))
						delete(e.pending, hash)
						continue
					}

					// Transaction is now ready
					if pLock.height+uint64(pLock.message.ConsistencyLevel) <= ev.Number.Uint64() {
						logger.Debug("observation confirmed", zap.Stringer("tx", pLock.message.TxHash),
							zap.Stringer("block", ev.Number), zap.String("eth_network", e.networkName))
						delete(e.pending, hash)
						e.msgChan <- pLock.message
						ethMessagesConfirmed.WithLabelValues(e.networkName).Inc()
					}
				}

				e.pendingMu.Unlock()
				logger.Info("processed new header", zap.Stringer("block", ev.Number),
					zap.Duration("took", time.Since(start)), zap.String("eth_network", e.networkName))
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

func (e *Watcher) fetchAndUpdateGuardianSet(
	logger *zap.Logger,
	ctx context.Context,
	caller *abi.AbiCaller,
) error {
	msm := time.Now()
	logger.Info("fetching guardian set")
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	idx, gs, err := fetchCurrentGuardianSet(timeout, caller)
	if err != nil {
		ethConnectionErrors.WithLabelValues(e.networkName, "guardian_set_fetch_error").Inc()
		p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
		return err
	}

	queryLatency.WithLabelValues(e.networkName, "get_guardian_set").Observe(time.Since(msm).Seconds())

	if e.currentGuardianSet != nil && *(e.currentGuardianSet) == idx {
		return nil
	}

	logger.Info("updated guardian set found",
		zap.Any("value", gs), zap.Uint32("index", idx),
		zap.String("eth_network", e.networkName))

	e.currentGuardianSet = &idx

	if e.setChan != nil {
		e.setChan <- &common.GuardianSet{
			Keys:  gs.Keys,
			Index: idx,
		}
	}

	return nil
}

// Fetch the current guardian set ID and guardian set from the chain.
func fetchCurrentGuardianSet(ctx context.Context, caller *abi.AbiCaller) (uint32, *abi.StructsGuardianSet, error) {
	opts := &bind.CallOpts{Context: ctx}

	currentIndex, err := caller.GetCurrentGuardianSetIndex(opts)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	gs, err := caller.GetGuardianSet(opts, currentIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}

	return currentIndex, &gs, nil
}
