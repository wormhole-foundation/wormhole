package txverifier

// TODOs:
//	* balances on Sui are stored as u64's. Consider using uint64 instead of big.Int
//  * replace errors with error.join() like EVM

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

			// If there is an error decoding the payload, skip the event
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
			logger.Debug("Event does not match the criteria", zap.String("event type", *event.Type), zap.String("event sender", *event.Message.Sender))
		}
	}

	return requestedOutOfBridge, numEventsProcessed
}

func (s *SuiTransferVerifier) processObjectUpdates(ctx context.Context, objectChanges []ObjectChange, logger *zap.Logger) (transferredIntoBridge map[string]*big.Int, numChangesProcessed uint) {
	transferredIntoBridge = make(map[string]*big.Int)

	for _, objectChange := range objectChanges {
		// Check that the type information is correct.
		if !objectChange.ValidateTypeInformation(s.suiTokenBridgeContract) {
			continue
		}

		// Get the previous version of the object.
		resp, err := s.suiApiConnection.TryMultiGetPastObjects(ctx, objectChange.ObjectId, objectChange.Version, objectChange.PreviousVersion)
		if err != nil {
			logger.Error("Error in getting past objects",
				zap.String("objectId", objectChange.ObjectId),
				zap.String("currentVersion", objectChange.Version),
				zap.String("previousVersion", objectChange.PreviousVersion),
				zap.Error(err))
			continue
		}

		decimals, err := resp.GetDecimals()
		if err != nil {
			logger.Error("Error in getting decimals", zap.Error(err))
			continue
		}

		address, err := resp.GetTokenAddress()
		if err != nil {
			logger.Error("Error in getting token address", zap.Error(err))
			continue
		}

		chain, err := resp.GetTokenChain()
		if err != nil {
			logger.Error("Error in getting token chain", zap.Error(err))
			continue
		}

		// Get the balance difference
		balanceDiff, err := resp.GetBalanceDiff()
		if err != nil {
			logger.Error("Error in getting balance difference", zap.Error(err))
			continue
		}

		normalized := normalize(balanceDiff, decimals)

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

// ProcessDigestFlagOnly wraps ProcessDigest and only returns true or false, indicating that a specific digest
// passed validation.
func (s *SuiTransferVerifier) ProcessDigestFlagOnly(ctx context.Context, digest string, logger *zap.Logger) bool {
	_, err := s.ProcessDigest(ctx, digest, logger)

	if err != nil {
		logger.Error("Failed to process digest", zap.Error(err))
		return false
	}

	return true
}

func (s *SuiTransferVerifier) ProcessDigest(ctx context.Context, digest string, logger *zap.Logger) (uint, error) {
	logger.Info("processing digest", zap.String("txDigest", digest))

	// Get the transaction block
	txBlock, err := s.suiApiConnection.GetTransactionBlock(ctx, digest)

	if err != nil {
		logger.Error("Error in retrieving transaction block", zap.Error(err))
		return 0, errors.New("failed to retrieve transaction block")
	}

	// process all events, indicating funds that are leaving the chain
	requestedOutOfBridge, numEventsProcessed := s.processEvents(txBlock.Result.Events, logger)

	// process all object changes, indicating funds that are entering the chain
	transferredIntoBridge, numChangesProcessed := s.processObjectUpdates(ctx, txBlock.Result.ObjectChanges, logger)

	for key, amountOut := range requestedOutOfBridge {
		keyParts := strings.Split(key, "-")

		if _, exists := transferredIntoBridge[key]; !exists {
			// This implies that a token leaving the bridge was never deposited into it.
			logger.Error("token bridge transfer requested for tokens that were never deposited",
				zap.String("tokenAddress", keyParts[0]))

			return 0, errors.New("transfer-out request for tokens that were never deposited")
		}

		amountIn := transferredIntoBridge[key]

		if amountOut.Cmp(amountIn) > 0 {
			// Implies that more tokens are being requested out of the bridge than were deposited into it.
			logger.Error("token bridge transfer requested for an amount larger than what was deposited",
				zap.String("tokenAddress", keyParts[0]), zap.String("amountOut", amountOut.String()), zap.String("amountIn", amountIn.String()))
			return 0, errors.New("requested amount out is larger than amount in")
		}

		logger.Info("bridge request processed",
			zap.String("tokenAddress", keyParts[0]),
			zap.String("chain", keyParts[1]),
			zap.String("amountOut", amountOut.String()),
			zap.String("amountIn", amountIn.String()))
	}

	logger.Info("Digest processed", zap.String("txDigest", digest), zap.Uint("numEventsProcessed", numEventsProcessed), zap.Uint("numChangesProcessed", numChangesProcessed))

	return numEventsProcessed, nil
}

type SuiApiResponse interface {
	GetError() error
}

// TODO: add context
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

	// Check if an error message exists
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
