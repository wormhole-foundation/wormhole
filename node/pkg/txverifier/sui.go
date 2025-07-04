package txverifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Errors
var (
	// Internal errors that can occur in the verifier.
	ErrFailedToRetrieveTxBlock = errors.New("failed to retrieve transaction block")
)

// Global variables
var (
	suiModule    = "publish_message"
	suiEventName = "WormholeMessage"
)

type SuiTransferVerifier struct {
	// Used to create the event filter.
	suiCoreContract string
	// Used to check the emitter of the `WormholeMessage` event.
	suiTokenBridgeEmitter string
	// Used to match the owning package of native and wrapped asset types.
	suiTokenBridgeContract string
	suiEventType           string
	suiApiConnection       SuiApiInterface
}

func NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeContract string, suiApiConnection SuiApiInterface) *SuiTransferVerifier {
	return &SuiTransferVerifier{
		suiCoreContract:        suiCoreContract,
		suiTokenBridgeEmitter:  suiTokenBridgeEmitter,
		suiTokenBridgeContract: suiTokenBridgeContract,
		suiEventType:           fmt.Sprintf("%s::%s::%s", suiCoreContract, suiModule, suiEventName),
		suiApiConnection:       suiApiConnection,
	}
}

func (s *SuiTransferVerifier) GetTokenBridgeEmitter() string {
	return s.suiTokenBridgeEmitter
}

// Filter to be used for querying events
// The `MoveEventType` filter doesn't seem to be available in the documentation. However, there is an example
// showing the inclusion of `type` in the `MoveModule` filter.
// Reference: https://docs.sui.io/guides/developer/sui-101/using-events#query-events-with-rpc
func (s *SuiTransferVerifier) GetEventFilter() string {
	return fmt.Sprintf(`
	{
		"MoveModule":{
			"package":"%s",
			"module":"%s",
			"type":"%s"
		}
	}`, s.suiCoreContract, suiModule, s.suiEventType)
}

// processEvents takes a list of events and processes them to determine the amount requested out of the bridge. It returns a mapping
// that maps the token address and chain ID to the amount requested out of the bridge. It does not return an error, because any faulty
// events can be skipped, since they would likely fail being processed by the guardian as well. Debug level logging can be used to
// reveal any potential locations where errors are occurring.
func (s *SuiTransferVerifier) processEvents(events []SuiEvent, logger *zap.Logger) (requestedOutOfBridge map[string]*big.Int, numEventsProcessed uint) {
	// Initialize the map to store the amount requested out of the bridge
	requestedOutOfBridge = make(map[string]*big.Int)

	// Filter events that have the sui token bridge emitter as the sender in the message. The events indicate
	// how much is going to leave the network.
	for _, event := range events {

		// If any of these event parameters are nil, skip the event
		if event.Message == nil || event.Message.Sender == nil || event.Type == nil {
			continue
		}

		// Only process the event if it is a WormholeMessage event from the token bridge emitter
		if *event.Type == s.suiEventType && *event.Message.Sender == s.suiTokenBridgeEmitter {

			// Parse the wormhole message. vaa.IsTransfer can be omitted, since this is done
			// inside `DecodeTransferPayloadHdr` already.
			hdr, err := vaa.DecodeTransferPayloadHdr(event.Message.Payload)

			// If there is an error decoding the payload, skip the event. One reason for a potential
			// failure in decoding is that an attestation of a token was requested.
			if err != nil {
				logger.Debug("Error decoding payload", zap.Error(err))
				continue
			}

			// Add the key if it does not exist yet
			key := fmt.Sprintf(KEY_FORMAT, hdr.OriginAddress.String(), hdr.OriginChain)
			if _, exists := requestedOutOfBridge[key]; !exists {
				requestedOutOfBridge[key] = big.NewInt(0)
			}

			// Add the amount requested out of the bridge
			requestedOutOfBridge[key] = new(big.Int).Add(requestedOutOfBridge[key], hdr.Amount)

			numEventsProcessed++
		} else {
			logger.Debug("Event does not match the criteria",
				zap.String("event type", *event.Type),
				zap.String("event sender", *event.Message.Sender),
			)
		}
	}

	return requestedOutOfBridge, numEventsProcessed
}

