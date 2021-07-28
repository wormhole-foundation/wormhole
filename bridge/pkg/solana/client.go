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
					queryLatency.WithLabelValues("get_slot", "processed").Observe(time.Since(start).Seconds())
					if err != nil {
						solanaConnectionErrors.WithLabelValues("get_slot_error").Inc()
						errC <- err
						return
					}
					currentSolanaHeight.Set(float64(slot))
					readiness.SetReady(common.ReadinessSolanaSyncing)
					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
						Height:        int64(slot),
						BridgeAddress: bridgeAddr,
					})

					logger.Info("current Solana height", zap.Uint64("slot", uint64(slot)))

					// Find MessagePublicationAccount accounts without a VAA
					rCtx, cancel = context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					start = time.Now()

					// Get finalized accounts
					fAccounts, err := rpcClient.GetProgramAccountsWithOpts(rCtx, s.bridge, &rpc.GetProgramAccountsOpts{
						Commitment: rpc.CommitmentFinalized,
						Filters: []rpc.RPCFilter{
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 0,                            // Start of the account
									Bytes:  solana.Base58{'m', 's', 'g'}, // Prefix of the posted message accounts
								},
							},
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 4,                 // Start of the ConsistencyLevel value
									Bytes:  solana.Base58{32}, // Only grab messages that require max confirmations
								},
							},
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 5,                         // Offset of VaaTime
									Bytes:  solana.Base58{0, 0, 0, 0}, // This means this VAA hasn't been signed yet
								},
							},
						},
					})
					queryLatency.WithLabelValues("get_program_accounts", "max").Observe(time.Since(start).Seconds())
					if err != nil {
						solanaConnectionErrors.WithLabelValues("get_program_account_error").Inc()
						errC <- err
						return
					}

					// Get confirmed accounts
					cAccounts, err := rpcClient.GetProgramAccountsWithOpts(rCtx, s.bridge, &rpc.GetProgramAccountsOpts{
						Commitment: rpc.CommitmentConfirmed,
						Filters: []rpc.RPCFilter{
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 0,                            // Start of the account
									Bytes:  solana.Base58{'m', 's', 'g'}, // Prefix of the posted message accounts
								},
							},
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 4,                // Start of the ConsistencyLevel value
									Bytes:  solana.Base58{1}, // Only grab messages that require the Confirmed level
								},
							},
							{
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 5,                         // Offset of VaaTime
									Bytes:  solana.Base58{0, 0, 0, 0}, // This means this VAA hasn't been signed yet
								},
							},
						},
					})
					queryLatency.WithLabelValues("get_program_accounts", "single").Observe(time.Since(start).Seconds())
					if err != nil {
						solanaConnectionErrors.WithLabelValues("get_program_account_error").Inc()
						errC <- err
						return
					}

					// Merge accounts
					accounts := append(fAccounts, cAccounts...)

					logger.Info("fetched transfer proposals without VAA",
						zap.Int("n", len(accounts)),
						zap.Duration("took", time.Since(start)),
					)

					for _, acc := range accounts {
						proposal, err := ParseTransferOutProposal(acc.Account.Data.GetBinary())
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
						if proposal.VaaTime != 0 {
							solanaAccountSkips.WithLabelValues("is_submitted_vaa").Inc()
							continue
						}

						var txHash eth_common.Hash
						copy(txHash[:], acc.Pubkey[:])

						lock := &common.MessagePublication{
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
						logger.Info("found message account without VAA", zap.Stringer("address", acc.Pubkey))
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
