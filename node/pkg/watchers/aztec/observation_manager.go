package aztec

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Global metrics variables
var (
	messagesConfirmedMetric *prometheus.CounterVec
	metricsInitialized      sync.Once
)

// initMetrics initializes the metrics only once
func initMetrics() {
	metricsInitialized.Do(func() {
		messagesConfirmedMetric = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "wormhole_aztec_observations_confirmed_total",
				Help: "Total number of verified observations found for the chain",
			}, []string{"chain_name"})
	})
}

// ObservationManager handles storage and lifecycle of pending observations
type ObservationManager interface {
	IncrementMessagesConfirmed()
}

// observationManager is the implementation of ObservationManager
type observationManager struct {
	networkID string
	logger    *zap.Logger
	metrics   observationMetrics
}

// observationMetrics holds the Prometheus metrics for the observation manager
type observationMetrics struct {
	messagesConfirmed *prometheus.CounterVec
}

// NewObservationManager creates a new observation manager
func NewObservationManager(networkID string, logger *zap.Logger) ObservationManager {
	logger.Info("Creating new observation manager", zap.String("networkID", networkID))

	// Initialize metrics if not already done
	initMetrics()

	// Use the global metrics
	metrics := observationMetrics{
		messagesConfirmed: messagesConfirmedMetric,
	}

	logger.Info("Observation manager created with metrics initialized")

	return &observationManager{
		networkID: networkID,
		logger:    logger,
		metrics:   metrics,
	}
}

// IncrementMessagesConfirmed increases the counter for confirmed messages
func (m *observationManager) IncrementMessagesConfirmed() {
	m.metrics.messagesConfirmed.WithLabelValues(m.networkID).Inc()
	m.logger.Info("Incremented messages confirmed counter")
}

// processLog handles an individual log entry
func (w *Watcher) processLog(ctx context.Context, extLog ExtendedPublicLog, blockInfo BlockInfo) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Skip empty logs
	if len(extLog.Log.Log) == 0 {
		return nil
	}

	// Extract event parameters
	params, err := w.parseLogParameters(extLog.Log.Log)
	if err != nil {
		return fmt.Errorf("failed to parse log parameters: %v", err)
	}

	// Create message payload
	payload := w.createPayload(extLog.Log.Log[4:])

	// Create a unique ID for this observation
	observationID := CreateObservationID(params.SenderAddress.String(), params.Sequence, extLog.ID.BlockNumber)

	// Log relevant information about the message
	w.logger.Info("Processing message",
		zap.Stringer("emitter", params.SenderAddress),
		zap.Uint64("sequence", params.Sequence),
		zap.Uint8("consistencyLevel", params.ConsistencyLevel))

	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Since we're processing finalized blocks, we can publish immediately
	// regardless of the original consistency level
	if err := w.publishObservation(ctx, params, payload, blockInfo, observationID); err != nil {
		return fmt.Errorf("failed to publish observation: %v", err)
	}

	return nil
}

// parseLogParameters extracts parameters from a log entry
func (w *Watcher) parseLogParameters(logEntries []string) (LogParameters, error) {
	if len(logEntries) < 4 {
		return LogParameters{}, fmt.Errorf("log has insufficient entries: %d", len(logEntries))
	}

	// First value is the sender
	sender := logEntries[0]
	var senderAddress vaa.Address
	copy(senderAddress[:], sender)

	// Parse sequence
	sequence, err := ParseHexUint64(logEntries[1])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse sequence: %v", err)
	}

	// Parse nonce
	nonce, err := ParseHexUint64(logEntries[2])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse nonce: %v", err)
	}

	// Parse consistency level
	consistencyLevel, err := ParseHexUint64(logEntries[3])
	if err != nil {
		return LogParameters{}, fmt.Errorf("failed to parse consistencyLevel: %v", err)
	}

	return LogParameters{
		SenderAddress:    senderAddress,
		Sequence:         sequence,
		Nonce:            uint32(nonce),
		ConsistencyLevel: uint8(consistencyLevel),
	}, nil
}

// createPayload processes log entries into a byte payload
func (w *Watcher) createPayload(logEntries []string) []byte {
	payload := make([]byte, 0, w.config.PayloadInitialCap)

	for _, entry := range logEntries {
		hexStr := strings.TrimPrefix(entry, "0x")

		// Try to decode as hex
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			w.logger.Debug("Failed to decode hex", zap.String("entry", entry), zap.Error(err))
			continue
		}

		// Add to payload
		payload = append(payload, bytes...)
	}

	return payload
}
