package txverifier

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/suiclient"
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
	// gRPC client used to fetch transactions and objects.
	client suiclient.SuiClient
}

func NewSuiTransferVerifier(suiCoreBridgePackageId, suiTokenBridgeEmitter, suiTokenBridgePackageId string, client suiclient.SuiClient) *SuiTransferVerifier {
	return &SuiTransferVerifier{
		suiCoreBridgePackageId:  suiCoreBridgePackageId,
		suiTokenBridgeEmitter:   suiTokenBridgeEmitter,
		suiTokenBridgePackageId: suiTokenBridgePackageId,
		suiEventType:            fmt.Sprintf("%s::%s::%s", suiCoreBridgePackageId, suiModule, suiEventName),
		client:                  client,
	}
}

func (s *SuiTransferVerifier) GetTokenBridgeEmitter() string {
	return s.suiTokenBridgeEmitter
}

// GetEventType returns the fully-qualified `WormholeMessage` event type emitted by the core
// bridge. It can be used to subscribe to the relevant events over gRPC.
func (s *SuiTransferVerifier) GetEventType() string {
	return s.suiEventType
}

// extractBridgeRequestsFromEvents iterates through all events, and tries to identify `WormholeMessage` events emitted by the token bridge.
// These events are parsed and collected in a `MsgIdToRequestOutOfBridge` object, mapping message IDs to requests out of the bridge. This
// function does not return errors, as any issues encountered during processing of individual events result in those events being skipped.
func (s *SuiTransferVerifier) extractBridgeRequestsFromEvents(events []suiclient.SuiEvent, logger *zap.Logger) MsgIdToRequestOutOfBridge {
	requests := make(MsgIdToRequestOutOfBridge)

	for _, event := range events {
		// Only process `WormholeMessage` events emitted by the core bridge.
		if event.EventType != s.suiEventType {
			continue
		}

		// BCS-decode the event contents into a WormholeMessage. This is done explicitly to avoid any
		// unnecessary error logging for events that can't be deserialized into `WormholeMessage`
		// instances. If the contents cannot be decoded, the event is simply skipped.
		wormholeMessage, err := suiclient.DecodeBcs[WormholeMessage](event.BcsBytes)
		if err != nil {
			// We expect that all events of the correct type can be decoded successfully so error loudly
			logger.Error("Failed to BCS-decode WormholeMessage event", zap.Error(err))
			continue
		}

		// The on-chain sender is a 32-byte address. The message ID format and the configured
		// token bridge emitter both use a hex encoding with a "0x" prefix.
		sender := "0x" + hex.EncodeToString(wormholeMessage.Sender[:])

		// Only process the event if it was emitted by the token bridge emitter.
		if sender != s.suiTokenBridgeEmitter {
			logger.Debug("Event does not match the criteria",
				zap.String("event type", event.EventType),
				zap.String("event sender", sender),
				zap.String("expected event type", s.suiEventType),
				zap.String("expected event sender", s.suiTokenBridgeEmitter),
			)
			continue
		}

		// Parse the wormhole message. vaa.IsTransfer can be omitted, since this is done
		// inside `DecodeTransferPayloadHdr` already.
		hdr, err := vaa.DecodeTransferPayloadHdr(wormholeMessage.Payload)

		// If there is an error decoding the payload, skip the event. One reason for a potential
		// failure in decoding is that an attestation of a token was requested.
		if err != nil {
			continue
		}

		// The sender address is prefixed with "0x", but the message ID format does not include that prefix.
		senderWithout0x := strings.TrimPrefix(sender, "0x")

		msgIDStr := fmt.Sprintf("%d/%s/%d", vaa.ChainIDSui, senderWithout0x, wormholeMessage.Sequence)
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
	}

	return requests
}

