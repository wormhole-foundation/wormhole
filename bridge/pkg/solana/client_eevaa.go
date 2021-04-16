package ethereum

import (
	"context"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

func (s *SolanaWatcher) RunEevaaBridge(ctx context.Context) error {
	// TODO(drozdziak1): Send heartbeat
	// eevaaBridgeAddr := base58.Encode(s.eevaaBridge[:])

	// p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{

	// 	BridgeAddress: bridgeAddr,
	// })

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
					// p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSolana, &gossipv1.Heartbeat_Network{
					// 	Height:        int64(slot),
					// 	BridgeAddress: eevaaBridgeAddr,
					// })

					logger.Info("current Solana height", zap.Uint64("slot", uint64(slot)))

					// Find TransferOutProposal accounts without a VAA
					rCtx, cancel = context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					start = time.Now()

					accounts, err := rpcClient.GetProgramAccounts(rCtx, s.eevaaBridge, &rpc.GetProgramAccountsOpts{
						Commitment: rpc.CommitmentMax, // TODO: deprecated, use Finalized
						Filters:    []rpc.RPCFilter{}, // {
						// 	{
						// 		DataSize: 1184, // Search for TransferOutProposal accounts
						// 	},
						// 	{
						// 		Memcmp: &rpc.RPCFilterMemcmp{
						// 			Offset: 1140,                      // Offset of VaaTime
						// 			Bytes:  solana.Base58{0, 0, 0, 0}, // VAA time is 0 when no VAA is present
						// 		},
						// 	},
						// },
					})

					if err != nil {
						logger.Warn("Failed to get program accounts", zap.Error(err))
					}

					logger.Info("Got program accounts", zap.Int("count", len(accounts)))

					for _, acc := range accounts {
						eevaa, err := common.ParseEevaa(acc.Account.Data)
						if err != nil {
							solanaAccountSkips.WithLabelValues("parse_eevaa").Inc()
							logger.Warn(
								"failed to parse EEVAA",
								zap.Stringer("account", acc.Pubkey),
								zap.Error(err),
							)
							continue
						}
						logger.Info("Processing EEVAA", zap.Stringer("eevaa", eevaa))

						s.eevaaC <- eevaa

						// VAA submitted
						// if eevaa.VaaTime.Unix() != 0 {
						// 	solanaAccountSkips.WithLabelValues("is_submitted_vaa").Inc()
						// 	continue
						// }

						// var txHash eth_common.Hash
						// copy(txHash[:], acc.Pubkey[:])

						// lock := &common.ChainLock{
						// 	TxHash:        txHash,
						// 	Timestamp:     eevaa.LockupTime,
						// 	Nonce:         eevaa.Nonce,
						// 	SourceAddress: eevaa.SourceAddress,
						// 	TargetAddress: eevaa.ForeignAddress,
						// 	SourceChain:   vaa.ChainIDSolana,
						// 	TargetChain:   eevaa.ToChainID,
						// 	TokenChain:    eevaa.Asset.Chain,
						// 	TokenAddress:  eevaa.Asset.Address,
						// 	TokenDecimals: eevaa.Asset.Decimals,
						// 	Amount:        eevaa.Amount,
						// }

						// solanaLockupsConfirmed.Inc()
						// logger.Info("found lockup without VAA", zap.Stringer("lockup_address", acc.Pubkey))
						// s.lockEvent <- lock
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
