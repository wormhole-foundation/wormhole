package txverifier

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	mystenbcs "github.com/block-vision/sui-go-sdk/mystenbcs"
	"github.com/certusone/wormhole/node/pkg/suiclient"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Tokens
const (
	EthereumUsdcAddress = "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	SuiUsdcAddress      = "5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf"
)

func newTestSuiTransferVerifier(client suiclient.SuiClient) *SuiTransferVerifier {
	suiCoreContract := "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a"
	suiTokenBridgeContract := "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d"
	suiTokenBridgeEmitter := "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5"

	return NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeContract, client)
}

// ObjectChange is a test-local representation of a changed object. It keeps the string fields
// used by the test tables; it is converted into a suiclient.SuiObjectChange when exercising the
// verifier, and used to register the corresponding BCS-encoded object versions on the mock.
type ObjectChange struct {
	ObjectType      string
	ObjectId        string
	Version         string
	PreviousVersion string
}

type ResultTestCase struct {
	decimals     uint8
	tokenChain   string
	tokenAddress string
	wrapped      bool
	newBalance   string
	oldBalance   string
	drop         bool
}

// mockSuiClient is a suiclient.SuiClient that serves a single prepared transaction and a set of
// BCS-encoded object versions keyed by (objectId, version). Only the methods exercised by the
// verifier are meaningfully implemented.
type mockSuiClient struct {
	transaction suiclient.SuiTransaction
	objects     map[string][]byte
}

func newMockSuiClient() *mockSuiClient {
	return &mockSuiClient{objects: make(map[string][]byte)}
}

func objKey(objectId string, version uint64) string {
	return fmt.Sprintf("%s@%d", objectId, version)
}

func (m *mockSuiClient) GetTransaction(ctx context.Context, digest string, fields []string) (suiclient.SuiTransaction, error) {
	return m.transaction, nil
}

func (m *mockSuiClient) GetObjectAtVersion(ctx context.Context, objectID string, version *uint64, fields []string) (suiclient.SuiObject, error) {
	if version == nil {
		return suiclient.SuiObject{}, fmt.Errorf("nil version for object %s", objectID)
	}
	contents, ok := m.objects[objKey(objectID, *version)]
	if !ok {
		return suiclient.SuiObject{}, fmt.Errorf("object %s@%d not found", objectID, *version)
	}
	return suiclient.SuiObject{ContentsBytes: contents}, nil
}

func (m *mockSuiClient) GetObject(ctx context.Context, objectID string, fields []string) (suiclient.SuiObject, error) {
	return m.GetObjectAtVersion(ctx, objectID, nil, fields)
}

func (m *mockSuiClient) GetLatestCheckpoint(ctx context.Context, fields []string) (suiclient.SuiCheckpoint, error) {
	return suiclient.SuiCheckpoint{}, nil
}

func (m *mockSuiClient) SubscribeToTransactionEvent(ctx context.Context, eventType string, eventWriteChannel chan<- suiclient.SuiTransactionEvent) (suiclient.SuiSubscription, error) {
	return suiclient.SuiSubscription{}, nil
}

func (m *mockSuiClient) SubscribeToTransactionEvents(ctx context.Context, eventTypes []string, eventWriteChannel chan<- suiclient.SuiTransactionEvent) (suiclient.SuiSubscription, error) {
	return suiclient.SuiSubscription{}, nil
}

func (m *mockSuiClient) Close() error {
	return nil
}

// registerObject registers the BCS-encoded contents for both the current (output) and previous
// (input) versions of an object change, unless the result case is marked to be dropped (in which
// case the object is left unregistered so lookups fail, mirroring a missing object).
func (m *mockSuiClient) registerObject(change ObjectChange, result ResultTestCase) {
	if result.drop {
		return
	}

	tokenAddress := parseByteCSV(result.tokenAddress)

	var currentContents, previousContents []byte
	if result.wrapped {
		tokenChain := vaa.ChainID(mustU16(result.tokenChain))
		currentContents = bcsWrappedObject(mustU64(result.newBalance), tokenAddress, tokenChain, result.decimals)
		previousContents = bcsWrappedObject(mustU64(result.oldBalance), tokenAddress, tokenChain, result.decimals)
	} else {
		currentContents = bcsNativeObject(mustU64(result.newBalance), tokenAddress, result.decimals)
		previousContents = bcsNativeObject(mustU64(result.oldBalance), tokenAddress, result.decimals)
	}

	m.objects[objKey(change.ObjectId, mustU64(change.Version))] = currentContents
	m.objects[objKey(change.ObjectId, mustU64(change.PreviousVersion))] = previousContents
}

