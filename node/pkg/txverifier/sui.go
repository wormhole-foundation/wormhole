package txverifier

// TODOs:
//	* balances on Sui are stored as u64's. Consider using uint64 instead of big.Int

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Global variables
var (
	suiModule    = "publish_message"
	suiEventName = "WormholeMessage"
)

type SuiTransferVerifier struct {
	suiCoreContract        string
	suiTokenBridgeEmitter  string
	suiTokenBridgeContract string
	suiEventType           string
}

func NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeContract string) *SuiTransferVerifier {
	return &SuiTransferVerifier{
		suiCoreContract:        suiCoreContract,
		suiTokenBridgeEmitter:  suiTokenBridgeEmitter,
		suiTokenBridgeContract: suiTokenBridgeContract,
		suiEventType:           fmt.Sprintf("%s::%s::%s", suiCoreContract, suiModule, suiEventName),
	}
}

// func (s *SuiTransferVerifier) GetSuiEventType() string {
// 	return s.suiEventType
// }

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

func (s *SuiTransferVerifier) processObjectUpdates(objectChanges []ObjectChange, suiApiConnection SuiApiInterface, logger *zap.Logger) (transferredIntoBridge map[string]*big.Int, numChangesProcessed uint) {
	transferredIntoBridge = make(map[string]*big.Int)

	for _, objectChange := range objectChanges {
		// Check that the type information is correct.
		if !objectChange.ValidateTypeInformation(s.suiTokenBridgeContract) {
			continue
		}

		// Get the past objects
		resp, err := suiApiConnection.TryMultiGetPastObjects(objectChange.ObjectId, objectChange.Version, objectChange.PreviousVersion)

		if err != nil {
			logger.Error("Error in getting past objects", zap.Error(err))
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

func (s *SuiTransferVerifier) ProcessDigest(digest string, suiApiConnection SuiApiInterface, logger *zap.Logger) (uint, error) {
	// Get the transaction block
	txBlock, err := suiApiConnection.GetTransactionBlock(digest)

	if err != nil {
		logger.Fatal("Error in getting transaction block", zap.Error(err))
	}

	// process all events, indicating funds that are leaving the chain
	requestedOutOfBridge, numEventsProcessed := s.processEvents(txBlock.Result.Events, logger)

	// process all object changes, indicating funds that are entering the chain
	transferredIntoBridge, numChangesProcessed := s.processObjectUpdates(txBlock.Result.ObjectChanges, suiApiConnection, logger)

	// TODO: Using `Warn` for testing purposes. Update to Fatal? when ready to go into PR.
	// TODO: Revisit error handling here.
	for key, amountOut := range requestedOutOfBridge {

		if _, exists := transferredIntoBridge[key]; !exists {
			logger.Warn("transfer-out request for tokens that were never deposited",
				zap.String("tokenAddress", key))
			// TODO: Is it better to return or continue here?
			return 0, errors.New("transfer-out request for tokens that were never deposited")
			// continue
		}

		amountIn := transferredIntoBridge[key]

		if amountOut.Cmp(amountIn) > 0 {
			logger.Warn("requested amount out is larger than amount in")
			return 0, errors.New("requested amount out is larger than amount in")
		}

		keyParts := strings.Split(key, "-")
		logger.Info("bridge request processed",
			zap.String("tokenAddress", keyParts[0]),
			zap.String("chain", keyParts[1]),
			zap.String("amountOut", amountOut.String()),
			zap.String("amountIn", amountIn.String()))
	}

	//nolint:gosec
	logger.Info("Digest processed", zap.String("txDigest", digest), zap.Uint("numEventsProcessed", numEventsProcessed), zap.Uint("numChangesProcessed", numChangesProcessed))

	return numEventsProcessed, nil
}

type SuiApiResponse interface {
	GetError() error
}

func suiApiRequest[T SuiApiResponse](rpc string, method string, params string) (T, error) {
	var defaultT T

	// Create the request
	requestBody := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "%s", "params": %s}`, method, params)

	//nolint:noctx
	req, err := http.NewRequest("POST", rpc, strings.NewReader(requestBody))
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
	body, err := io.ReadAll(resp.Body)
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

func (s *SuiApiConnection) GetTransactionBlock(txDigest string) (SuiGetTransactionBlockResponse, error) {
	method := "sui_getTransactionBlock"
	params := fmt.Sprintf(`[
				"%s", 
				{
					"showObjectChanges":true,
					"showEvents": true
				}
			]`, txDigest)

	return suiApiRequest[SuiGetTransactionBlockResponse](s.rpc, method, params)
}

func (s *SuiApiConnection) QueryEvents(filter string, cursor string, limit int, descending bool) (SuiQueryEventsResponse, error) {
	method := "suix_queryEvents"
	params := fmt.Sprintf(`[%s, %s, %d, %t]`, filter, cursor, limit, descending)

	return suiApiRequest[SuiQueryEventsResponse](s.rpc, method, params)
}

func (s *SuiApiConnection) TryMultiGetPastObjects(objectId string, version string, previousVersion string) (SuiTryMultiGetPastObjectsResponse, error) {
	method := "sui_tryMultiGetPastObjects"
	params := fmt.Sprintf(`[
			[
				{"objectId" : "%s", "version" : "%s"},
				{"objectId" : "%s", "version" : "%s"}
			],
			{"showContent": true}
		]`, objectId, version, objectId, previousVersion)

	return suiApiRequest[SuiTryMultiGetPastObjectsResponse](s.rpc, method, params)
}
