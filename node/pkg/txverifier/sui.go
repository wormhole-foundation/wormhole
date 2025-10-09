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

	// The Sui transfer verifier needs the original token bridge package Id to address the token registry correctly.
	// The token registry holds the balances for all assets, wrapped and native. If the token registry on chain is
	// ever moved/upgraded, these values will need to be updated.
	SuiOriginalTokenBridgePackageIds = map[common.Environment]string{
		// Obtained from the mainnet state object at 0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9
		common.MainNet: "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d",
		// Obtained from the testnet state object at 0x6fb10cdb7aa299e9a4308752dadecb049ff55a892de92992a1edbd7912b3d6da
		common.TestNet: "0x562760fc51d90d4ae1835bac3e91e0e6987d3497b06f066941d3e51f6e8d76d0",
		// Obtained from tilt output when deploying the token bridge to devnet
		common.UnsafeDevNet:   "0xa6a3da85bbe05da5bfd953708d56f1a3a023e7fb58e5a824a3d4de3791e8f690",
		common.GoTest:         "0xa6a3da85bbe05da5bfd953708d56f1a3a023e7fb58e5a824a3d4de3791e8f690",
		common.AccountantMock: "0xa6a3da85bbe05da5bfd953708d56f1a3a023e7fb58e5a824a3d4de3791e8f690",
	}
)

type SuiTransferVerifier struct {
	// Used to create the event filter.
	suiCoreBridgePackageId string
	// Used to check the emitter of the `WormholeMessage` event.
	suiTokenBridgeEmitter string
	// Used to match the owning package of native and wrapped asset types.
	suiTokenBridgePackageId string
	suiEventType            string
	suiApiConnection        SuiApiInterface
}