// --- test helpers ---

func mustU64(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid uint64 %q: %v", s, err))
	}
	return v
}

func mustU16(s string) uint16 {
	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		panic(fmt.Sprintf("invalid uint16 %q: %v", s, err))
	}
	return uint16(v)
}

// parseByteCSV parses a comma-separated list of decimal byte values (e.g. "0,1,255") into a []byte.
func parseByteCSV(s string) []byte {
	parts := strings.Split(s, ",")
	out := make([]byte, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseUint(strings.TrimSpace(p), 10, 8)
		if err != nil {
			panic(fmt.Sprintf("invalid byte %q: %v", p, err))
		}
		out[i] = byte(v)
	}
	return out
}

// senderBytes converts a hex sender/emitter string (with or without a 0x prefix) into the 32-byte
// representation used by the on-chain WormholeMessage event.
func senderBytes(sender string) [32]byte {
	raw, err := hex.DecodeString(strings.TrimPrefix(sender, "0x"))
	if err != nil {
		panic(fmt.Sprintf("invalid sender %q: %v", sender, err))
	}
	var out [32]byte
	copy(out[:], raw)
	return out
}

func bcsNativeObject(balance uint64, tokenAddress []byte, decimals uint8) []byte {
	return mystenbcs.MustMarshal(suiNativeAssetField{
		Value: suiNativeAsset{
			Custody:      balance,
			TokenAddress: suiExternalAddress{Value: suiBytes32{Data: tokenAddress}},
			Decimals:     decimals,
		},
	})
}

func bcsWrappedObject(supply uint64, tokenAddress []byte, tokenChain vaa.ChainID, decimals uint8) []byte {
	return mystenbcs.MustMarshal(suiWrappedAssetField{
		Value: suiWrappedAsset{
			Info: suiForeignInfo{
				TokenChain:     tokenChain,
				TokenAddress:   suiExternalAddress{Value: suiBytes32{Data: tokenAddress}},
				NativeDecimals: decimals,
				Symbol:         []byte("TEST"),
			},
			TreasuryCap: suiTreasuryCap{TotalSupply: supply},
			Decimals:    decimals,
		},
	})
}

// makeWormholeEvent builds a suiclient.SuiEvent whose BCS contents are a WormholeMessage with the
// given sender, sequence and payload.
func makeWormholeEvent(eventType string, sender string, payload []byte, sequence uint64) suiclient.SuiEvent {
	return suiclient.SuiEvent{
		EventType: eventType,
		BcsBytes: mystenbcs.MustMarshal(WormholeMessage{
			Sender:   senderBytes(sender),
			Sequence: sequence,
			Payload:  payload,
		}),
	}
}

// toSuiObjectChange converts a test-local ObjectChange into a suiclient.SuiObjectChange.
func toSuiObjectChange(change ObjectChange) suiclient.SuiObjectChange {
	objectType := change.ObjectType
	objectId := change.ObjectId
	outputVersion := mustU64(change.Version)
	inputVersion := mustU64(change.PreviousVersion)
	return suiclient.SuiObjectChange{
		ObjectID:      &objectId,
		ObjectType:    &objectType,
		OutputVersion: &outputVersion,
		InputVersion:  &inputVersion,
	}
}

func toSuiObjectChanges(changes []ObjectChange) []suiclient.SuiObjectChange {
	out := make([]suiclient.SuiObjectChange, 0, len(changes))
	for _, c := range changes {
		out = append(out, toSuiObjectChange(c))
	}
	return out
}

// Generate WormholeMessage payload.
//
//	Payload type: payload[0]
//	Amount: payload[1] for 32
//	Origin address: payload[33] for 32
//	Origin chain ID: payload[65] for 2
func generatePayload(payloadType byte, amount *big.Int, originAddressHex string, originChainID uint16) []byte {
	originAddress, _ := hex.DecodeString(originAddressHex)

	payload := make([]byte, 0, 101)

	// Append payload type
	payload = append(payload, payloadType)

	// Append amount (32 bytes)
	amountBytes := amount.FillBytes(make([]byte, 32))
	payload = append(payload, amountBytes...)

	// Append origin address (32 bytes)
	payload = append(payload, originAddress...)

	// Append origin chain ID (2 bytes)
	originChainIDBytes := []byte{byte(originChainID >> 8), byte(originChainID & 0xff)}
	payload = append(payload, originChainIDBytes...)

	// Right-pad the payload to 101 bytes
	padding := make([]byte, 101-len(payload))
	payload = append(payload, padding...)

	return payload
}

