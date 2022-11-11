package solana

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/mr-tron/base58"
	"github.com/near/borsh-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type SolanaWatcher struct {
	contract     solana.PublicKey
	rpcUrl       string
	commitment   rpc.CommitmentType
	messageEvent chan *common.MessagePublication
	obsvReqC     chan *gossipv1.ObservationRequest
	rpcClient    *rpc.Client
	// Readiness component
	readiness readiness.Component
	// VAA ChainID of the network we're connecting to.
	chainID vaa.ChainID
	// Human readable name of network
	networkName string
	// The last slot processed by the watcher.
	lastSlot uint64
}

var (
	solanaConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_connection_errors_total",
			Help: "Total number of Solana connection errors",
		}, []string{"solana_network", "commitment", "reason"})
	solanaAccountSkips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_account_updates_skipped_total",
			Help: "Total number of account updates skipped due to invalid data",
		}, []string{"solana_network", "reason"})
	solanaMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_observations_confirmed_total",
			Help: "Total number of verified Solana observations found",
		}, []string{"solana_network"})
	currentSolanaHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_solana_current_height",
			Help: "Current Solana slot height",
		}, []string{"solana_network", "commitment"})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_solana_query_latency",
			Help: "Latency histogram for Solana RPC calls",
		}, []string{"solana_network", "operation", "commitment"})
)

const rpcTimeout = time.Second * 5

// Maximum retries for Solana fetching
const maxRetries = 10
const retryDelay = 5 * time.Second

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
	postMessageInstructionNumAccounts  = 9
	postMessageInstructionID           = 0x01
	postMessageUnreliableInstructionID = 0x08
	accountPrefixReliable              = "msg"
	accountPrefixUnreliable            = "msu"
)

// PostMessageData represents the user-supplied, untrusted instruction data
// for message publications. We use this to determine consistency level before fetching accounts.
type PostMessageData struct {
	Nonce            uint32
	Payload          []byte
	ConsistencyLevel ConsistencyLevel
}

func NewSolanaWatcher(
	rpcUrl string,
	contractAddress solana.PublicKey,
	messageEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
	commitment rpc.CommitmentType,
	readiness readiness.Component,
	chainID vaa.ChainID) *SolanaWatcher {
	return &SolanaWatcher{
		rpcUrl:       rpcUrl,
		contract:     contractAddress,
		messageEvent: messageEvents,
		obsvReqC:     obsvReqC,
		commitment:   commitment,
		rpcClient:    rpc.New(rpcUrl),
		readiness:    readiness,
		chainID:      chainID,
		networkName:  vaa.ChainID(chainID).String(),
	}
}

func (s *SolanaWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	contractAddr := base58.Encode(s.contract[:])
	p2p.DefaultRegistry.SetNetworkStats(s.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: contractAddr,
	})

	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	go func() {
		timer := time.NewTicker(time.Second * 1)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case m := <-s.obsvReqC:
				if m.ChainId != uint32(s.chainID) {
					panic("unexpected chain id")
				}

				acc := solana.PublicKeyFromBytes(m.TxHash)
				logger.Info("received observation request", zap.String("account", acc.String()))

				rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
				s.fetchMessageAccount(rCtx, logger, acc, 0)
				cancel()
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
					errC <- err
					return
				}

				lastSlot := s.lastSlot
				if lastSlot == 0 {
					lastSlot = slot - 1
				}
				currentSolanaHeight.WithLabelValues(s.networkName, string(s.commitment)).Set(float64(slot))
				readiness.SetReady(s.readiness)
				p2p.DefaultRegistry.SetNetworkStats(s.chainID, &gossipv1.Heartbeat_Network{
					Height:          int64(slot),
					ContractAddress: contractAddr,
				})

				rangeStart := lastSlot + 1
				rangeEnd := slot

				logger.Debug("fetched current Solana height",
					zap.String("commitment", string(s.commitment)),
					zap.Uint64("slot", slot),
					zap.Uint64("lastSlot", lastSlot),
					zap.Uint64("pendingSlots", slot-lastSlot),
					zap.Uint64("from", rangeStart), zap.Uint64("to", rangeEnd),
					zap.Duration("took", time.Since(start)))

				// Requesting each slot
				for slot := rangeStart; slot <= rangeEnd; slot++ {
					go s.retryFetchBlock(ctx, logger, slot, 0)
				}

				s.lastSlot = slot
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

