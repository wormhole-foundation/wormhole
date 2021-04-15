package ethereum

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"math/big"
	"time"
)

type SolanaWatcher struct {
	bridge       solana.PublicKey
	wsUrl        string
	rpcUrl       string
	messageEvent chan *common.MessagePublication
}

var (
	solanaConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_connection_errors_total",
			Help: "Total number of Solana connection errors",
		}, []string{"reason"})
	solanaAccountSkips = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_solana_account_updates_skipped_total",
			Help: "Total number of account updates skipped due to invalid data",
		}, []string{"reason"})
	solanaLockupsConfirmed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_solana_lockups_confirmed_total",
			Help: "Total number of verified Solana lockups found",
		})
	currentSolanaHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_solana_current_height",
			Help: "Current Solana slot height (at default commitment level, not the level used for lockups)",
		})
	queryLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_solana_query_latency",
			Help: "Latency histogram for Solana RPC calls",
		}, []string{"operation"})
)

func init() {
	prometheus.MustRegister(solanaConnectionErrors)
	prometheus.MustRegister(solanaAccountSkips)
	prometheus.MustRegister(solanaLockupsConfirmed)
	prometheus.MustRegister(currentSolanaHeight)
	prometheus.MustRegister(queryLatency)
}

func NewSolanaWatcher(wsUrl, rpcUrl string, bridgeAddress solana.PublicKey, messageEvents chan *common.MessagePublication) *SolanaWatcher {
	return &SolanaWatcher{bridge: bridgeAddress, wsUrl: wsUrl, rpcUrl: rpcUrl, messageEvent: messageEvents}
}

func (s *SolanaWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	bridgeAddr := base58.Encode(s.bridge[:])
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
		BridgeAddress: bridgeAddr,
	})

	rpcClient := rpc.NewClient(s.rpcUrl)
	logger := supervisor.Logger(ctx)
	errC := make(chan error)

	go func() {
		timer := time.NewTicker(time.Second * 5)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				func() {
					// Get current slot height
					rCtx, cancel := context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					start := time.Now()
					slot, err := rpcClient.GetSlot(rCtx, "")
					queryLatency.WithLabelValues("get_slot").Observe(time.Since(start).Seconds())
					if err != nil {
						solanaConnectionErrors.WithLabelValues("get_slot_error").Inc()
						errC <- err
						return
					}
					currentSolanaHeight.Set(float64(slot))
					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
						Height:        int64(slot),
						BridgeAddress: bridgeAddr,
					})

					logger.Info("current Solana height", zap.Uint64("slot", uint64(slot)))

					// Find MessagePublicationAccount accounts without a VAA
					rCtx, cancel = context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					start = time.Now()

					accounts, err := rpcClient.GetProgramAccounts(rCtx, s.bridge, &rpc.GetProgramAccountsOpts{
						Commitment: rpc.CommitmentMax, // TODO: deprecated, use Finalized
						Filters: []rpc.RPCFilter{
							{
								DataSize: 1184, // Search for MessagePublicationAccount accounts
							},
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 1140,                      // Offset of VaaTime
									Bytes:  solana.Base58{0, 0, 0, 0}, // VAA time is 0 when no VAA is present
								},
							},
						},
					})
					queryLatency.WithLabelValues("get_program_accounts").Observe(time.Since(start).Seconds())
					if err != nil {
						solanaConnectionErrors.WithLabelValues("get_program_account_error").Inc()
						errC <- err
						return
					}

					logger.Debug("fetched transfer proposals without VAA",
						zap.Int("n", len(accounts)),
						zap.Duration("took", time.Since(start)),
					)

					for _, acc := range accounts {
						proposal, err := ParseTransferOutProposal(acc.Account.Data)
						if err != nil {
							solanaAccountSkips.WithLabelValues("parse_transfer_out").Inc()
							logger.Warn(
								"failed to parse transfer proposal",
								zap.Stringer("account", acc.Pubkey),
								zap.Error(err),
							)
							continue
						}

						// VAA submitted
						if proposal.VaaTime.Unix() != 0 {
							solanaAccountSkips.WithLabelValues("is_submitted_vaa").Inc()
							continue
						}

						var txHash eth_common.Hash
						copy(txHash[:], acc.Pubkey[:])

						lock := &common.MessagePublication{
							TxHash:        txHash,
							Timestamp:     proposal.LockupTime,
							Nonce:         proposal.Nonce,
							SourceAddress: proposal.SourceAddress,
							TargetAddress: proposal.ForeignAddress,
							SourceChain:   vaa.ChainIDSolana,
							TargetChain:   proposal.ToChainID,
							TokenChain:    proposal.Asset.Chain,
							TokenAddress:  proposal.Asset.Address,
							TokenDecimals: proposal.Asset.Decimals,
							Amount:        proposal.Amount,
						}

						solanaLockupsConfirmed.Inc()
						logger.Info("found lockup without VAA", zap.Stringer("lockup_address", acc.Pubkey))
						s.messageEvent <- lock
					}
				}()
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

type (
	MessagePublicationAccount struct {
		VaaVersion          uint8
		VaaTime             time.Time
		VaaSignatureAccount vaa.Address
		SubmissionTime      time.Time
		Nonce               uint32
		EmitterChain        vaa.ChainID
		EmitterAddress      vaa.Address
		Payload             []byte
	}
)

func ParseTransferOutProposal(data []byte) (*MessagePublicationAccount, error) {
	prop := &MessagePublicationAccount{}
	r := bytes.NewBuffer(data)

	// Skip initialized bool
	r.Next(1)

	if err := binary.Read(r, binary.LittleEndian, &prop.VaaVersion); err != nil {
		return nil, fmt.Errorf("failed to read to vaa version: %w", err)
	}

	var vaaTime uint32
	if err := binary.Read(r, binary.LittleEndian, &vaaTime); err != nil {
		return nil, fmt.Errorf("failed to read vaa time: %w", err)
	}
	prop.VaaTime = time.Unix(int64(vaaTime), 0)

	if n, err := r.Read(prop.VaaSignatureAccount[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read signature account: %w", err)
	}

	var submissionTime uint32
	if err := binary.Read(r, binary.LittleEndian, &submissionTime); err != nil {
		return nil, fmt.Errorf("failed to read lockup time: %w", err)
	}
	prop.SubmissionTime = time.Unix(int64(submissionTime), 0)

	if err := binary.Read(r, binary.LittleEndian, &prop.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &prop.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	if n, err := r.Read(prop.EmitterAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address: %w", err)
	}

	payload := make([]byte, 1000)
	n, err := r.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read vaa: %w", err)
	}
	prop.Payload = payload[:n]

	return prop, nil
}