// processObjectUpdates iterates through all object changes present in the PTB. It searches for tokens
// that belong to the token bridge, and then calculates the balance difference between the token object's
// previous version and current version.
//
// The `transferredIntoBridge` return value is a mapping from token address to transfer delta, indicating
// the amount of funds deposited into the token bridge in this transaction block.
func (s *SuiTransferVerifier) processObjectUpdates(ctx context.Context, objectChanges []ObjectChange, logger *zap.Logger) (transferredIntoBridge map[string]*big.Int, numChangesProcessed uint) {
	transferredIntoBridge = make(map[string]*big.Int)

	for _, objectChange := range objectChanges {
		// Check that the type information is correct. Doing it here means it's not necessary to do it
		// again, even in the case where `GetObject` is used for a single object instead of getting past
		// objects.
		if !objectChange.ValidateTypeInformation(s.suiTokenBridgeContract) {
			continue
		}

		if objectChange.PreviousVersion == "" {
			logger.Warn("No previous version of asset available",
				zap.String("objectId", objectChange.ObjectId),
				zap.String("currentVersion", objectChange.Version))
			continue
		}

		// Get the previous version of the object. This makes a call to the Sui API.
		resp, err := s.suiApiConnection.TryMultiGetPastObjects(ctx, objectChange.ObjectId, objectChange.Version, objectChange.PreviousVersion)
		if err != nil {
			logger.Error("Error getting past objects",
				zap.String("objectId", objectChange.ObjectId),
				zap.String("currentVersion", objectChange.Version),
				zap.String("previousVersion", objectChange.PreviousVersion),
				zap.Error(err))
			continue
		}

		decimals, err := resp.GetDecimals()
		if err != nil {
			logger.Error("Error getting decimals", zap.Error(err))
			continue
		}

		address, err := resp.GetTokenAddress()
		if err != nil {
			logger.Error("Error getting token address", zap.Error(err))
			continue
		}

		chain, err := resp.GetTokenChain()
		if err != nil {
			logger.Error("Error getting token chain", zap.Error(err))
			continue
		}

		// Get the change in balance
		balanceChange, err := resp.GetBalanceChange()
		if err != nil {
			logger.Error("Error getting balance difference", zap.Error(err))
			continue
		}

		normalized := normalize(balanceChange, decimals)

		// Add the key if it does not exist yet
		key := fmt.Sprintf(KEY_FORMAT, address, chain)

		// Add the normalized amount to the transferredIntoBridge map
		// Intentionally use 'Set' instead of 'Add' because there should only be a single objectChange per token
		var amount big.Int
		transferredIntoBridge[key] = amount.Set(normalized)

		// Increment the number of changes processed
		numChangesProcessed++
	}

	return transferredIntoBridge, numChangesProcessed
}

func (s *SuiTransferVerifier) ProcessDigest(ctx context.Context, digest string, logger *zap.Logger) (bool, error) {
	_, verified, err := s.ProcessDigestWithCount(ctx, digest, logger)
	return verified, err
}

func (s *SuiTransferVerifier) ProcessDigestWithCount(ctx context.Context, digest string, logger *zap.Logger) (uint, bool, error) {
	count, verified, err := s.processDigestInternal(ctx, digest, logger)

	// check if the error is an invariant violation
	var invariantError *InvariantError
	if errors.As(err, &invariantError) {
		logger.Error("Sui txverifier invariant violated", zap.String("txdigest", digest), zap.String("invariant", invariantError.Msg))
		return count, false, nil
	} else {
		return count, verified, err
	}
}

