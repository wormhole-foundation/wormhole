package solana

import (
	"context"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/readiness"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mr-tron/base58"
	"github.com/near/borsh-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"time"
)

type SolanaWatcher struct {
	bridge       solana.PublicKey
	wsUrl        string
	rpcUrl       string
	messageEvent chan *common.MessagePublication
}

var (
	solanaConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_connection_errors_total",
			Help: "Total number of Solana connection errors",
		}, []string{"reason"})
	solanaAccountSkips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_account_updates_skipped_total",
			Help: "Total number of account updates skipped due to invalid data",
		}, []string{"reason"})
	solanaMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_solana_observations_confirmed_total",
			Help: "Total number of verified Solana observations found",
		})
	currentSolanaHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_solana_current_height",
			Help: "Current Solana slot height (at default commitment level, not the level used for observations)",
		})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_solana_query_latency",
			Help: "Latency histogram for Solana RPC calls",
		}, []string{"operation", "commitment"})
)

const rpcTimeout = time.Second * 5

func NewSolanaWatcher(wsUrl, rpcUrl string, bridgeAddress solana.PublicKey, messageEvents chan *common.MessagePublication) *SolanaWatcher {
	return &SolanaWatcher{bridge: bridgeAddress, wsUrl: wsUrl, rpcUrl: rpcUrl, messageEvent: messageEvents}
}

func (s *SolanaWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	bridgeAddr := base58.Encode(s.bridge[:])
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
		BridgeAddress: bridgeAddr,
	})

	rpcClient := rpc.New(s.rpcUrl)
	logger := supervisor.Logger(ctx)
	errC := make(chan error)
	var lastSlot uint64

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				commitment := rpc.CommitmentFinalized

				// Get current slot height
				rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
				defer cancel()
				start := time.Now()
				slot, err := rpcClient.GetSlot(rCtx, commitment)
				queryLatency.WithLabelValues("get_slot", string(commitment)).Observe(time.Since(start).Seconds())
				if err != nil {
					solanaConnectionErrors.WithLabelValues("get_slot_error").Inc()
					errC <- err
					return
				}
				if lastSlot == 0 {
					lastSlot = slot - 1
				}
				currentSolanaHeight.Set(float64(slot))
				readiness.SetReady(common.ReadinessSolanaSyncing)
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
					Height:        int64(slot),
					BridgeAddress: bridgeAddr,
				})
				logger.Info("fetched current Solana height",
					zap.String("commitment", string(commitment)),
					zap.Uint64("slot", slot),
					zap.Uint64("lastSlot", lastSlot),
					zap.Uint64("pendingSlots", slot-lastSlot),
					zap.Duration("took", time.Since(start)))

				// Determine which slots we're missing
				//
				// Get list of confirmed blocks since the last request. The result
				// won't contain skipped slots.
				rangeStart := lastSlot + 1
				rangeEnd := slot
				rCtx, cancel = context.WithTimeout(ctx, rpcTimeout)
				defer cancel()
				start = time.Now()
				slots, err := rpcClient.GetConfirmedBlocks(rCtx, rangeStart, &rangeEnd, commitment)
				queryLatency.WithLabelValues("get_confirmed_blocks", string(commitment)).Observe(time.Since(start).Seconds())
				if err != nil {
					solanaConnectionErrors.WithLabelValues("get_confirmed_blocks_error").Inc()
					errC <- err
					return
				}

				logger.Info("fetched slots in range",
					zap.Uint64("from", rangeStart), zap.Uint64("to", rangeEnd),
					zap.Duration("took", time.Since(start)),
					zap.String("commitment", string(commitment)))

				// Requesting each slot
				for _, slot := range slots {
					if slot <= lastSlot {
						// Skip out-of-range result
						// https://github.com/solana-labs/solana/issues/18946
						continue
					}

					go s.fetchBlock(ctx, logger, commitment, rpcClient, slot)
				}

				lastSlot = slot
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

func (s *SolanaWatcher) fetchBlock(ctx context.Context, logger *zap.Logger, commitment rpc.CommitmentType, rpcClient *rpc.Client, slot uint64) {
	logger.Debug("requesting block", zap.Uint64("slot", slot), zap.String("commitment", string(commitment)))
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	rewards := false
	out, err := rpcClient.GetConfirmedBlockWithOpts(rCtx, slot, &rpc.GetConfirmedBlockOpts{
		Encoding:           "json",
		TransactionDetails: "full",
		Rewards:            &rewards,
		Commitment:         commitment,
	})

	queryLatency.WithLabelValues("get_confirmed_block", string(commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		solanaConnectionErrors.WithLabelValues("get_confirmed_block_error").Inc()
		logger.Error("failed to request block", zap.Error(err), zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)))
		return
	}

	if out == nil {
		logger.Error("nil response when requesting block", zap.Error(err), zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)))
		return
	}

	logger.Info("fetched block",
		zap.Uint64("slot", slot),
		zap.Int("num_tx", len(out.Transactions)),
		zap.Duration("took", time.Since(start)),
		zap.String("commitment", string(commitment)))

