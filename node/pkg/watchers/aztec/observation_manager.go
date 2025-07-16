package aztec

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
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
	// Initialize metrics if not already done
	initMetrics()

	// Use the global metrics
	metrics := observationMetrics{
		messagesConfirmed: messagesConfirmedMetric,
	}

	return &observationManager{
		networkID: networkID,
		logger:    logger,
		metrics:   metrics,
	}
}

// IncrementMessagesConfirmed increases the counter for confirmed messages
func (m *observationManager) IncrementMessagesConfirmed() {
	m.metrics.messagesConfirmed.WithLabelValues(m.networkID).Inc()
	m.logger.Debug("Incremented messages confirmed counter")
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
	if len(extLog.Log.Fields) == 0 {
		return nil
	}

	// Extract event parameters
	params, err := w.parseLogParameters(extLog.Log.Fields)
	if err != nil {
		return fmt.Errorf("failed to parse log parameters: %v", err)
	}

	// Set the transaction ID from the block info
	params.TxID = blockInfo.TxHash

	// Create message payload (now including the txID)
	rawPayload := w.createPayload(extLog.Log.Fields, params.TxID)

	w.logger.Debug("Created payload",
		zap.Int("payloadLength", len(rawPayload)),
		zap.String("txID", params.TxID))

	// Extract structured data from the payload (accounting for txID at the beginning)
	arbitrumAddress, arbitrumChainID, amount, _, err := w.extractPayloadData(rawPayload)
	if err != nil {
		w.logger.Debug("Failed to extract payload data", zap.Error(err))
		// Continue with empty values for these fields
	} else {
		// Add the extracted values to the parameters
		params.ArbitrumAddress = arbitrumAddress
		params.ArbitrumChainID = arbitrumChainID
		params.Amount = amount
	}

	// Create a unique ID for this observation
	observationID := CreateObservationID(params.SenderAddress.String(), params.Sequence, extLog.ID.BlockNumber)

	// Log relevant information about the message
	w.logger.Info("Processing message",
		zap.Stringer("emitter", params.SenderAddress),
		zap.Uint64("sequence", params.Sequence),
		zap.String("arbitrumAddress", fmt.Sprintf("0x%x", params.ArbitrumAddress)),
		zap.Uint16("arbitrumChainID", params.ArbitrumChainID),
		zap.Uint64("amount", params.Amount))

	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Since we're processing finalized blocks, we can publish immediately
	// regardless of the original consistency level
	if err := w.publishObservation(ctx, params, rawPayload, blockInfo, observationID); err != nil {
		return fmt.Errorf("failed to publish observation: %v", err)
	}

	return nil
}

// extractPayloadData parses the structured payload to extract key information
// Modified to account for txID at the beginning of the payload
func (w *Watcher) extractPayloadData(payload []byte) ([]byte, uint16, uint64, []byte, error) {
	// Skip past the txID (first 32 bytes)
	txIDOffset := 32

	if len(payload) < txIDOffset+93 { // Need txID + at least 3 full 31-byte arrays
		return nil, 0, 0, nil, fmt.Errorf("payload too short, expected at least %d bytes, got %d", txIDOffset+93, len(payload))
	}

	// Extract the txID for debugging
	txID := payload[:txIDOffset]
	w.logger.Debug("Extracted txID from payload", zap.String("txID", fmt.Sprintf("0x%x", txID)))

	// Each array is 31 bytes long for address and chain ID
	const arraySize = 31

	// Extract Arbitrum address (first 20 bytes after txID)
	arbitrumAddress := make([]byte, 20)
	copy(arbitrumAddress, payload[txIDOffset:txIDOffset+20])

	// Extract Arbitrum chain ID (first 2 bytes of second array after txID)
	chainIDLower := uint16(payload[txIDOffset+arraySize])
	chainIDUpper := uint16(payload[txIDOffset+arraySize+1])
	arbitrumChainID := (chainIDUpper << 8) | chainIDLower

	// The amount is the first byte of the third array after txID
	amount := uint64(0)
	if len(payload) >= txIDOffset+2*arraySize+1 {
		// Just read the first byte as the amount value
		amount = uint64(payload[txIDOffset+2*arraySize])
	}

	// The verification data starts after the amount array
	verificationDataStart := txIDOffset + 3*arraySize
	verificationDataLength := len(payload) - verificationDataStart
	verificationData := make([]byte, verificationDataLength)

	if verificationDataLength > 0 {
		copy(verificationData, payload[verificationDataStart:])
	}

	// Log what we've extracted at debug level
	w.logger.Debug("Extracted payload data",
		zap.String("arbitrumAddress", fmt.Sprintf("0x%x", arbitrumAddress)),
		zap.Uint16("arbitrumChainID", arbitrumChainID),
		zap.Uint64("amount", amount),
		zap.Int("verificationDataLength", verificationDataLength))

	return arbitrumAddress, arbitrumChainID, amount, verificationData, nil
}