// Return conditions:
//
//	_, true, nil - verification succeeded
//	_, false, nil - verification failed
//	_, false, err - verification could not be performed due to an internal error
func (s *SuiTransferVerifier) processDigestInternal(ctx context.Context, digest string, logger *zap.Logger) (uint, bool, error) {
	logger.Debug("processing digest", zap.String("txDigest", digest))

	// Get the transaction block
	txBlock, err := s.suiApiConnection.GetTransactionBlock(ctx, digest)

	if err != nil {
		logger.Error("failed to retrieve transaction block",
			zap.String("txDigest", digest),
			zap.Error(err),
		)
		return 0, false, ErrFailedToRetrieveTxBlock
	}

	// Process all events, indicating funds that are leaving the chain
	requestedOutOfBridge, numEventsProcessed := s.processEvents(txBlock.Result.Events, logger)

	if numEventsProcessed == 0 {
		// No valid events were identified, so the digest does not require further processing.
		return 0, true, nil
	}

	// Process all object changes, specifically looking for transfers into the token bridge
	transferredIntoBridge, numChangesProcessed := s.processObjectUpdates(ctx, txBlock.Result.ObjectChanges, logger)

	for key, amountOut := range requestedOutOfBridge {

		if _, exists := transferredIntoBridge[key]; !exists {
			// This implies that a token leaving the bridge was never deposited into it.
			invariantError := &InvariantError{Msg: INVARIANT_NO_DEPOSIT}

			return 0, false, invariantError
		}

		amountIn := transferredIntoBridge[key]

		if amountOut.Cmp(amountIn) > 0 {
			// Implies that more tokens are being requested out of the bridge than were deposited into it.
			invariantError := &InvariantError{Msg: INVARIANT_INSUFFICIENT_DEPOSIT}

			return 0, false, invariantError
		}

		logger.Info("bridge request processed",
			zap.String("tokenAddress-chain", key),
			zap.String("amountOut", amountOut.String()),
			zap.String("amountIn", amountIn.String()))
	}

	logger.Debug("Digest processed", zap.String("txDigest", digest), zap.Uint("numEventsProcessed", numEventsProcessed), zap.Uint("numChangesProcessed", numChangesProcessed))

	return numEventsProcessed, true, nil
}

type SuiApiResponse interface {
	GetError() error
}

func suiApiRequest[T SuiApiResponse](ctx context.Context, rpc string, method string, params string) (T, error) {
	var defaultT T

	// Create the request
	requestBody := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "%s", "params": %s}`, method, params)

	req, err := http.NewRequestWithContext(ctx, "POST", rpc, strings.NewReader(requestBody))
	if err != nil {
		return defaultT, fmt.Errorf("cannot create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return defaultT, fmt.Errorf("cannot send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return defaultT, fmt.Errorf("cannot read response: %w", err)
	}

	// Parse the response
	var res T
	err = json.Unmarshal(body, &res)
	if err != nil {
		return defaultT, fmt.Errorf("cannot parse response: %w", err)
	}

	// Check if the API returned an error
	if res.GetError() != nil {
		return defaultT, fmt.Errorf("error from Sui RPC: %w", res.GetError())
	}

	return res, nil
}

type SuiApiConnection struct {
	rpc string
}

func NewSuiApiConnection(rpc string) SuiApiInterface {
	return &SuiApiConnection{rpc: rpc}
}

func (s *SuiApiConnection) GetTransactionBlock(ctx context.Context, txDigest string) (SuiGetTransactionBlockResponse, error) {
	method := "sui_getTransactionBlock"
	params := fmt.Sprintf(`[
				"%s", 
				{
					"showObjectChanges":true,
					"showEvents": true
				}
			]`, txDigest)

	return suiApiRequest[SuiGetTransactionBlockResponse](ctx, s.rpc, method, params)
}

func (s *SuiApiConnection) QueryEvents(ctx context.Context, filter string, cursor string, limit int, descending bool) (SuiQueryEventsResponse, error) {
	method := "suix_queryEvents"
	params := fmt.Sprintf(`[%s, %s, %d, %t]`, filter, cursor, limit, descending)

	return suiApiRequest[SuiQueryEventsResponse](ctx, s.rpc, method, params)
}

func (s *SuiApiConnection) TryMultiGetPastObjects(ctx context.Context, objectId string, version string, previousVersion string) (SuiTryMultiGetPastObjectsResponse, error) {
	method := "sui_tryMultiGetPastObjects"
	params := fmt.Sprintf(`[
			[
				{"objectId" : "%s", "version" : "%s"},
				{"objectId" : "%s", "version" : "%s"}
			],
			{"showContent": true}
		]`, objectId, version, objectId, previousVersion)

	return suiApiRequest[SuiTryMultiGetPastObjectsResponse](ctx, s.rpc, method, params)
}