OUTER:
	for _, tx := range out.Transactions {
		signature := tx.Transaction.Signatures[0]
		var programIndex uint16
		for n, key := range tx.Transaction.Message.AccountKeys {
			if key.Equals(s.bridge) {
				programIndex = uint16(n)
			}
		}
		if programIndex == 0 {
			continue
		}

		logger.Info("found Wormhole transaction",
			zap.Stringer("signature", signature),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)))

		// Find top-level instructions
		for _, inst := range tx.Transaction.Message.Instructions {
			if inst.ProgramIDIndex == programIndex {
				// The second account in a well-formed Wormhole instruction is the
				// VAA program account.
				if len(inst.Accounts) != 9 {
					logger.Error("malformed Wormhole instruction: wrong number of accounts",
						zap.Stringer("signature", signature),
						zap.Uint64("slot", slot),
						zap.String("commitment", string(commitment)))
					continue OUTER
				}

				acc := tx.Transaction.Message.AccountKeys[inst.Accounts[1]]
				go s.fetchMessageAccount(ctx, logger, acc, rpcClient, commitment, slot)
				continue OUTER
			}
		}

		// Call GetConfirmedTransaction to get at innerTransactions
		rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		defer cancel()
		start := time.Now()
		tr, err := rpcClient.GetConfirmedTransactionWithOpts(rCtx, signature, &rpc.GetTransactionOpts{
			Encoding:   "json",
			Commitment: commitment,
		})
		queryLatency.WithLabelValues("get_confirmed_transaction", string(commitment)).Observe(time.Since(start).Seconds())
		if err != nil {
			solanaConnectionErrors.WithLabelValues("get_confirmed_transaction_error").Inc()
			logger.Error("failed to request transaction",
				zap.Error(err),
				zap.Uint64("slot", slot),
				zap.String("commitment", string(commitment)),
				zap.Stringer("signature", signature))
			return
		}

		logger.Info("fetched transaction",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)),
			zap.Stringer("signature", signature),
			zap.Duration("took", time.Since(start)))

		for _, inner := range tr.Meta.InnerInstructions {
			for _, inst := range inner.Instructions {
				if inst.ProgramIDIndex == programIndex {
					if len(inst.Accounts) != 9 {
						logger.Error("malformed Wormhole instruction: wrong number of accounts",
							zap.Stringer("signature", signature),
							zap.Uint64("slot", slot),
							zap.String("commitment", string(commitment)))
						continue OUTER
					}

					acc := tx.Transaction.Message.AccountKeys[inst.Accounts[1]]
					go s.fetchMessageAccount(ctx, logger, acc, rpcClient, commitment, slot)
					continue OUTER
				}
			}
		}
	}
}

func (s *SolanaWatcher) fetchMessageAccount(ctx context.Context, logger *zap.Logger, acc solana.PublicKey, rpcClient *rpc.Client, commitment rpc.CommitmentType, slot uint64) {
	// Fetching account
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	info, err := rpcClient.GetAccountInfoWithOpts(rCtx, acc, &rpc.GetAccountInfoOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: commitment,
	})
	queryLatency.WithLabelValues("get_account_info", string(commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		solanaConnectionErrors.WithLabelValues("get_account_info_error").Inc()
		logger.Error("failed to request account",
			zap.Error(err),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)),
			zap.Stringer("account", acc))
		return
	}

	if !info.Value.Owner.Equals(s.bridge) {
		solanaConnectionErrors.WithLabelValues("account_owner_mismatch").Inc()
		logger.Error("account has invalid owner",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)),
			zap.Stringer("account", acc),
			zap.Stringer("unexpected_owner", info.Value.Owner))
		return
	}

	data := info.Value.Data.GetBinary()
	if string(data[:3]) != "msg" {
		solanaConnectionErrors.WithLabelValues("bad_account_data").Inc()
		logger.Error("account is not a message account",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(commitment)),
			zap.Stringer("account", acc))
		return
	}

	logger.Info("found valid VAA account",
		zap.Uint64("slot", slot),
		zap.String("commitment", string(commitment)),
		zap.Stringer("account", acc),
		zap.Binary("data", data))

	s.processMessageAccount(logger, data, acc)
}

func (s *SolanaWatcher) processMessageAccount(logger *zap.Logger, data []byte, acc solana.PublicKey) {
	proposal, err := ParseTransferOutProposal(data)
	if err != nil {
		solanaAccountSkips.WithLabelValues("parse_transfer_out").Inc()
		logger.Error(
			"failed to parse transfer proposal",
			zap.Stringer("account", acc),
			zap.Binary("data", data),
			zap.Error(err))
		return
	}

	var txHash eth_common.Hash
	copy(txHash[:], acc[:])

	observation := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        time.Unix(int64(proposal.SubmissionTime), 0),
		Nonce:            proposal.Nonce,
		Sequence:         proposal.Sequence,
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   proposal.EmitterAddress,
		Payload:          proposal.Payload,
		ConsistencyLevel: proposal.ConsistencyLevel,
	}

	solanaMessagesConfirmed.Inc()

	logger.Info("message observed",
		zap.Stringer("account", acc),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistency_level", observation.ConsistencyLevel),
	)

	s.messageEvent <- observation
}

type (
	MessagePublicationAccount struct {
		VaaVersion uint8
		// Borsh does not seem to support booleans, so 0=false / 1=true
		ConsistencyLevel    uint8
		VaaTime             uint32
		VaaSignatureAccount vaa.Address
		SubmissionTime      uint32
		Nonce               uint32
		Sequence            uint64
		EmitterChain        uint16
		EmitterAddress      vaa.Address
		Payload             []byte
	}
)

func ParseTransferOutProposal(data []byte) (*MessagePublicationAccount, error) {
	prop := &MessagePublicationAccount{}
	// Skip the b"msg" prefix
	if err := borsh.Deserialize(prop, data[3:]); err != nil {
		return nil, err
	}

	return prop, nil
}
