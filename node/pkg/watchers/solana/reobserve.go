package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// handleReobservationRequest performs a reobservation request and publishes any observed transactions.
func (s *SolanaWatcher) handleReobservationRequest(chainId vaa.ChainID, txID []byte, rpcClient *rpc.Client) (numObservations uint32, err error) {
	if chainId != s.chainID {
		return 0, fmt.Errorf("unexpected chain id: %v", chainId)
	}
	if len(txID) == SolanaAccountLen { // Request by account ID
		acc := solana.PublicKeyFromBytes(txID)
		s.logger.Info("received observation request with account id", zap.String("account", acc.String()))
		rCtx, cancel := context.WithTimeout(s.ctx, rpcTimeout)
		numObservations, _ = s.fetchMessageAccount(rCtx, rpcClient, acc, 0, true)
		cancel()
	} else if len(txID) == SolanaSignatureLen { // Request by transaction ID
		signature := solana.SignatureFromBytes(txID)
		s.logger.Info("received observation request with transaction id", zap.Stringer("signature", signature))
		rCtx, cancel := context.WithTimeout(s.ctx, rpcTimeout)
		version := uint64(0)
		result, err := rpcClient.GetTransaction(
			rCtx,
			signature,
			&rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &version,
				Encoding:                       solana.EncodingBase64,
			},
		)
		cancel()
		if err != nil {
			return 0, fmt.Errorf("failed to get transaction for observation request: %v", err)
		}

		tx, err := result.Transaction.GetTransaction()
		if err != nil {
			return 0, fmt.Errorf("failed to extract transaction for observation request: %v", err)
		}
		numObservations = s.processTransaction(s.ctx, rpcClient, tx, result.Meta, result.Slot, true)
	} else {
		return 0, fmt.Errorf("ignoring an observation request of unexpected length: %d", len(txID))
	}
	return numObservations, nil
}

// Reobserve is the interface for reobserving using a custom URL. It opens a connection to that URL and does the reobservation on it.
// This function does not use the passed in context because it can spawn go routines to fetch accounts and those may still be running
// when the passed in context gets deleted. This is also why we don't close the rpcClient in this function. It will get deleted / closed
// when the go routine exits.
func (s *SolanaWatcher) Reobserve(_ context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	s.logger.Info("received a request to reobserve using a custom endpoint", zap.Stringer("chainID", chainID), zap.Any("txID", txID), zap.String("url", customEndpoint))
	rpcClient := rpc.New(customEndpoint)
	//nolint:contextcheck // See comment above for the reason why we don't use the passed in context.
	return s.handleReobservationRequest(chainID, txID, rpcClient)
}