// extractTransfersIntoBridgeFromObjectChanges iterates through all object changes, and tries to identify token transfers into the bridge.
// These transfers are accumulated in an `AssetKeyToTransferIntoBridge` object, which is returned to the caller. The default behaviour
// of this function is to fail-close, meaning that any errors that occur during processing result in the offending object change being ignored.
func (s *SuiTransferVerifier) extractTransfersIntoBridgeFromObjectChanges(ctx context.Context, objectChanges []suiclient.SuiObjectChange, logger *zap.Logger) AssetKeyToTransferIntoBridge {
	transfers := make(AssetKeyToTransferIntoBridge)

	for _, objectChange := range objectChanges {
		// All of these fields are required to look up and decode the object at both versions.
		if objectChange.ObjectID == nil || objectChange.ObjectType == nil || objectChange.InputVersion == nil || objectChange.OutputVersion == nil {
			continue
		}

		objectType := *objectChange.ObjectType

		// Check that the type information is correct. Doing it here means it's not necessary to do it
		// again after decoding the object contents.
		if !validateSuiAssetType(objectType, s.suiTokenBridgePackageId) {
			continue
		}

		// Fetch the object at the version after this transaction executed (the current version)
		// and at the version before it executed (the previous version). These calls go to the Sui API.
		currentObject, err := s.client.GetObjectAtVersion(ctx, *objectChange.ObjectID, objectChange.OutputVersion, []string{suiclient.ObjectFieldContents})
		if err != nil {
			logger.Error("Error getting current object version",
				zap.String("objectId", *objectChange.ObjectID),
				zap.Uint64("version", *objectChange.OutputVersion),
				zap.Error(err))
			continue
		}

		previousObject, err := s.client.GetObjectAtVersion(ctx, *objectChange.ObjectID, objectChange.InputVersion, []string{suiclient.ObjectFieldContents})
		if err != nil {
			logger.Error("Error getting previous object version",
				zap.String("objectId", *objectChange.ObjectID),
				zap.Uint64("version", *objectChange.InputVersion),
				zap.Error(err))
			continue
		}

		currentInfo, err := decodeSuiAssetObject(objectType, currentObject.ContentsBytes)
		if err != nil {
			logger.Error("Error decoding current asset object", zap.String("objectId", *objectChange.ObjectID), zap.Error(err))
			continue
		}

		previousInfo, err := decodeSuiAssetObject(objectType, previousObject.ContentsBytes)
		if err != nil {
			logger.Error("Error decoding previous asset object", zap.String("objectId", *objectChange.ObjectID), zap.Error(err))
			continue
		}

		// The decimals, token address, and token chain are immutable properties of an asset and
		// must not change across versions. A mismatch indicates malformed or unexpected data.
		if currentInfo.decimals != previousInfo.decimals {
			logger.Error("decimals do not match between object versions", zap.String("objectId", *objectChange.ObjectID))
			continue
		}
		if currentInfo.tokenAddress != previousInfo.tokenAddress {
			logger.Error("token addresses do not match between object versions", zap.String("objectId", *objectChange.ObjectID))
			continue
		}
		if currentInfo.tokenChain != previousInfo.tokenChain {
			logger.Error("token chains do not match between object versions", zap.String("objectId", *objectChange.ObjectID))
			continue
		}

		// Compute the change in balance between the previous and current versions. For wrapped
		// assets the supply is burned (decreases) when tokens are sent into the bridge, so the
		// sign is inverted to represent the deposit as a positive amount.
		balanceChange := new(big.Int).Sub(currentInfo.balance, previousInfo.balance)
		if currentInfo.isWrapped {
			balanceChange.Neg(balanceChange)
		}

		normalized := normalize(balanceChange, currentInfo.decimals)

		// Add the key if it does not exist yet
		assetKey := fmt.Sprintf(KEY_FORMAT, currentInfo.tokenAddress, currentInfo.tokenChain)

		if _, exists := transfers[assetKey]; !exists {
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

	// Get the transaction, including its events and the objects changed by its effects.
	txn, err := s.client.GetTransaction(ctx, digest, []string{
		suiclient.TransactionFieldEvents,
		suiclient.TransactionFieldChangedObjects,
	})

	if err != nil {
		logger.Error("failed to retrieve transaction",
			zap.String("txDigest", digest),
			zap.Error(err),
		)
		return false, ErrFailedToRetrieveTxBlock
	}

	// Extract bridge requests from events
	bridgeOutRequests := s.extractBridgeRequestsFromEvents(txn.Events, logger)

	if len(bridgeOutRequests) == 0 {
		logger.Debug("No relevant events found in transaction block", zap.String("txDigest", digest))
		// No valid events were identified, so the digest does not require further processing.
		return true, nil
	}

	// Process all object changes, specifically looking for transfers into the token bridge
	transfersIntoBridge := s.extractTransfersIntoBridgeFromObjectChanges(ctx, txn.ObjectChanges, logger)

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