func TestProcessEvents(t *testing.T) {
	suiTxVerifier := newTestSuiTransferVerifier(nil)

	arbitraryEventType := "arbitrary::EventType"
	arbitraryEmitter := "0x3117"

	logger := zap.NewNop()

	// Constants used throughout the tests
	suiEventType := suiTxVerifier.suiEventType
	suiTokenBridgeEmitter := suiTxVerifier.suiTokenBridgeEmitter

	// Define test cases. Each event is given a unique sequence number so that distinct events
	// produce distinct message IDs (and therefore distinct bridge-out requests).
	tests := []struct {
		name           string
		events         []suiclient.SuiEvent
		expectedResult map[string]*big.Int
		expectedCount  int
	}{
		{
			name:           "TestNoEvents",
			events:         []suiclient.SuiEvent{},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
		{
			name: "TestSingleEthereumUSDCEvent",
			events: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, 2), 1),
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
			},
			expectedCount: 1,
		},
		{
			name: "TestMultipleEthereumUSDCEvents",
			events: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 2),
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 3),
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(200),
			},
			expectedCount: 2,
		},
		{
			name: "TestMixedEthereumAndSuiUSDCEvents",
			events: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 4),
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 5),
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
				fmt.Sprintf(KEY_FORMAT, SuiUsdcAddress, vaa.ChainIDSui):           big.NewInt(100),
			},
			expectedCount: 2,
		},
		{
			name: "TestIncorrectSender",
			events: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, arbitraryEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 6),
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
		{
			name: "TestSkipNonWormholeEvents",
			events: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 7),
				makeWormholeEvent(arbitraryEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 8),
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
			},
			expectedCount: 1,
		},
		{
			name: "TestInvalidWormholePayloads",
			events: []suiclient.SuiEvent{
				// Invalid payload type
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(0, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 9),
				// Empty payload
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, []byte{}, 10),
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := suiTxVerifier.extractBridgeRequestsFromEvents(tt.events, logger)
			assert.Equal(t, tt.expectedCount, len(requests))
		})
	}
}

