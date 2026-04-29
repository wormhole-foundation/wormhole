package watchers

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// LogInvalidObservationRequest logs a validation failure for a raw gossiped
// observation request using a consistent set of structured fields.
func LogInvalidObservationRequest(logger *zap.Logger, req *gossipv1.ObservationRequest, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	if req == nil {
		logger.Error("invalid observation request", append(fields, zap.Bool("nilRequest", true))...)
		return
	}

	logger.Error("invalid observation request", append(fields,
		zap.Uint32("chainID", req.GetChainId()),
		zap.String("txID", hex.EncodeToString(req.GetTxHash())),
		zap.Int64("timestamp", req.GetTimestamp()),
	)...)
}

// ValidObservation is a watcher-scoped observation request that has passed the
// fundamental chain and request-shape validation checks.
type ValidObservation struct {
	chainID   vaa.ChainID
	txHash    []byte
	timestamp int64
}

// ValidateObservationRequest validates a raw observation request against the
// expected watcher chain and returns a copy-on-read validated value.
func ValidateObservationRequest(req *gossipv1.ObservationRequest, expectedChainID vaa.ChainID) (ValidObservation, error) {
	if req == nil {
		return ValidObservation{}, errors.New("observation request is nil")
	}

	chainID, err := vaa.KnownChainIDFromNumber(req.ChainId)
	if err != nil {
		return ValidObservation{}, fmt.Errorf("invalid chain id %d: %w", req.ChainId, err)
	}

	if chainID != expectedChainID {
		return ValidObservation{}, fmt.Errorf("unexpected chain id %v, expected %v", chainID, expectedChainID)
	}

	return ValidObservation{
		chainID:   chainID,
		txHash:    append([]byte(nil), req.TxHash...),
		timestamp: req.Timestamp,
	}, nil
}

// ChainID returns the validated chain ID.
func (v ValidObservation) ChainID() vaa.ChainID {
	return v.chainID
}

// TxHash returns a defensive copy of the validated transaction hash bytes.
func (v ValidObservation) TxHash() []byte {
	return append([]byte(nil), v.txHash...)
}

// Timestamp returns the validated wire timestamp.
func (v ValidObservation) Timestamp() int64 {
	return v.timestamp
}

// ZapFields takes some zap fields and appends zap fields related to the
// validated observation request.
func (v ValidObservation) ZapFields(fields ...zap.Field) []zap.Field {
	return append(fields,
		zap.Uint32("chainID", uint32(v.chainID)),
		zap.String("chain", v.chainID.String()),
		zap.String("txID", hex.EncodeToString(v.txHash)),
		zap.Int64("timestamp", v.timestamp),
	)
}

// RequireTxHashLength enforces watcher-specific transaction identifier sizes.
func (v ValidObservation) RequireTxHashLength(lengths ...int) error {
	for _, length := range lengths {
		if len(v.txHash) == length {
			return nil
		}
	}

	return fmt.Errorf("unexpected tx hash length %d", len(v.txHash))
}

// ValidateReobservedMessage ensures that a message publication is safe to
// publish as a reobservation for the validated watcher chain.
func ValidateReobservedMessage(observation ValidObservation, msg *common.MessagePublication) error {
	if msg == nil {
		return errors.New("message publication is nil")
	}

	if msg.EmitterChain != observation.chainID {
		return fmt.Errorf("message publication emitter chain %v does not match validated observation chain %v", msg.EmitterChain, observation.chainID)
	}

	return nil
}
