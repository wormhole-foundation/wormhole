package aptos

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// handleReobservationRequest performs a reobservation against the given Aptos RPC base URL
// and publishes any observed messages. Returns the number of messages successfully observed.
func (e *Watcher) handleReobservationRequest(logger *zap.Logger, chainID vaa.ChainID, txHash []byte, rpcURL string) (uint32, error) {
	// The caller is expected to send us only requests for our chainID.
	if chainID != e.chainID {
		return 0, fmt.Errorf("unexpected chain id: %v", chainID)
	}

	// Aptos's TxID is a uint64. Historically, all TxIDs used a fixed 32-byte hash type.
	// This parsing is leftover from that time period. It should be possible to refactor
	// this code such that the TxID received from p2p is exactly 8 bytes, which would
	// obviate the need for the below bounds check and parsing.
	//
	// SECURITY: This acts as a bounds check for the BigEndian.Uint64 call below.
	const AptosTxIDExpectedLen = 32
	if len(txHash) < AptosTxIDExpectedLen {
		return 0, fmt.Errorf("invalid TxID: too short")
	}

	// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
	nativeSeq := binary.BigEndian.Uint64(txHash[24:])

	logger.Info("Received obsv request",
		zap.Uint64("tx_hash", nativeSeq),
		zap.String("rpc", rpcURL),
	)

	// SECURITY: the API guarantees that we only get the events from the right contract.
	eventsEndpoint := fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, rpcURL, e.aptosAccount, e.aptosHandle)
	s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, nativeSeq)

	body, err := e.retrievePayload(s)
	if err != nil {
		return 0, fmt.Errorf("retrievePayload: %w", err)
	}

	if !gjson.Valid(string(body)) {
		return 0, fmt.Errorf("invalid JSON in reobservation response: %s", string(body))
	}

	var numObservations uint32
	for _, chunk := range gjson.ParseBytes(body).Array() {
		newSeq := chunk.Get("sequence_number")
		if !newSeq.Exists() {
			break
		}

		if newSeq.Uint() != nativeSeq {
			return numObservations, fmt.Errorf("newSeq != nativeSeq")
		}

		data := chunk.Get("data")
		if !data.Exists() {
			break
		}
		if e.observeData(logger, data, nativeSeq, true) {
			numObservations++
		}
	}
	return numObservations, nil
}

// Reobserve is the interface for reobserving using a custom URL. It performs the reobservation against that URL.
func (e *Watcher) Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	logger := e.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Info("received a request to reobserve using a custom endpoint",
		zap.Stringer("chainID", chainID),
		zap.Any("txID", txID),
		zap.String("url", customEndpoint),
	)

	// Verify that this endpoint is for the correct chain.
	if err := e.verifyAptosChainID(ctx, logger, customEndpoint); err != nil {
		return 0, fmt.Errorf("failed to verify aptos chain id: %w", err)
	}

	return e.handleReobservationRequest(logger, chainID, txID, customEndpoint)
}