func (s *SolanaWatcher) retryFetchBlock(ctx context.Context, logger *zap.Logger, slot uint64, retry uint) {
	ok := s.fetchBlock(ctx, logger, slot, 0)

	if !ok {
		if retry >= maxRetries {
			logger.Error("max retries for block",
				zap.Uint64("slot", slot),
				zap.String("commitment", string(s.commitment)),
				zap.Uint("retry", retry))
			return
		}

		time.Sleep(retryDelay)

		logger.Info("retrying block",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Uint("retry", retry))

		go s.retryFetchBlock(ctx, logger, slot, retry+1)
	}
}

func (s *SolanaWatcher) fetchBlock(ctx context.Context, logger *zap.Logger, slot uint64, emptyRetry uint) (ok bool) {
	logger.Debug("requesting block",
		zap.Uint64("slot", slot),
		zap.String("commitment", string(s.commitment)),
		zap.Uint("empty_retry", emptyRetry))
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	rewards := false

	maxSupportedTransactionVersion := uint64(0)
	out, err := s.rpcClient.GetBlockWithOpts(rCtx, slot, &rpc.GetBlockOpts{
		Encoding:                       solana.EncodingBase64, // solana-go doesn't support json encoding.
		TransactionDetails:             "full",
		Rewards:                        &rewards,
		Commitment:                     s.commitment,
		MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
	})

	queryLatency.WithLabelValues(s.networkName, "get_confirmed_block", string(s.commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		var rpcErr *jsonrpc.RPCError
		if errors.As(err, &rpcErr) && (rpcErr.Code == -32007 /* SLOT_SKIPPED */ || rpcErr.Code == -32004 /* BLOCK_NOT_AVAILABLE */) {
			logger.Info("empty slot", zap.Uint64("slot", slot),
				zap.Int("code", rpcErr.Code),
				zap.String("commitment", string(s.commitment)))

			// TODO(leo): clean this up once we know what's happening
			// https://github.com/solana-labs/solana/issues/20370
			var maxEmptyRetry uint
			if s.commitment == rpc.CommitmentFinalized {
				maxEmptyRetry = 5
			} else {
				maxEmptyRetry = 1
			}

			// Schedule a single retry just in case the Solana node was confused about the block being missing.
			if emptyRetry < maxEmptyRetry {
				go func() {
					time.Sleep(retryDelay)
					s.fetchBlock(ctx, logger, slot, emptyRetry+1)
				}()
			}
			return true
		} else {
			logger.Error("failed to request block", zap.Error(err), zap.Uint64("slot", slot),
				zap.String("commitment", string(s.commitment)))
			p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
			solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "get_confirmed_block_error").Inc()
		}
		return false
	}

	if out == nil {
		// Per the API, nil just means the block is not confirmed.
		logger.Info("block is not yet finalized", zap.Uint64("slot", slot))
		return false
	}

	logger.Debug("fetched block",
		zap.Uint64("slot", slot),
		zap.Int("num_tx", len(out.Transactions)),
		zap.Duration("took", time.Since(start)),
		zap.String("commitment", string(s.commitment)))

OUTER:
	for txNum, txRpc := range out.Transactions {
		if txRpc.Meta.Err != nil {
			logger.Debug("Transaction failed, skipping it",
				zap.Uint64("slot", slot),
				zap.Int("txNum", txNum),
				zap.String("err", fmt.Sprint(txRpc.Meta.Err)),
			)
			continue
		}
		tx, err := txRpc.GetTransaction()
		if err != nil {
			logger.Error("failed to unmarshal transaction",
				zap.Uint64("slot", slot),
				zap.Int("txNum", txNum),
				zap.Int("dataLen", len(txRpc.Transaction.GetBinary())),
				zap.Error(err),
			)
			continue
		}
		signature := tx.Signatures[0]
		var programIndex uint16
		for n, key := range tx.Message.AccountKeys {
			if key.Equals(s.contract) {
				programIndex = uint16(n)
			}
		}
		if programIndex == 0 {
			continue
		}

		if txRpc.Meta.Err != nil {
			logger.Debug("skipping failed Wormhole transaction",
				zap.Stringer("signature", signature),
				zap.Uint64("slot", slot),
				zap.String("commitment", string(s.commitment)))
			continue
		}

		logger.Debug("found Wormhole transaction",
			zap.Stringer("signature", signature),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)))

		// Find top-level instructions
		for i, inst := range tx.Message.Instructions {
			found, err := s.processInstruction(ctx, logger, slot, inst, programIndex, tx, signature, i)
			if err != nil {
				logger.Error("malformed Wormhole instruction",
					zap.Error(err),
					zap.Int("idx", i),
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
		start := time.Now()
		maxSupportedTransactionVersion := uint64(0)
		tr, err := s.rpcClient.GetTransaction(rCtx, signature, &rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64, // solana-go doesn't support json encoding.
			Commitment:                     s.commitment,
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		})
		cancel()
		queryLatency.WithLabelValues(s.networkName, "get_confirmed_transaction", string(s.commitment)).Observe(time.Since(start).Seconds())
		if err != nil {
			p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
			solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "get_confirmed_transaction_error").Inc()
			logger.Error("failed to request transaction",
				zap.Error(err),
				zap.Uint64("slot", slot),
				zap.String("commitment", string(s.commitment)),
				zap.Stringer("signature", signature))
			return false
		}

		logger.Debug("fetched transaction",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("signature", signature),
			zap.Duration("took", time.Since(start)))

		for _, inner := range tr.Meta.InnerInstructions {
			for i, inst := range inner.Instructions {
				_, err = s.processInstruction(ctx, logger, slot, inst, programIndex, tx, signature, i)
				if err != nil {
					logger.Error("malformed Wormhole instruction",
						zap.Error(err),
						zap.Int("idx", i),
						zap.Stringer("signature", signature),
						zap.Uint64("slot", slot),
						zap.String("commitment", string(s.commitment)))
				}
			}
		}
	}

	if emptyRetry > 0 {
		logger.Warn("SOLANA BUG: skipped or unavailable block retrieved on retry attempt",
			zap.Uint("empty_retry", emptyRetry),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)))
	}

	return true
}