func TestProcessObjectUpdates(t *testing.T) {
	ctx := context.TODO()
	logger := zap.NewNop()

	// Constants used throughout the tests
	normalObjectNativeType := "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x2::sui::SUI>>"
	normalObjectForeignType := "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::wrapped_asset::WrappedAsset<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>>"
	normalVersion := "6565"
	normalPreviousVersion := "4040"
	normalObjectNativeId := "0x831c45a8d512c9cf46e7a8a947f7cbbb5e0a59829aa72450ff26fb1873fd0e94"
	normalObjectForeignId := "0xf8f80c0d569fb076adb5fdc3a717dcb9ac14f7fd7512dc17efbf0f80a8b7fa8a"

	normalTokenAddressForeign := "0,0,0,0,0,0,0,0,0,0,0,0,160,184,105,145,198,33,139,54,193,209,157,74,46,158,176,206,54,6,235,72"
	normalTokenAddressNative := "146,88,24,31,92,234,200,219,255,183,3,8,144,36,60,174,214,154,149,153,210,136,109,149,122,156,183,101,106,243,189,179"
	normalChainIdNative := "21"
	normalChainIdForeign := "2"

	// Decimals, token chain, token address, wrapped or not, balance/custody
	tests := []struct {
		name           string
		objectChanges  []ObjectChange
		resultList     []ResultTestCase
		expectedResult map[string]TransferIntoBridge
		expectedCount  uint
	}{
		{
			name: "TestProcessObjectNativeBase",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(990)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectForeignBase",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "10", oldBalance: "1000", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": {Amount: big.NewInt(990)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectNativeNegative",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "10", oldBalance: "1000", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(-990)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectForeignNegative", // Unsure if this test case is possible from Sui API
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": {Amount: big.NewInt(-990)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectNativeMultiple",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0xb779486cfd6c19e9218cc7dc17c453014d2d9ba12d2ee4dbb0ec4e1e02ae1cca::spt::SPT>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0xb779486cfd6c19e9218cc7dc17c453014d2d9ba12d2ee4dbb0ec4e1e02ae1cca::spt::SPT>>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        "0x0063d37cdce648a7c6f72f69a75a114fbcc81ef23300e4ace60c7941521163db",
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
				{tokenChain: normalChainIdNative, tokenAddress: "80,117,89,76,1,212,111,59,203,196,167,239,20,98,5,130,115,190,206,119,147,238,189,4,100,150,53,151,201,253,9,53", wrapped: false, newBalance: "5000", oldBalance: "50", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(990)},
				"5075594c01d46f3bcbc4a7ef1462058273bece7793eebd0464963597c9fd0935-21": {Amount: big.NewInt(4950)},
			},
			expectedCount: 2,
		},
		{
			name: "TestProcessObjectNativeAndForeign",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "50", oldBalance: "5000", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(990)},
				"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2":  {Amount: big.NewInt(4950)},
			},
			expectedCount: 2,
		},
		{
			name: "TestProcessObjectWrongPackageIdType",
			objectChanges: []ObjectChange{
				{
					ObjectType:      "0x2::dynamic_field::Field<0xa340e3db1332c21f20f5c08bef0fa459e733575f9a7e2f5faca64f72cd5a54f2::token_registry::Key<0x2::sui::SUI>, 0xa340e3db1332c21f20f5c08bef0fa459e733575f9a7e2f5faca64f72cd5a54f2::native_asset::NativeAsset<0x2::sui::SUI>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{},
			expectedCount:  0,
		},
		{
			name: "TestProcessObjectNotDynamicField",
			objectChanges: []ObjectChange{
				{
					ObjectType:      "0x11111111111111111111::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x2::sui::SUI>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{},
			expectedCount:  0,
		},
		{
			name: "TestProcessObjectMismatchedCoinTypes",
			objectChanges: []ObjectChange{
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x11111111111111111111::sui::SUI>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{},
			expectedCount:  0,
		},
		{
			name: "TestProcessObjectNotAssetType",
			objectChanges: []ObjectChange{
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::not_native_asset::NativeAsset<0x2::sui::SUI>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{},
			expectedCount:  0,
		},
		{
			name: "TestProcessObjectOneGoodOneBad",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::not_native_asset::NativeAsset<0x2::sui::SUI>",
					Version:         fmt.Sprintf("%s111", normalVersion),
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "10", oldBalance: "1000", decimals: 8},
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": {Amount: big.NewInt(990)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectRealNumbers",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000000000000000", oldBalance: "999999000000000", decimals: 8},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(1000000000)},
			},
			expectedCount: 1,
		},
		{
			// Balances are kept within uint64 (the on-chain Balance/Supply type) while still
			// exercising the >8-decimals normalization: a change of 1e18 at 18 decimals
			// normalizes to 1e8.
			name: "TestProcessObjectNormalize",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "2000000000000000000", oldBalance: "1000000000000000000", decimals: 18},
			},
			expectedResult: map[string]TransferIntoBridge{
				"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": {Amount: big.NewInt(100000000)},
			},
			expectedCount: 1,
		},
		{
			name: "TestProcessObjectMissingVersion",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8, drop: true},
			},
			expectedResult: map[string]TransferIntoBridge{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, len(tt.objectChanges), len(tt.resultList))

			mock := newMockSuiClient()
			suiTxVerifier := newTestSuiTransferVerifier(mock)

			// Register all object versions on the mock for future lookups
			for index := range tt.objectChanges {
				mock.registerObject(tt.objectChanges[index], tt.resultList[index])
			}

			// Run function and check results
			transfers := suiTxVerifier.extractTransfersIntoBridgeFromObjectChanges(ctx, toSuiObjectChanges(tt.objectChanges), logger)

			// Check that expectedResult and transfers have same number of keys
			assert.Equal(t, uint(len(tt.expectedResult)), uint(len(transfers)))

			// Check that each key in expectedResult exists in transfers and has the expected amount
			for key, expectedValue := range tt.expectedResult {
				actualValue, exists := transfers[key]
				if !exists {
					t.Errorf("Expected key %s not found in result", key)
				} else if actualValue.Amount.Cmp(expectedValue.Amount) != 0 {
					t.Errorf("For key %s, expected amount %s but got %s", key, expectedValue.Amount.String(), actualValue.Amount.String())
				}
			}
		})
	}
}

