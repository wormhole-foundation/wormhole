package watchers

import (
	"bytes"
	"encoding/hex"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

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
		return ValidObservation{}, ErrNilObservationRequest
	}

	chainID, err := vaa.KnownChainIDFromNumber(req.ChainId)
	if err != nil {
		return ValidObservation{}, InvalidChainIDError(req.ChainId, err)
	}

	if chainID != expectedChainID {
		return ValidObservation{}, UnexpectedChainIDError(chainID, expectedChainID)
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

	return UnexpectedTxHashLengthError(len(v.txHash))
}

// ValidateReobservedMessage ensures that a message publication is safe to
// publish as a reobservation for the validated watcher chain.
func ValidateReobservedMessage(observation ValidObservation, msg *common.MessagePublication) error {
	if msg == nil {
		return ErrNilMessagePublication
	}

	if !bytes.Equal(msg.TxID, observation.txHash) {
		return MessagePublicationTxIDMismatchError(msg, observation)
	}

	if msg.EmitterChain != observation.chainID {
		return ReobservedMessageChainMismatchError(msg.EmitterChain, observation.chainID)
	}

	return nil
}
