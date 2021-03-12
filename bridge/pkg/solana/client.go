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
	bridge    solana.PublicKey
	wsUrl     string
	rpcUrl    string
	lockEvent chan *common.ChainLock
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

func NewSolanaWatcher(wsUrl, rpcUrl string, bridgeAddress solana.PublicKey, lockEvents chan *common.ChainLock) *SolanaWatcher {
	return &SolanaWatcher{bridge: bridgeAddress, wsUrl: wsUrl, rpcUrl: rpcUrl, lockEvent: lockEvents}
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

					// Find TransferOutProposal accounts without a VAA
					rCtx, cancel = context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					start = time.Now()

					accounts, err := rpcClient.GetProgramAccounts(rCtx, s.bridge, &rpc.GetProgramAccountsOpts{
						Commitment: rpc.CommitmentMax, // TODO: deprecated, use Finalized
						Filters: []rpc.RPCFilter{
							{
								DataSize: 1184, // Search for TransferOutProposal accounts
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

						lock := &common.ChainLock{
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
						s.lockEvent <- lock
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
	TransferOutProposal struct {
		Amount           *big.Int
		ToChainID        vaa.ChainID
		SourceChain      uint8
		SourceAddress    vaa.Address
		ForeignAddress   vaa.Address
		Asset            vaa.AssetMeta
		Nonce            uint32
		VAA              [1001]byte
		VaaTime          time.Time
		LockupTime       time.Time
		PokeCounter      uint8
		SignatureAccount solana.PublicKey
	}
)

func ParseTransferOutProposal(data []byte) (*TransferOutProposal, error) {
	prop := &TransferOutProposal{}
	r := bytes.NewBuffer(data)

	var amountBytes [32]byte
	if n, err := r.Read(amountBytes[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read amount: %w", err)
	}
	// Reverse (little endian -> big endian)
	for i := 0; i < len(amountBytes)/2; i++ {
		amountBytes[i], amountBytes[len(amountBytes)-i-1] = amountBytes[len(amountBytes)-i-1], amountBytes[i]
	}
	prop.Amount = new(big.Int).SetBytes(amountBytes[:])

	if err := binary.Read(r, binary.LittleEndian, &prop.ToChainID); err != nil {
		return nil, fmt.Errorf("failed to read to chain id: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &prop.SourceChain); err != nil {
		return nil, fmt.Errorf("failed to read source chain: %w", err)
	}

	if n, err := r.Read(prop.SourceAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read source address: %w", err)
	}

	if n, err := r.Read(prop.ForeignAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read source address: %w", err)
	}

	assetMeta := vaa.AssetMeta{}
	if n, err := r.Read(assetMeta.Address[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read asset meta address: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &assetMeta.Chain); err != nil {
		return nil, fmt.Errorf("failed to read asset meta chain: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &assetMeta.Decimals); err != nil {
		return nil, fmt.Errorf("failed to read asset meta decimals: %w", err)
	}
	prop.Asset = assetMeta

	if err := binary.Read(r, binary.LittleEndian, &prop.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if n, err := r.Read(prop.VAA[:]); err != nil || n != 1001 {
		return nil, fmt.Errorf("failed to read vaa: %w", err)
	}

	// Skip alignment bytes
	r.Next(3)

	var vaaTime uint32
	if err := binary.Read(r, binary.LittleEndian, &vaaTime); err != nil {
		return nil, fmt.Errorf("failed to read vaa time: %w", err)
	}
	prop.VaaTime = time.Unix(int64(vaaTime), 0)

	var lockupTime uint32
	if err := binary.Read(r, binary.LittleEndian, &lockupTime); err != nil {
		return nil, fmt.Errorf("failed to read lockup time: %w", err)
	}
	prop.LockupTime = time.Unix(int64(lockupTime), 0)

	if err := binary.Read(r, binary.LittleEndian, &prop.PokeCounter); err != nil {
		return nil, fmt.Errorf("failed to read poke counter: %w", err)
	}

	if n, err := r.Read(prop.SignatureAccount[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read signature account: %w", err)
	}

	return prop, nil
}