func TestProcessDigest(t *testing.T) {
	// Constants used throughout the tests
	normalObjectNativeType := "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x2::sui::SUI>>"
	normalObjectForeignType := "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::wrapped_asset::WrappedAsset<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>>"
	normalVersion := "6565"
	normalPreviousVersion := "4040"
	normalObjectNativeId := "0x831c45a8d512c9cf46e7a8a947f7cbbb5e0a59829aa72450ff26fb1873fd0e94"
	normalObjectForeignId := "0xf8f80c0d569fb076adb5fdc3a717dcb9ac14f7fd7512dc17efbf0f80a8b7fa8a"

	normalTokenAddressForeign := "0,0,0,0,0,0,0,0,0,0,0,0,160,184,105,145,198,33,139,54,193,209,157,74,46,158,176,206,54,6,235,72"
	normalTokenAddressNative := "93,75,48,37,6,100,92,55,255,19,59,152,196,181,10,90,225,72,65,101,151,56,214,215,51,213,157,13,33,122,147,191"
	normalChainIdNative := "21"
	normalChainIdForeign := "2"

	suiTxVerifier := newTestSuiTransferVerifier(nil)
	suiEventType := suiTxVerifier.suiEventType
	suiTokenBridgeEmitter := suiTxVerifier.suiTokenBridgeEmitter

	logger := zap.Must(zap.NewDevelopment())

	// Needs BOTH events and ObjectChange information to be set on the mocked transaction.
	tests := []struct {
		name          string
		objectChanges []ObjectChange
		resultList    []ResultTestCase
		suiEvents     []suiclient.SuiEvent
		expectedError error
	}{
		{
			name: "TestProcessDigestNativeBase",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "1000", oldBalance: "10", decimals: 8},
			},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(990), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 101),
			},
			expectedError: nil,
		},
		{
			name: "TestProcessDigestTakingMoreThanPuttingIn",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectNativeType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectNativeId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdNative, tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: "100000", oldBalance: "100000", decimals: 8},
			},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 102),
			},
			expectedError: &InvariantError{Msg: INVARIANT_INSUFFICIENT_DEPOSIT},
		},
		{
			name:          "TestProcessDigestNoEventsNative",
			objectChanges: []ObjectChange{},
			resultList:    []ResultTestCase{},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 103),
			},
			expectedError: &InvariantError{Msg: INVARIANT_NO_DEPOSIT},
		},
		{
			name: "TestProcessDigestForeignBase",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "10", oldBalance: "1000", decimals: 8},
			},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 104),
			},
			expectedError: nil,
		},
		{
			name:          "TestProcessDigestNoEventsForeign",
			objectChanges: []ObjectChange{},
			resultList:    []ResultTestCase{},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)), 105),
			},
			expectedError: &InvariantError{Msg: INVARIANT_NO_DEPOSIT},
		},
		{
			name: "TestProcessDigestMultipleEvents",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "10", oldBalance: "2000", decimals: 8},
			},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 106),
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(1000), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 107),
			},
			expectedError: nil,
		},
		{
			name: "TestProcessDigestMultipleEventsOverWithdraw",
			objectChanges: []ObjectChange{
				{ObjectType: normalObjectForeignType, Version: normalVersion, PreviousVersion: normalPreviousVersion, ObjectId: normalObjectForeignId},
			},
			resultList: []ResultTestCase{
				{tokenChain: normalChainIdForeign, tokenAddress: normalTokenAddressForeign, wrapped: true, newBalance: "10", oldBalance: "2000", decimals: 8},
			},
			suiEvents: []suiclient.SuiEvent{
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 108),
				makeWormholeEvent(suiEventType, suiTokenBridgeEmitter, generatePayload(1, big.NewInt(1001), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)), 109),
			},
			expectedError: &InvariantError{Msg: INVARIANT_INSUFFICIENT_DEPOSIT},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			assert.Equal(t, len(tt.objectChanges), len(tt.resultList))

			mock := newMockSuiClient()
			mock.transaction = suiclient.SuiTransaction{
				Events:        tt.suiEvents,
				ObjectChanges: toSuiObjectChanges(tt.objectChanges),
			}

			// Register object version data on the mock
			for index := range tt.objectChanges {
				mock.registerObject(tt.objectChanges[index], tt.resultList[index])
			}

			suiTxVerifier := newTestSuiTransferVerifier(mock)

			_, err := suiTxVerifier.processDigestInternal(ctx, "HASH", "", logger)

			assert.Equal(t, true, tt.expectedError == nil && err == nil || err != nil && err.Error() == tt.expectedError.Error())
		})
	}
}

