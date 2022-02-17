package ethereum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/ethereum/abi"
	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"time"
)

var (
	// SECURITY: Hardcoded ABI identifier for the LogTokensLocked topic. When using the watcher, we don't need this
	// since the node will only hand us pre-filtered events. In this case, we need to manually verify it
	// since ParseLogTokensLocked will only verify whether it parses.
	logTokensLockedTopic = eth_common.HexToHash("0x6bbd554ad75919f71fd91bf917ca6e4f41c10f03ab25751596a22253bb39aab8")
)

// MessageEventsForTransaction returns the lockup events for a given transaction.
// Returns the block number and a list of MessagePublication events.
func MessageEventsForTransaction(
	ctx context.Context,
	c *ethclient.Client,
	contract eth_common.Address,
	tx eth_common.Hash) (uint64, []*common.ChainLock, error) {

	f, err := abi.NewAbiFilterer(contract, c)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create ABI filterer: %w", err)
	}

	// Get transactions logs from transaction
	receipt, err := c.TransactionReceipt(ctx, tx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Get block
	block, err := c.BlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get block: %w", err)
	}

	msgs := make([]*common.ChainLock, 0, len(receipt.Logs))

	// Extract logs
	for _, l := range receipt.Logs {
		// SECURITY: Skip logs not produced by our contract.
		if l.Address != contract {
			continue
		}

		if l == nil {
			continue
		}

		if l.Topics[0] != logTokensLockedTopic {
			continue
		}

		ev, err := f.ParseLogTokensLocked(*l)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to parse log: %w", err)
		}

		lock := &common.ChainLock{
			TxHash:        tx,
			Timestamp:     time.Unix(int64(block.Time()), 0),
			Nonce:         ev.Nonce,
			SourceAddress: ev.Sender,
			TargetAddress: ev.Recipient,
			SourceChain:   vaa.ChainIDEthereum,
			TargetChain:   vaa.ChainID(ev.TargetChain),
			TokenChain:    vaa.ChainID(ev.TokenChain),
			TokenAddress:  ev.Token,
			TokenDecimals: ev.TokenDecimals,
			Amount:        ev.Amount,
		}

		msgs = append(msgs, lock)
	}

	return receipt.BlockNumber.Uint64(), msgs, nil
}