// parseLogParameters extracts parameters from a log entry
func (w *Watcher) parseLogParameters(logEntries []string) (LogParameters, error) {
	if len(logEntries) < 4 {
		return LogParameters{}, fmt.Errorf("log has insufficient entries: %d", len(logEntries))
	}

	// First value is the sender
	senderHex := strings.TrimPrefix(logEntries[0], "0x")
	senderBytes, err := hex.DecodeString(senderHex)
	if err != nil {
		return LogParameters{}, &ErrParsingFailed{
			What: "sender address",
			Err:  err,
		}
	}

	var senderAddress vaa.Address
	copy(senderAddress[:], senderBytes)

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
		Nonce:            safeUint64ToUint32(nonce),
		ConsistencyLevel: safeUint64ToUint8(consistencyLevel),
		Amount:           0, // Initialize with 0, will be set later
	}, nil
}

// createPayload processes log entries that contain field elements into a byte payload
// Modified to include txID at the beginning of the payload
func (w *Watcher) createPayload(logEntries []string, txID string) []byte {
	// Start by adding the txID to the payload
	txIDHex := strings.TrimPrefix(txID, "0x")
	txIDBytes, err := hex.DecodeString(txIDHex)
	if err != nil {
		w.logger.Debug("Failed to decode txID hex, using empty txID", zap.Error(err))
		txIDBytes = make([]byte, 0)
	}

	// Create a 32-byte array for txID
	paddedTxID := make([]byte, 32)
	// Copy txID bytes (this will handle padding correctly)
	copy(paddedTxID, txIDBytes)

	// Initialize payload with the txID
	payload := paddedTxID

	// Now continue with the rest of the payload
	remainingPayload := make([]byte, 0, w.config.PayloadInitialCap)

	// Skip the first 5 entries which are metadata (sender, sequence, nonce, consistency level, timestamp)
	for i, entry := range logEntries[5:] {
		// Clean up the entry - remove 0x
		entry = strings.TrimPrefix(entry, "0x")

		// Try to decode as hex
		bytes, err := hex.DecodeString(entry)
		if err != nil {
			w.logger.Debug("Failed to decode hex entry", zap.Error(err), zap.Int("entryIndex", i+5))
			continue
		}

		// Remove leading zeros
		for len(bytes) > 0 && bytes[0] == 0 {
			bytes = bytes[1:]
		}

		// Reverse the bytes to correct the order
		for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
			bytes[i], bytes[j] = bytes[j], bytes[i]
		}

		// Special handling for amount (the 3rd array, index 2)
		if i == 2 {
			// This is the amount field (third array)
			// Ensure it's padded to 32 bytes
			// First, add the current bytes
			remainingPayload = append(remainingPayload, bytes...)

			// Then add padding to make it 32 bytes total
			// Don't include Jack after it - move Jack to the next 32-byte chunk
			paddingNeeded := 32 - len(bytes)
			padding := make([]byte, paddingNeeded)
			remainingPayload = append(remainingPayload, padding...)

			// Continue to next entry - skip the normal append
			continue
		}

		// If this is the entry after the amount (Jack), ensure it starts on a new 32-byte boundary
		if i == 3 {
			// This is where Jack would start
			// Calculate padding needed to align to next 32-byte boundary
			currentLength := len(remainingPayload)
			paddingNeeded := (32 - (currentLength % 32)) % 32
			if paddingNeeded > 0 {
				padding := make([]byte, paddingNeeded)
				remainingPayload = append(remainingPayload, padding...)
			}
		}

		// Add to payload
		remainingPayload = append(remainingPayload, bytes...)
	}

	// Combine txID and remainingPayload
	payload = append(payload, remainingPayload...)

	// Log the final payload length at debug level
	w.logger.Debug("Payload created",
		zap.Int("length", len(payload)))

	return payload
}

// ParseHexUint64 converts a hex string to uint64
func ParseHexUint64(hexStr string) (uint64, error) {
	// Remove "0x" prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	// Parse the hex string to uint64
	value, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, &ErrParsingFailed{
			What: "hex uint64",
			Err:  err,
		}
	}

	return value, nil
}

// safeUint64ToUint32 safely converts uint64 to uint32
func safeUint64ToUint32(value uint64) uint32 {
	if value > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(value)
}

// safeUint64ToUint8 safely converts uint64 to uint8
func safeUint64ToUint8(value uint64) uint8 {
	if value > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(value)
}