func (s *SolanaWatcher) processInstruction(ctx context.Context, logger *zap.Logger, slot uint64, inst solana.CompiledInstruction, programIndex uint16, tx *solana.Transaction, signature solana.Signature, idx int) (bool, error) {
	if inst.ProgramIDIndex != programIndex {
		return false, nil
	}

	if len(inst.Data) == 0 {
		return false, nil
	}

	if inst.Data[0] != postMessageInstructionID && inst.Data[0] != postMessageUnreliableInstructionID {
		return false, nil
	}

	if len(inst.Accounts) != postMessageInstructionNumAccounts {
		return false, fmt.Errorf("invalid number of accounts: %d instead of %d",
			len(inst.Accounts), postMessageInstructionNumAccounts)
	}

	// Decode instruction data (UNTRUSTED)
	var data PostMessageData
	if err := borsh.Deserialize(&data, inst.Data[1:]); err != nil {
		return false, fmt.Errorf("failed to deserialize instruction data: %w", err)
	}

	logger.Debug("post message data", zap.Any("deserialized_data", data),
		zap.Stringer("signature", signature), zap.Uint64("slot", slot), zap.Int("idx", idx))

	level, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return false, fmt.Errorf("failed to determine commitment: %w", err)
	}

	if level != s.commitment {
		return true, nil
	}

	// The second account in a well-formed Wormhole instruction is the VAA program account.
	acc := tx.Message.AccountKeys[inst.Accounts[1]]

	logger.Debug("fetching VAA account", zap.Stringer("acc", acc),
		zap.Stringer("signature", signature), zap.Uint64("slot", slot), zap.Int("idx", idx))

	go s.retryFetchMessageAccount(ctx, logger, acc, slot, 0)

	return true, nil
}