func TestSuiVerifierGetters(t *testing.T) {
	v := newTestSuiTransferVerifier(nil)
	assert.Equal(t, "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5", v.GetTokenBridgeEmitter())
	assert.Equal(t, "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage", v.GetEventType())
}

func TestDecodeSuiAssetObject(t *testing.T) {
	addr := parseByteCSV("0,0,0,0,0,0,0,0,0,0,0,0,160,184,105,145,198,33,139,54,193,209,157,74,46,158,176,206,54,6,235,72")

	// Unknown asset type -> error.
	_, err := decodeSuiAssetObject("0x2::dynamic_field::Field<x::token_registry::Key<c>, y::other::Other<c>>", bcsNativeObject(1, addr, 8))
	assert.Error(t, err)

	// Native type but undecodable (truncated) bytes -> error.
	_, err = decodeSuiAssetObject("a::native_asset::NativeAsset<c>", []byte{0x01, 0x02})
	assert.Error(t, err)

	// Wrapped asset with an unknown token chain -> KnownChainIDFromNumber error.
	_, err = decodeSuiAssetObject("a::wrapped_asset::WrappedAsset<c>", bcsWrappedObject(1, addr, 60000, 8))
	assert.Error(t, err)

	// Wrapped asset happy path.
	info, err := decodeSuiAssetObject("a::wrapped_asset::WrappedAsset<c>", bcsWrappedObject(990, addr, 2, 8))
	assert.NoError(t, err)
	assert.True(t, info.isWrapped)
	assert.Equal(t, uint64(990), info.balance.Uint64())
	assert.Equal(t, vaa.ChainIDEthereum, info.tokenChain)
}

// TestProcessDigestPublic exercises the public ProcessDigest wrapper, including its handling of
// invariant violations (which it converts to a (false, nil) result).
func TestProcessDigestPublic(t *testing.T) {
	normalObjectNativeType := "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x2::sui::SUI>>"
	normalTokenAddressNative := "93,75,48,37,6,100,92,55,255,19,59,152,196,181,10,90,225,72,65,101,151,56,214,215,51,213,157,13,33,122,147,191"
	objectId := "0x831c45a8d512c9cf46e7a8a947f7cbbb5e0a59829aa72450ff26fb1873fd0e94"
	logger := zap.NewNop()
	ctx := context.TODO()

	build := func(newBalance, oldBalance string, amount *big.Int) *SuiTransferVerifier {
		change := ObjectChange{ObjectType: normalObjectNativeType, Version: "6565", PreviousVersion: "4040", ObjectId: objectId}
		result := ResultTestCase{tokenChain: "21", tokenAddress: normalTokenAddressNative, wrapped: false, newBalance: newBalance, oldBalance: oldBalance, decimals: 8}
		mock := newMockSuiClient()
		v := newTestSuiTransferVerifier(mock)
		mock.transaction = suiclient.SuiTransaction{
			Events:        []suiclient.SuiEvent{makeWormholeEvent(v.suiEventType, v.suiTokenBridgeEmitter, generatePayload(1, amount, SuiUsdcAddress, uint16(vaa.ChainIDSui)), 1)},
			ObjectChanges: toSuiObjectChanges([]ObjectChange{change}),
		}
		mock.registerObject(change, result)
		return v
	}

	// Solvent: a deposit of 990 covers the 990 requested out of the bridge.
	ok, err := build("1000", "10", big.NewInt(990)).ProcessDigest(ctx, "HASH", "", logger)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Insolvent: nothing deposited for the 100000 requested -> ProcessDigest converts the
	// invariant violation into (false, nil).
	ok, err = build("100000", "100000", big.NewInt(100000)).ProcessDigest(ctx, "HASH", "", logger)
	assert.NoError(t, err)
	assert.False(t, ok)
}
