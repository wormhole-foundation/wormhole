package solana

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
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
	contract     solana.PublicKey
	wsUrl        string
	rpcUrl       string
	commitment   rpc.CommitmentType
	messageEvent chan *common.MessagePublication
	logger       *zap.Logger
	rpcClient    *rpc.Client
}

var (
	solanaConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_connection_errors_total",
			Help: "Total number of Solana connection errors",
		}, []string{"commitment", "reason"})
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
	currentSolanaHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_solana_current_height",
			Help: "Current Solana slot height",
		}, []string{"commitment"})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_solana_query_latency",
			Help: "Latency histogram for Solana RPC calls",
		}, []string{"operation", "commitment"})
)

const rpcTimeout = time.Second * 5

type ConsistencyLevel uint8

// Mappings from consistency levels constants to commitment level.
const (
	consistencyLevelConfirmed ConsistencyLevel = 0
	consistencyLevelFinalized ConsistencyLevel = 1
)

func (c ConsistencyLevel) Commitment() (rpc.CommitmentType, error) {
	switch c {
	case consistencyLevelConfirmed:
		return rpc.CommitmentConfirmed, nil
	case consistencyLevelFinalized:
		return rpc.CommitmentFinalized, nil
	default:
		return "", fmt.Errorf("unsupported consistency level: %d", c)
	}
}

const (
	postMessageInstructionNumAccounts = 9
	postMessageInstructionID          = 0x01
)

// PostMessageData represents the user-supplied, untrusted instruction data
// for message publications. We use this to determine consistency level before fetching accounts.
type PostMessageData struct {
	Nonce            uint32
	Payload          []byte
	ConsistencyLevel ConsistencyLevel
}

func NewSolanaWatcher(
	wsUrl, rpcUrl string,
	contractAddress solana.PublicKey,
	messageEvents chan *common.MessagePublication,
	commitment rpc.CommitmentType) *SolanaWatcher {
	return &SolanaWatcher{
		contract: contractAddress,
		wsUrl:    wsUrl, rpcUrl: rpcUrl,
		messageEvent: messageEvents,
		commitment:   commitment,
		rpcClient:    rpc.New(rpcUrl),
	}
}

func (s *SolanaWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	contractAddr := base58.Encode(s.contract[:])
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
		ContractAddress: contractAddr,
	})

	s.logger = supervisor.Logger(ctx)
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
				// Get current slot height
				rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
				defer cancel()
				start := time.Now()
				slot, err := s.rpcClient.GetSlot(rCtx, s.commitment)
				queryLatency.WithLabelValues("get_slot", string(s.commitment)).Observe(time.Since(start).Seconds())
				if err != nil {
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
					solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_slot_error").Inc()
					errC <- err
					return
				}
				if lastSlot == 0 {
					lastSlot = slot - 1
				}
				currentSolanaHeight.WithLabelValues(string(s.commitment)).Set(float64(slot))
				readiness.SetReady(common.ReadinessSolanaSyncing)
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
					Height:          int64(slot),
					ContractAddress: contractAddr,
				})
				s.logger.Info("fetched current Solana height",
					zap.String("commitment", string(s.commitment)),
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
				slots, err := s.rpcClient.GetConfirmedBlocks(rCtx, rangeStart, &rangeEnd, s.commitment)
				queryLatency.WithLabelValues("get_confirmed_blocks", string(s.commitment)).Observe(time.Since(start).Seconds())
				if err != nil {
					p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
					solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_confirmed_blocks_error").Inc()
					errC <- err
					return
				}

				s.logger.Info("fetched slots in range",
					zap.Uint64("from", rangeStart), zap.Uint64("to", rangeEnd),
					zap.Duration("took", time.Since(start)),
					zap.String("commitment", string(s.commitment)))

				// Requesting each slot
				for _, slot := range slots {
					if slot <= lastSlot {
						// Skip out-of-range result
						// https://github.com/solana-labs/solana/issues/18946
						continue
					}

					go s.fetchBlock(ctx, slot)
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

func (s *SolanaWatcher) fetchBlock(ctx context.Context, slot uint64) {
	s.logger.Debug("requesting block", zap.Uint64("slot", slot), zap.String("commitment", string(s.commitment)))
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	rewards := false
	out, err := s.rpcClient.GetConfirmedBlockWithOpts(rCtx, slot, &rpc.GetConfirmedBlockOpts{
		Encoding:           "json",
		TransactionDetails: "full",
		Rewards:            &rewards,
		Commitment:         s.commitment,
	})

	queryLatency.WithLabelValues("get_confirmed_block", string(s.commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
		solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_confirmed_block_error").Inc()
		s.logger.Error("failed to request block", zap.Error(err), zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)))
		return
	}

	if out == nil {
		solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_confirmed_block_error").Inc()
		s.logger.Error("nil response when requesting block", zap.Error(err), zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
		return
	}

	s.logger.Info("fetched block",
		zap.Uint64("slot", slot),
		zap.Int("num_tx", len(out.Transactions)),
		zap.Duration("took", time.Since(start)),
		zap.String("commitment", string(s.commitment)))

OUTER:
	for _, tx := range out.Transactions {
		signature := tx.Transaction.Signatures[0]
		var programIndex uint16
		for n, key := range tx.Transaction.Message.AccountKeys {
			if key.Equals(s.contract) {
				programIndex = uint16(n)
			}
		}
		if programIndex == 0 {
			continue
		}

		s.logger.Info("found Wormhole transaction",
			zap.Stringer("signature", signature),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)))

		// Find top-level instructions
		for _, inst := range tx.Transaction.Message.Instructions {
			found, err := s.processInstruction(ctx, slot, inst, programIndex, tx)
			if err != nil {
				s.logger.Error("malformed Wormhole instruction",
					zap.Error(err),
					zap.Stringer("signature", signature),
					zap.Uint64("slot", slot),
					zap.String("commitment", string(s.commitment)),
					zap.Binary("data", inst.Data))
				continue OUTER
			}
			if found {
				continue OUTER
			}
		}

		// Call GetConfirmedTransaction to get at innerTransactions
		rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		defer cancel()
		start := time.Now()
		tr, err := s.rpcClient.GetConfirmedTransactionWithOpts(rCtx, signature, &rpc.GetTransactionOpts{
			Encoding:   "json",
			Commitment: s.commitment,
		})
		queryLatency.WithLabelValues("get_confirmed_transaction", string(s.commitment)).Observe(time.Since(start).Seconds())
		if err != nil {
			p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
			solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_confirmed_transaction_error").Inc()
			s.logger.Error("failed to request transaction",
				zap.Error(err),
				zap.Uint64("slot", slot),
				zap.String("commitment", string(s.commitment)),
				zap.Stringer("signature", signature))
			return
		}

		s.logger.Info("fetched transaction",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("signature", signature),
			zap.Duration("took", time.Since(start)))

		for _, inner := range tr.Meta.InnerInstructions {
			for _, inst := range inner.Instructions {
				_, err := s.processInstruction(ctx, slot, inst, programIndex, tx)
				if err != nil {
					s.logger.Error("malformed Wormhole instruction",
						zap.Error(err),
						zap.Stringer("signature", signature),
						zap.Uint64("slot", slot),
						zap.String("commitment", string(s.commitment)))
				}
			}
		}
	}
}