func NewSuiTransferVerifier(suiCoreBridgePackageId, suiTokenBridgeEmitter, suiTokenBridgePackageId string, suiApiConnection SuiApiInterface) *SuiTransferVerifier {
	return &SuiTransferVerifier{
		suiCoreBridgePackageId:  suiCoreBridgePackageId,
		suiTokenBridgeEmitter:   suiTokenBridgeEmitter,
		suiTokenBridgePackageId: suiTokenBridgePackageId,
		suiEventType:            fmt.Sprintf("%s::%s::%s", suiCoreBridgePackageId, suiModule, suiEventName),
		suiApiConnection:        suiApiConnection,
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
	}`, s.suiCoreBridgePackageId, suiModule, s.suiEventType)
}

// extractBridgeRequestsFromEvents iterates through all events, and tries to identify `WormholeMessage` events emitted by the token bridge.
// These events are parsed and collected in a `MsgIdToRequestOutOfBridge` object, mapping message IDs to requests out of the bridge. This
// function does not return errors, as any issues encountered during processing of individual events result in those events being skipped.
func (s *SuiTransferVerifier) extractBridgeRequestsFromEvents(events []SuiEvent, logger *zap.Logger) MsgIdToRequestOutOfBridge {
	requests := make(MsgIdToRequestOutOfBridge)

	for _, event := range events {
		var wormholeMessage WormholeMessage

		// Parse the ParsedJson field into a WormholeMessage. This is done explicitly to avoid any unnecessary
		// error logging for events that can't be deserialized into `WormholeMessage` instances. If an event's
		// ParsedJson cannot be unmarshaled into a WormholeMessage, it is simply skipped.
		if event.ParsedJson != nil {
			err := json.Unmarshal(*event.ParsedJson, &wormholeMessage)
			if err != nil {
				// If an error ocurrs, the ParsedJson is rejected as an event that is not emitted by the bridge
				continue
			}
		}

		// If any of these event parameters are nil, skip the event
		if wormholeMessage.Sender == nil || wormholeMessage.Sequence == nil || event.Type == nil {
			continue
		}

		// Only process the event if it is a WormholeMessage event from the token bridge emitter
		if *event.Type == s.suiEventType && *wormholeMessage.Sender == s.suiTokenBridgeEmitter {

			// Parse the wormhole message. vaa.IsTransfer can be omitted, since this is done
			// inside `DecodeTransferPayloadHdr` already.
			hdr, err := vaa.DecodeTransferPayloadHdr(wormholeMessage.Payload)

			// If there is an error decoding the payload, skip the event. One reason for a potential
			// failure in decoding is that an attestation of a token was requested.
			if err != nil {
				logger.Debug("Error decoding payload", zap.Error(err))
				continue
			}

			// The sender address is prefixed with "0x", but the message ID format does not include that prefix.
			senderWithout0x := strings.TrimPrefix(*wormholeMessage.Sender, "0x")

			msgIDStr := fmt.Sprintf("%d/%s/%s", vaa.ChainIDSui, senderWithout0x, *wormholeMessage.Sequence)
			assetKey := fmt.Sprintf(KEY_FORMAT, hdr.OriginAddress.String(), hdr.OriginChain)

			logger.Debug("Found request out of bridge",
				zap.String("msgID", msgIDStr),
				zap.String("assetKey", assetKey),
				zap.String("amount", hdr.Amount.String()),
			)

			requests[msgIDStr] = &RequestOutOfBridge{
				AssetKey:       assetKey,
				Amount:         hdr.Amount,
				DepositMade:    false,
				DepositSolvent: false,
			}
		} else {
			logger.Debug("Event does not match the criteria",
				zap.String("event type", *event.Type),
				zap.String("event sender", *wormholeMessage.Sender),
				zap.String("expected event type", s.suiEventType),
				zap.String("expected event sender", s.suiTokenBridgeEmitter),
			)
		}
	}

	return requests
}

// extractTransfersIntoBridgeFromChanges iterates through all object changes, and tries to identify token transfers into the bridge.
// These transfers are accumulated in an `AssetKeyToTransferIntoBridge` object, which is returned to the caller. The default behaviour
// of this function is to fail-close, meaning that any errors that occur during processing result in the offending object change being ignored.
func (s *SuiTransferVerifier) extractTransfersIntoBridgeFromObjectChanges(ctx context.Context, objectChanges []ObjectChange, logger *zap.Logger) AssetKeyToTransferIntoBridge {
	transfers := make(AssetKeyToTransferIntoBridge)

	for _, objectChange := range objectChanges {
		// Check that the type information is correct. Doing it here means it's not necessary to do it
		// again, even in the case where `GetObject` is used for a single object instead of getting past
		// objects.
		if !objectChange.ValidateTypeInformation(s.suiTokenBridgePackageId) {
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
		assetKey := fmt.Sprintf(KEY_FORMAT, address, chain)

		if _, exists := transfers[assetKey]; !exists {
			// logger.Debug("First time seeing transfer into bridge for asset", zap.String("assetKey", assetKey))
			transfers[assetKey] = &TransferIntoBridge{
				Amount:  big.NewInt(0),
				Solvent: false,
			}
		}

		// Add the amount transferred into the bridge
		logger.Debug("Adding transfer into bridge", zap.String("assetKey", assetKey), zap.String("amount", normalized.String()))
		transfers[assetKey].Amount = new(big.Int).Add(transfers[assetKey].Amount, normalized)
	}

	return transfers
}

func (s *SuiTransferVerifier) ProcessDigest(ctx context.Context, digest string, msgIdStr string, logger *zap.Logger) (bool, error) {
	verified, err := s.processDigestInternal(ctx, digest, msgIdStr, logger)

	// check if the error is an invariant violation
	var invariantError *InvariantError
	if errors.As(err, &invariantError) {
		logger.Error("Sui txverifier invariant violated", zap.String("txdigest", digest), zap.String("invariant", invariantError.Msg))
		return false, nil
	} else {
		return verified, err
	}
}

// Return conditions:
//
//	true, nil - verification succeeded
//	false, nil - verification failed
//	false, err - verification failed due to an internal error or invariant violation
//
// NOTE: it is up to the caller to check if the error is an invariant violation, and handle it accordingly.
func (s *SuiTransferVerifier) processDigestInternal(ctx context.Context, digest string, msgIdStr string, logger *zap.Logger) (bool, error) {
	logger.Debug("processing digest", zap.String("txDigest", digest), zap.String("msgId", msgIdStr))

	// Get the transaction block
	txBlock, err := s.suiApiConnection.GetTransactionBlock(ctx, digest)

	if err != nil {
		logger.Error("failed to retrieve transaction block",
			zap.String("txDigest", digest),
			zap.Error(err),
		)
		return false, ErrFailedToRetrieveTxBlock
	}

	// Extract bridge requests from events
	bridgeOutRequests := s.extractBridgeRequestsFromEvents(txBlock.Result.Events, logger)

	if len(bridgeOutRequests) == 0 {
		logger.Debug("No relevant events found in transaction block", zap.String("txDigest", digest))
		// No valid events were identified, so the digest does not require further processing.
		return true, nil
	}

	// Process all object changes, specifically looking for transfers into the token bridge
	transfersIntoBridge := s.extractTransfersIntoBridgeFromObjectChanges(ctx, txBlock.Result.ObjectChanges, logger)

	// Validate solvency using the requests out of the bridge vs the transfers into the bridge.
	resolved, err := validateSolvency(bridgeOutRequests, transfersIntoBridge)

	if err != nil {
		logger.Error("Error validating solvency", zap.Error(err))
		return false, err
	}

	// If msgIdStr is found in the resolved map, check only that request. Otherwise, check all requests.
	if request, exists := resolved[msgIdStr]; exists {

		// Checking for nil, since the map value is a pointer.
		if request == nil {
			logger.Debug("No matching request found for message ID", zap.String("msgId", msgIdStr))
			// No matching request was found for the given message ID.
			return false, fmt.Errorf("no matching request found for message ID %s", msgIdStr)
		}

		if !request.DepositMade {
			logger.Debug("No deposit made for request out of bridge",
				zap.String("msgId", msgIdStr),
				zap.String("assetKey", request.AssetKey),
				zap.String("amount", request.Amount.String()))
			// A deposit was not made for the given message ID.
			return false, &InvariantError{Msg: INVARIANT_NO_DEPOSIT}
		}

		if !request.DepositSolvent {
			logger.Debug("Deposit for request out of bridge was insolvent",
				zap.String("msgId", msgIdStr),
				zap.String("assetKey", request.AssetKey),
				zap.String("amount", request.Amount.String()))
			// A deposit was not solvent for the given message ID.
			return false, &InvariantError{Msg: INVARIANT_INSUFFICIENT_DEPOSIT}
		}

		logger.Debug("Request for message ID is valid", zap.String("msgId", msgIdStr))
	} else {
		// Any request that is not valid causes the entire transaction to be considered invalid.
		for msgIdStrLoc, request := range resolved {
			if !request.DepositMade {
				logger.Debug("No deposit made for request out of bridge",
					zap.String("assetKey", request.AssetKey),
					zap.String("amount", request.Amount.String()))
				// A request was not fulfilled by a deposit into the bridge.
				return false, &InvariantError{Msg: INVARIANT_NO_DEPOSIT}
			}

			if !request.DepositSolvent {
				logger.Debug("Deposit for request out of bridge was insolvent",
					zap.String("assetKey", request.AssetKey),
					zap.String("amount", request.Amount.String()))
				// A request was not solvent.
				return false, &InvariantError{Msg: INVARIANT_INSUFFICIENT_DEPOSIT}
			}

			logger.Debug("Request for message ID is valid", zap.String("msgId", msgIdStrLoc))
		}
	}

	return true, nil
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
