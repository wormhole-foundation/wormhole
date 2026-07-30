package solana

import (
	"context"
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/coder/websocket"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// runDataPump is the websocket reader routine for account subscription data.
// It owns no parsing logic; it only moves bytes from the websocket into the watcher channel.
func (s *SolanaWatcher) runDataPump(ctx context.Context, logger *zap.Logger, ws *websocket.Conn) error {
	defer ws.Close(websocket.StatusNormalClosure, "")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if msg, err := s.readWebSocketWithTimeout(ctx, ws); err != nil {
				logger.Error("failed to read from account web socket", zap.Error(err))
				return err
			} else {
				s.pumpData <- msg // Note on channel capacity: Only pauses this watcher
			}
		}
	}
}

// runWatcher is the main Solana watcher routine.
// It coordinates polling, websocket messages, and reobservation requests while delegating deterministic parsing to narrower helpers.
func (s *SolanaWatcher) runWatcher(ctx context.Context, logger *zap.Logger, pollInterval time.Duration, useWs bool, contractAddr string) error {
	timer := time.NewTicker(pollInterval)
	defer timer.Stop()
	useStdPolling := (!s.pollForTx) && (!useWs)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-s.pumpData:
			err := s.processAccountSubscriptionData(ctx, msg, false)
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
				solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "account_subscription_data").Inc()
				s.errC <- err // Note on channel capacity: The watcher will exit anyway
				return err
			}
		case m := <-s.obsvReqC:
			chainId, err := vaa.KnownChainIDFromNumber[uint32](m.ChainId)
			if err != nil {
				logger.Error("invalid chain id for observation request",
					zap.Uint32("chainID", m.ChainId),
					zap.String("txID", hex.EncodeToString(m.TxHash)),
					zap.Error(err),
				)
				continue
			}

			//nolint:contextcheck // Passed via the 's' object instead of as a parameter.
			numObservations, err := s.handleReobservationRequest(chainId, m.TxHash, s.rpcClient)
			if err != nil {
				logger.Error("failed to process observation request",
					zap.Uint32("chainID", m.ChainId),
					zap.String("identifier", base58.Encode(m.TxHash)),
					zap.Error(err),
				)
			} else {
				logger.Info("reobserved transactions",
					zap.Uint32("chainID", m.ChainId),
					zap.String("identifier", base58.Encode(m.TxHash)),
					zap.Uint32("numObservations", numObservations),
				)
			}
		case <-timer.C:
			// Get current slot height
			rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
			start := time.Now()
			slot, err := s.rpcClient.GetSlot(rCtx, s.commitment)
			cancel()
			queryLatency.WithLabelValues(s.networkName, "get_slot", string(s.commitment)).Observe(time.Since(start).Seconds())
			if err != nil {
				p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
				solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "get_slot_error").Inc()
				s.errC <- err // Note on channel capacity: The watcher will exit anyway
				return err
			}

			currentSolanaHeight.WithLabelValues(s.networkName, string(s.commitment)).Set(float64(slot))
			readiness.SetReady(s.readinessSync)
			p2p.DefaultRegistry.SetNetworkStats(s.chainID, &gossipv1.Heartbeat_Network{
				Height:          int64(slot), // #nosec G115 -- This conversion is safe indefinitely
				ContractAddress: contractAddr,
			})

			if logger.Level().Enabled(zapcore.DebugLevel) {
				logger.Debug("fetched current Solana height", zap.Uint64("slot", slot))
			}

			if useStdPolling {
				lastSlot := atomic.LoadUint64(&s.lastSlot)
				if lastSlot == 0 {
					lastSlot = slot - 1
				}

				rangeStart := lastSlot + 1
				rangeEnd := slot

				// Requesting each slot
				for slotIdx := rangeStart; slotIdx <= rangeEnd; slotIdx++ {
					fetchSlot := slotIdx
					common.RunWithScissors(ctx, s.errC, "SolanaWatcherSlotFetcher", func(ctx context.Context) error {
						return s.runSlotFetcher(ctx, logger, fetchSlot)
					})
				}
			}

			atomic.StoreUint64(&s.lastSlot, slot)
		}
	}
}

