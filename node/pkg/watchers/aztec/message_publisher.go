package aztec

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// publishObservation creates and publishes a message observation
func (w *Watcher) publishObservation(ctx context.Context, params LogParameters, payload []byte, blockInfo BlockInfo, observationID string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Convert transaction hash to byte array for txID
	txID, err := hex.DecodeString(strings.TrimPrefix(blockInfo.TxHash, "0x"))
	if err != nil {
		w.logger.Error("Failed to decode transaction hash", zap.Error(err))
		// Fall back to default
		txID = []byte{0x0}
	}

	// Check for context cancellation after potentially long operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Create the observation
	observation := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(blockInfo.Timestamp), 0),
		Nonce:            params.Nonce,
		Sequence:         params.Sequence,
		EmitterChain:     w.config.ChainID,
		EmitterAddress:   params.SenderAddress,
		Payload:          payload,
		ConsistencyLevel: params.ConsistencyLevel,
		IsReobservation:  false,
	}

	// Increment metrics
	w.observationManager.IncrementMessagesConfirmed()

	// Log the observation
	w.logger.Info("Message observed",
		zap.String("id", observationID),
		zap.String("txHash", observation.TxIDString()),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress))

	// Send to the message channel
	select {
	case w.msgC <- observation:
		// Message sent successfully
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
