package watchers

import (
	"encoding/hex"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
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