func (s *SolanaWatcher) processInstruction(ctx context.Context, slot uint64, inst solana.CompiledInstruction, programIndex uint16, tx rpc.TransactionWithMeta) (bool, error) {
	if inst.ProgramIDIndex != programIndex {
		return false, nil
	}

	if len(inst.Accounts) != postMessageInstructionNumAccounts {
		return false, fmt.Errorf("invalid number of accounts: %d instead of %d",
			len(inst.Accounts), postMessageInstructionNumAccounts)
	}

	if inst.Data[0] != postMessageInstructionID {
		return false, fmt.Errorf("invalid postMessage instruction ID, got: %d", inst.Data[0])
	}

	// Decode instruction data (UNTRUSTED)
	var data PostMessageData
	if err := borsh.Deserialize(&data, inst.Data[1:]); err != nil {
		return false, fmt.Errorf("failed to deserialize instruction data: %w", err)
	}

	s.logger.Info("post message data", zap.Any("deserialized_data", data))

	level, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return false, fmt.Errorf("failed to determine commitment: %w", err)
	}

	if level != s.commitment {
		return true, nil
	}

	// The second account in a well-formed Wormhole instruction is the VAA program account.
	acc := tx.Transaction.Message.AccountKeys[inst.Accounts[1]]
	go s.fetchMessageAccount(ctx, acc, slot)

	return true, nil
}

func (s *SolanaWatcher) fetchMessageAccount(ctx context.Context, acc solana.PublicKey, slot uint64) {
	// Fetching account
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	info, err := s.rpcClient.GetAccountInfoWithOpts(rCtx, acc, &rpc.GetAccountInfoOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: s.commitment,
	})
	queryLatency.WithLabelValues("get_account_info", string(s.commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
		solanaConnectionErrors.WithLabelValues(string(s.commitment), "get_account_info_error").Inc()
		s.logger.Error("failed to request account",
			zap.Error(err),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc))
		return
	}

	if !info.Value.Owner.Equals(s.contract) {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
		solanaConnectionErrors.WithLabelValues(string(s.commitment), "account_owner_mismatch").Inc()
		s.logger.Error("account has invalid owner",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc),
			zap.Stringer("unexpected_owner", info.Value.Owner))
		return
	}

	data := info.Value.Data.GetBinary()
	if string(data[:3]) != "msg" {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSolana, 1)
		solanaConnectionErrors.WithLabelValues(string(s.commitment), "bad_account_data").Inc()
		s.logger.Error("account is not a message account",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc))
		return
	}

	s.logger.Info("found valid VAA account",
		zap.Uint64("slot", slot),
		zap.String("commitment", string(s.commitment)),
		zap.Stringer("account", acc),
		zap.Binary("data", data))

	s.processMessageAccount(data, acc)
}

func (s *SolanaWatcher) processMessageAccount(data []byte, acc solana.PublicKey) {
	proposal, err := ParseTransferOutProposal(data)
	if err != nil {
		solanaAccountSkips.WithLabelValues("parse_transfer_out").Inc()
		s.logger.Error(
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

	s.logger.Info("message observed",
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
