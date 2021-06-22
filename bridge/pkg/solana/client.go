package solana

import (
	"context"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/near/borsh-go"
	"github.com/prometheus/client_golang/prometheus"
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
								Memcmp: &rpc.RPCFilterMemcmp{
									Offset: 0,                            // Offset of VaaTime
									Bytes:  solana.Base58{'m', 's', 'g'}, // Prefix of the posted message accounts
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
						if proposal.VaaTime != 0 {
							solanaAccountSkips.WithLabelValues("is_submitted_vaa").Inc()
							continue
						}

						var txHash eth_common.Hash
						copy(txHash[:], acc.Pubkey[:])

						lock := &common.MessagePublication{
							TxHash:         txHash,
							Timestamp:      time.Unix(int64(proposal.SubmissionTime), 0),
							Nonce:          proposal.Nonce,
							Sequence:       proposal.Sequence,
							EmitterChain:   proposal.EmitterChain,
							EmitterAddress: proposal.EmitterAddress,
							Payload:        proposal.Payload,
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
		VaaTime             uint32
		VaaSignatureAccount vaa.Address
		SubmissionTime      uint32
		Nonce               uint32
		Sequence            uint64
		EmitterChain        vaa.ChainID
		EmitterAddress      vaa.Address
		Payload             []byte
	}
)

func ParseTransferOutProposal(data []byte) (*MessagePublicationAccount, error) {
	prop := &MessagePublicationAccount{}
	if err := borsh.Deserialize(prop, data); err != nil {
		return nil, err
	}

	return prop, nil
}