// transactionProcessor is the runnable that periodically queries for transactions
// involving the core contract. It is separate from runWatcher so transaction-query
// delays do not block slot polling and heartbeat updates.
func (s *SolanaWatcher) transactionProcessor(ctx context.Context) error {
	// Preserve the previous signature across watcher restarts. On a fresh guardian
	// start, initialize from the most recent core-contract transaction.
	if s.pollPrevWormholeSignature.IsZero() {
		var err error
		s.pollPrevWormholeSignature, err = s.getPrevWormholeSignature()
		if err != nil {
			s.logger.Error("failed to get the last wormhole signature on start up", zap.Error(err))
			s.errC <- err
			return err
		}
	}

	s.logger.Info("starting from previous wormhole signature", zap.Stringer("prevSig", s.pollPrevWormholeSignature))

	timer := time.NewTicker(DefaultPollDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			//nolint:contextcheck // Passed via the 's' object instead of as a parameter.
			err := s.processNewTransactions()
			if err != nil {
				s.logger.Error("failed to get transactions", zap.Error(err))
				s.errC <- err
				return err
			}
		}
	}
}

func (s *SolanaWatcher) runSlotFetcher(ctx context.Context, logger *zap.Logger, slot uint64) error {
	s.retryFetchBlock(ctx, logger, slot, 0, false)
	return nil
}

func (s *SolanaWatcher) retryFetchBlock(ctx context.Context, logger *zap.Logger, slot uint64, retry uint, isReobservation bool) {
	ok := s.fetchBlock(ctx, logger, slot, 0, isReobservation)

	if !ok {
		if retry >= maxRetries {
			logger.Error("max retries for block",
				zap.Uint64("slot", slot),
				zap.Uint("retry", retry))
			return
		}

		time.Sleep(retryDelay) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep

		if logger.Level().Enabled(zapcore.DebugLevel) {
			logger.Debug("retrying block",
				zap.Uint64("slot", slot),
				zap.Uint("retry", retry))
		}

		common.RunWithScissors(ctx, s.errC, "retryFetchBlock", func(ctx context.Context) error {
			return s.runRetryFetchBlock(ctx, logger, slot, retry, isReobservation)
		})
	}
}

func (s *SolanaWatcher) runRetryFetchBlock(ctx context.Context, logger *zap.Logger, slot uint64, retry uint, isReobservation bool) error {
	s.retryFetchBlock(ctx, logger, slot, retry+1, isReobservation)
	return nil
}

func (s *SolanaWatcher) runDelayedFetchBlock(ctx context.Context, logger *zap.Logger, slot uint64, emptyRetry uint, isReobservation bool) error {
	time.Sleep(retryDelay) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep
	s.fetchBlock(ctx, logger, slot, emptyRetry+1, isReobservation)
	return nil
}

func (s *SolanaWatcher) runInitialFetchMessageAccount(ctx context.Context, rpcClient solanaRPCClient, acc solana.PublicKey, slot uint64, isReobservation bool, signature solana.Signature) error {
	s.retryFetchMessageAccount(ctx, rpcClient, acc, slot, 0, isReobservation, signature)
	return nil
}

func (s *SolanaWatcher) retryFetchMessageAccount(ctx context.Context, rpcClient solanaRPCClient, acc solana.PublicKey, slot uint64, retry uint, isReobservation bool, signature solana.Signature) {
	_, retryable := s.fetchMessageAccount(ctx, rpcClient, acc, slot, isReobservation, signature)

	if retryable {
		if retry >= maxRetries {
			s.logger.Error("max retries for account",
				zap.Uint64("slot", slot),
				zap.Stringer("account", acc),
				zap.Uint("retry", retry))
			return
		}

		time.Sleep(retryDelay) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep

		s.logger.Info("retrying account",
			zap.Uint64("slot", slot),
			zap.Stringer("account", acc),
			zap.Uint("retry", retry))

		common.RunWithScissors(ctx, s.errC, "retryFetchMessageAccount", func(ctx context.Context) error {
			return s.runRetryFetchMessageAccount(ctx, rpcClient, acc, slot, retry, isReobservation, signature)
		})
	}
}

func (s *SolanaWatcher) runRetryFetchMessageAccount(ctx context.Context, rpcClient solanaRPCClient, acc solana.PublicKey, slot uint64, retry uint, isReobservation bool, signature solana.Signature) error {
	s.retryFetchMessageAccount(ctx, rpcClient, acc, slot, retry+1, isReobservation, signature)
	return nil
}