func (s *SolanaWatcher) retryFetchMessageAccount(ctx context.Context, logger *zap.Logger, acc solana.PublicKey, slot uint64, retry uint) {
	retryable := s.fetchMessageAccount(ctx, logger, acc, slot)

	if retryable {
		if retry >= maxRetries {
			logger.Error("max retries for account",
				zap.Uint64("slot", slot),
				zap.Stringer("account", acc),
				zap.String("commitment", string(s.commitment)),
				zap.Uint("retry", retry))
			return
		}

		time.Sleep(retryDelay)

		logger.Info("retrying account",
			zap.Uint64("slot", slot),
			zap.Stringer("account", acc),
			zap.String("commitment", string(s.commitment)),
			zap.Uint("retry", retry))

		go s.retryFetchMessageAccount(ctx, logger, acc, slot, retry+1)
	}
}

func (s *SolanaWatcher) fetchMessageAccount(ctx context.Context, logger *zap.Logger, acc solana.PublicKey, slot uint64) (retryable bool) {
	// Fetching account
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	start := time.Now()
	info, err := s.rpcClient.GetAccountInfoWithOpts(rCtx, acc, &rpc.GetAccountInfoOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: s.commitment,
	})
	queryLatency.WithLabelValues(s.networkName, "get_account_info", string(s.commitment)).Observe(time.Since(start).Seconds())
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
		solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "get_account_info_error").Inc()
		logger.Error("failed to request account",
			zap.Error(err),
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc))
		return true
	}

	if !info.Value.Owner.Equals(s.contract) {
		p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
		solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "account_owner_mismatch").Inc()
		logger.Error("account has invalid owner",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc),
			zap.Stringer("unexpected_owner", info.Value.Owner))
		return false
	}

	data := info.Value.Data.GetBinary()
	if string(data[:3]) != accountPrefixReliable && string(data[:3]) != accountPrefixUnreliable {
		p2p.DefaultRegistry.AddErrorCount(s.chainID, 1)
		solanaConnectionErrors.WithLabelValues(s.networkName, string(s.commitment), "bad_account_data").Inc()
		logger.Error("account is not a message account",
			zap.Uint64("slot", slot),
			zap.String("commitment", string(s.commitment)),
			zap.Stringer("account", acc))
		return false
	}

	logger.Debug("found valid VAA account",
		zap.Uint64("slot", slot),
		zap.String("commitment", string(s.commitment)),
		zap.Stringer("account", acc),
		zap.Binary("data", data))

	s.processMessageAccount(logger, data, acc)
	return false
}

func (s *SolanaWatcher) processMessageAccount(logger *zap.Logger, data []byte, acc solana.PublicKey) {
	proposal, err := ParseMessagePublicationAccount(data)
	if err != nil {
		solanaAccountSkips.WithLabelValues(s.networkName, "parse_transfer_out").Inc()
		logger.Error(
			"failed to parse transfer proposal",
			zap.Stringer("account", acc),
			zap.Binary("data", data),
			zap.Error(err))
		return
	}

	var txHash eth_common.Hash
	copy(txHash[:], acc[:])

	var reliable bool
	switch string(data[:3]) {
	case accountPrefixReliable:
		reliable = true
	case accountPrefixUnreliable:
		reliable = false
	default:
		panic("invalid prefix")
	}

	observation := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        time.Unix(int64(proposal.SubmissionTime), 0),
		Nonce:            proposal.Nonce,
		Sequence:         proposal.Sequence,
		EmitterChain:     s.chainID,
		EmitterAddress:   proposal.EmitterAddress,
		Payload:          proposal.Payload,
		ConsistencyLevel: proposal.ConsistencyLevel,
		Unreliable:       !reliable,
	}

	solanaMessagesConfirmed.WithLabelValues(s.networkName).Inc()

	logger.Debug("message observed",
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

func ParseMessagePublicationAccount(data []byte) (*MessagePublicationAccount, error) {
	prop := &MessagePublicationAccount{}
	// Skip the b"msg" prefix
	if err := borsh.Deserialize(prop, data[3:]); err != nil {
		return nil, err
	}

	return prop, nil
}
