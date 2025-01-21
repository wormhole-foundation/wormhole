package txverifier

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Tokens
const (
	EthereumUsdcAddress = "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	SuiUsdcAddress      = "5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf"
)

// func initGlobals() {
// 	suiEventType = fmt.Sprintf("%s::%s::%s", *suiCoreContract, suiModule, suiEventName)
// }

func newTestSuiTransferVerifier() *SuiTransferVerifier {
	suiCoreContract := "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a"
	suiTokenBridgeContract := "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d"
	suiTokenBridgeEmitter := "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5"

	return NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeContract)
}

type MockSuiApiConnection struct {
	// The events to be returned by QueryEvents
	Events           []SuiEvent
	ObjectsResponses []SuiTryMultiGetPastObjectsResponse
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

func NewMockSuiApiConnection(events []SuiEvent) *MockSuiApiConnection {
	return &MockSuiApiConnection{
		Events:           events,
		ObjectsResponses: nil,
	}
}

func (mock *MockSuiApiConnection) SetEvents(events []SuiEvent) {
	mock.Events = events
}

func (mock *MockSuiApiConnection) SetObjectsResponse(ObjectResponse SuiTryMultiGetPastObjectsResponse) {
	mock.ObjectsResponses = append(mock.ObjectsResponses, ObjectResponse)
}

func (mock *MockSuiApiConnection) QueryEvents(filter string, cursor string, limit int, descending bool) (SuiQueryEventsResponse, error) {
	return SuiQueryEventsResponse{}, nil
}

func (mock *MockSuiApiConnection) GetTransactionBlock(txDigest string) (SuiGetTransactionBlockResponse, error) {

	objectChanges := []ObjectChange{}

	// Create new nested object that unwraps some of it
	for _, objectResponse := range mock.ObjectsResponses {
		objectType, _ := objectResponse.GetObjectType()
		objectId, _ := objectResponse.GetObjectId()
		version, _ := objectResponse.GetVersion()
		previousVersion, _ := objectResponse.GetPreviousVersion()

		obj := ObjectChange{
			ObjectType:      objectType,
			ObjectId:        objectId,
			Version:         version,
			PreviousVersion: previousVersion,
		}
		objectChanges = append(objectChanges, obj)
	}

	return SuiGetTransactionBlockResponse{Result: SuiGetTransactionBlockResult{Events: mock.Events, ObjectChanges: objectChanges}}, nil
}
func (mock *MockSuiApiConnection) TryMultiGetPastObjects(objectId string, version string, previousVersion string) (SuiTryMultiGetPastObjectsResponse, error) {

	for _, response := range mock.ObjectsResponses {
		keyIn := fmt.Sprintf("%s-%s-%s", objectId, version, previousVersion)
		objectId, err0 := response.GetObjectId()
		version, err1 := response.GetVersion()
		previousVersion, err2 := response.GetPreviousVersion()
		if err0 != nil || err1 != nil || err2 != nil {
			return SuiTryMultiGetPastObjectsResponse{}, fmt.Errorf("Error processing version data")
		}

		keyCur := fmt.Sprintf("%s-%s-%s", objectId, version, previousVersion)
		if keyIn == keyCur {
			return response, nil
		}
	}

	return SuiTryMultiGetPastObjectsResponse{}, fmt.Errorf("Can't find entry")
}

func TestNewSuiApiConnection(t *testing.T) {
	sampleUrl := "http://localhost:8080"

	api := NewSuiApiConnection(sampleUrl)
	if rpc, ok := api.(*SuiApiConnection); ok {
		assert.Equal(t, sampleUrl, rpc.rpc)
	} else {
		t.Errorf("Unable to get RPC from SuiApiConnection")
	}
}

func TestProcessEvents(t *testing.T) {
	suiTxVerifier := newTestSuiTransferVerifier()

	arbitraryEventType := "arbitrary::EventType"
	arbitraryEmitter := "0x3117"

	logger := zap.NewNop()

	// Constants used throughout the tests
	suiEventType := suiTxVerifier.suiEventType
	suiTokenBridgeEmitter := suiTxVerifier.suiTokenBridgeEmitter

	// Define test cases
	tests := []struct {
		name           string
		events         []SuiEvent
		expectedResult map[string]*big.Int
		expectedCount  uint
	}{
		{
			name:           "TestNoEvents",
			events:         []SuiEvent{},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
		{
			name: "TestSingleEthereumUSDCEvent",
			events: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, 2),
					},
				},
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
			},
			expectedCount: 1,
		},
		{
			name: "TestMultipleEthereumUSDCEvents",
			events: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(200),
			},
			expectedCount: 2,
		},
		{
			name: "TestMixedEthereumAndSuiUSDCEvents",
			events: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
				fmt.Sprintf(KEY_FORMAT, SuiUsdcAddress, vaa.ChainIDSui):           big.NewInt(100),
			},
			expectedCount: 2,
		},
		{
			name: "TestIncorrectSender",
			events: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &arbitraryEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
		{
			name: "TestSkipNonWormholeEvents",
			events: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{
					Type: &arbitraryEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedResult: map[string]*big.Int{
				fmt.Sprintf(KEY_FORMAT, EthereumUsdcAddress, vaa.ChainIDEthereum): big.NewInt(100),
			},
			expectedCount: 1,
		},
		{
			name: "TestInvalidWormholePayloads",
			events: []SuiEvent{
				{ // Invalid payload type
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(0, big.NewInt(100), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{ // Empty payload
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: []byte{},
					},
				},
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, count := suiTxVerifier.processEvents(tt.events, logger)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestProcessObjectUpdates(t *testing.T) {
	suiTxVerifier := newTestSuiTransferVerifier()

	logger := zap.NewNop() // zap.Must(zap.NewDevelopment())

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

	oneToken := new(big.Int)
	oneToken.SetString("1000000000000000000", 10)

	// Decimals, token chain, token address, wrapped or not, balance/custody
	tests := []struct {
		name           string
		objectChanges  []ObjectChange
		resultList     []ResultTestCase
		expectedResult map[string]*big.Int
		expectedCount  uint
	}{
		{
			name: "TestProcessObjectNativeBase",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(990)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectForeignBase",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "10",
					oldBalance:   "1000",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": big.NewInt(990)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectNativeNegative",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "10",
					oldBalance:   "1000",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(-990)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectForeignNegative", // Unsure if this test case is possible from Sui API
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": big.NewInt(-990)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectNativeMultiple",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0xb779486cfd6c19e9218cc7dc17c453014d2d9ba12d2ee4dbb0ec4e1e02ae1cca::spt::SPT>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0xb779486cfd6c19e9218cc7dc17c453014d2d9ba12d2ee4dbb0ec4e1e02ae1cca::spt::SPT>>",
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        "0x0063d37cdce648a7c6f72f69a75a114fbcc81ef23300e4ace60c7941521163db",
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: "80,117,89,76,1,212,111,59,203,196,167,239,20,98,5,130,115,190,206,119,147,238,189,4,100,150,53,151,201,253,9,53",
					wrapped:      false,
					newBalance:   "5000",
					oldBalance:   "50",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(990), "5075594c01d46f3bcbc4a7ef1462058273bece7793eebd0464963597c9fd0935-21": big.NewInt(4950)},
			expectedCount:  2,
		},
		{
			name: "TestProcessObjectNativeAndForeign",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "50",
					oldBalance:   "5000",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(990), "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": big.NewInt(4950)},
			expectedCount:  2,
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
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{},
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
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{},
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
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{},
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
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
		{
			name: "TestProcessObjectOneGoodOneBad",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
				{
					ObjectType:      "0x2::dynamic_field::Field<0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>, 0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::not_native_asset::NativeAsset<0x2::sui::SUI>",
					Version:         fmt.Sprintf("%s111", normalVersion),
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "10",
					oldBalance:   "1000",
					decimals:     8,
				},
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48-2": big.NewInt(990)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectRealNumbers",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000000000000000",
					oldBalance:   "999999000000000",
					decimals:     8,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(1000000000)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectNormalize",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "101000000000000000000",
					oldBalance:   "100000000000000000000",
					decimals:     18,
				},
			},
			expectedResult: map[string]*big.Int{"9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3-21": big.NewInt(100000000)},
			expectedCount:  1,
		},
		{
			name: "TestProcessObjectMissingVersion",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
					drop:         true,
				},
			},
			expectedResult: map[string]*big.Int{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection := NewMockSuiApiConnection([]SuiEvent{})

			assert.Equal(t, len(tt.objectChanges), len(tt.resultList))

			// Add all changes to the mock Sui API for future lookups
			for index := 0; index < len(tt.objectChanges); index++ {
				change := tt.objectChanges[index]
				queryResult := tt.resultList[index]

				if !queryResult.drop {
					responseObject := generateResponsesObject(change.ObjectId, change.Version, change.ObjectType, change.PreviousVersion, queryResult.newBalance, queryResult.oldBalance, queryResult.tokenAddress, queryResult.tokenChain, queryResult.decimals, queryResult.wrapped)
					connection.SetObjectsResponse(responseObject)
				}
			}

			// Run function and check results
			transferredIntoBridge, numEventsProcessed := suiTxVerifier.processObjectUpdates(tt.objectChanges, connection, logger)
			assert.Equal(t, tt.expectedResult, transferredIntoBridge)
			assert.Equal(t, tt.expectedCount, numEventsProcessed)
		})
	}
}

// TODO
func TestProcessDigest(t *testing.T) {
	suiTxVerifier := newTestSuiTransferVerifier()

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

	suiEventType := suiTxVerifier.suiEventType
	suiTokenBridgeEmitter := suiTxVerifier.suiTokenBridgeEmitter

	logger := zap.Must(zap.NewDevelopment())

	// func processDigest(digest string, suiApiConnection SuiApiInterface, logger *zap.Logger) error {
	// Needs BOTH events and ObjectChange information to be updated
	tests := []struct {
		name          string
		objectChanges []ObjectChange
		resultList    []ResultTestCase
		suiEvents     []SuiEvent
		expectedError string
		expectedCount uint
	}{
		{
			name: "TestProcessDigestNativeBase",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "1000",
					oldBalance:   "10",
					decimals:     8,
				},
			},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(990), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedError: "",
			expectedCount: 1,
		},
		{
			name: "TestProcessDigestTakingMoreThanPuttingIn",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectNativeType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectNativeId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdNative,
					tokenAddress: normalTokenAddressNative,
					wrapped:      false,
					newBalance:   "100000",
					oldBalance:   "100000",
					decimals:     8,
				},
			},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedError: "requested amount out is larger than amount in",
			expectedCount: 0,
		},
		{
			name:          "TestProcessDigestNoEvents",
			objectChanges: []ObjectChange{},
			resultList:    []ResultTestCase{},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedError: "transfer-out request for tokens that were never deposited",
			expectedCount: 0,
		},
		{
			name: "TestProcessDigestForeignBase",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "10",
					oldBalance:   "1000",
					decimals:     8,
				},
			},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
			},
			expectedError: "",
			expectedCount: 1,
		},
		{
			name:          "TestProcessDigestNoEvents",
			objectChanges: []ObjectChange{},
			resultList:    []ResultTestCase{},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(100000), SuiUsdcAddress, uint16(vaa.ChainIDSui)),
					},
				},
			},
			expectedError: "transfer-out request for tokens that were never deposited",
			expectedCount: 0,
		},
		{
			name: "TestProcessDigestMultipleEvents",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "10",
					oldBalance:   "2000",
					decimals:     8,
				},
			},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(1000), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
			},
			expectedError: "",
			expectedCount: 2,
		},
		{
			name: "TestProcessDigestMultipleEventsOverWithdraw",
			objectChanges: []ObjectChange{
				{
					ObjectType:      normalObjectForeignType,
					Version:         normalVersion,
					PreviousVersion: normalPreviousVersion,
					ObjectId:        normalObjectForeignId,
				},
			},
			resultList: []ResultTestCase{
				{
					tokenChain:   normalChainIdForeign,
					tokenAddress: normalTokenAddressForeign,
					wrapped:      true,
					newBalance:   "10",
					oldBalance:   "2000",
					decimals:     8,
				},
			},
			suiEvents: []SuiEvent{
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(990), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
				{
					Type: &suiEventType,
					Message: &WormholeMessage{
						Sender:  &suiTokenBridgeEmitter,
						Payload: generatePayload(1, big.NewInt(1001), EthereumUsdcAddress, uint16(vaa.ChainIDEthereum)),
					},
				},
			},
			expectedError: "requested amount out is larger than amount in",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assert.Equal(t, len(tt.objectChanges), len(tt.resultList))
			connection := NewMockSuiApiConnection(tt.suiEvents) // Set events for connection

			// Add Object Response data for Sui connections
			for index := 0; index < len(tt.objectChanges); index++ {
				change := tt.objectChanges[index]
				queryResult := tt.resultList[index]

				responseObject := generateResponsesObject(change.ObjectId, change.Version, change.ObjectType, change.PreviousVersion, queryResult.newBalance, queryResult.oldBalance, queryResult.tokenAddress, queryResult.tokenChain, queryResult.decimals, queryResult.wrapped)

				connection.SetObjectsResponse(responseObject)
			}

			numProcessed, err := suiTxVerifier.ProcessDigest("HASH", connection, logger)

			assert.Equal(t, true, tt.expectedError == "" && err == nil || err != nil && err.Error() == tt.expectedError)
			assert.Equal(t, tt.expectedCount, numProcessed)
		})
	}
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

/*
JSON data

Decimals, token chain, token address, wrapped or not, balance/custody
*/

func generateResponsesObject(objectId string, version string, objectType string, previousVersion string, balanceAfter string, balanceBefore string, tokenAddress string, tokenChain string, decimals uint8, isWrapped bool) SuiTryMultiGetPastObjectsResponse {

	var newVersion string
	var oldVersion string

	if isWrapped == false {
		newVersion = generateResponseObjectNative(objectId, version, objectType, balanceAfter, tokenAddress, decimals)
		oldVersion = generateResponseObjectNative(objectId, previousVersion, objectType, balanceBefore, tokenAddress, decimals)
	} else {
		newVersion = generateResponseObjectForeign(objectId, version, objectType, balanceAfter, tokenAddress, tokenChain, decimals)
		oldVersion = generateResponseObjectForeign(objectId, previousVersion, objectType, balanceBefore, tokenAddress, tokenChain, decimals)
	}

	// Complete the rest of the response data
	responseString := fmt.Sprintf(`{"result": [{"details" : %s}, {"details" : %s}]}`, newVersion, oldVersion)

	data := SuiTryMultiGetPastObjectsResponse{}
	err := json.Unmarshal([]byte(responseString), &data)
	if err != nil {
		fmt.Println("Error in JSON parsing...")
	}

	return data
}

func generateResponseObjectNative(objectId string, version string, objectType string, balance string, tokenAddress string, decimals uint8) string {
	json_string_per_object := fmt.Sprintf(`{
		"objectId": "%s",
		"version": "%s",
		"digest": "4ne8fjG16hAXP8GxuXzoA5hBwuHz6C4D7cyf4TZza4Pa",
		"type": "%s",
		"owner": {
			"ObjectOwner": "0x334881831bd89287554a6121087e498fa023ce52c037001b53a4563a00a281a5"
		},
		"previousTransaction": "FRx1iHA3Wq2ybDe3hhMSkS5yqsKJ4wUDUWY3Xp8K6g18",
		"storageRebate": "3146400",
		"content": {
			"type": "%s",
			"fields": {
			"id": {
				"id": "0x831c45a8d512c9cf46e7a8a947f7cbbb5e0a59829aa72450ff26fb1873fd0e94"
			},
			"name": {
				"type": "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x2::sui::SUI>",
				"fields": {
				"dummy_field": false
				}
			},
			"value": {
				"type": "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::native_asset::NativeAsset<0x2::sui::SUI>",
				"fields": {
				"custody": "%s",
				"decimals": %d,
				"token_address": {
					"type": "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::external_address::ExternalAddress",
					"fields": {
					"value": {
						"type": "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::bytes32::Bytes32",
						"fields": {
						"data": [
							%s
						]
						}
					}
					}
				}
				}
			}
    }}}`, objectId, version, objectType, objectType, balance, decimals, tokenAddress)

	return json_string_per_object
}

func generateResponseObjectForeign(objectId string, version string, objectType string, balance string, tokenAddress string, tokenChain string, decimals uint8) string {
	json_string_per_object := fmt.Sprintf(`{
		"objectId": "%s",
		"version": "%s",
		"digest": "CWXv7KJrNawMqREtVYCRT9PVF2H8cogW1WCLMd5iQchr",
		"type": "%s",
		"owner": {
		  "ObjectOwner": "0x334881831bd89287554a6121087e498fa023ce52c037001b53a4563a00a281a5"
		},
		"previousTransaction": "EaqLzHQTeiPq2FjYCRobDH5E91DAVZgKgZzwQUJ5FaNU",
		"storageRebate": "4050800",
		"content": {
		  "dataType": "moveObject",
		  "type": "%s",
		  "hasPublicTransfer": false,
		  "fields": {
			"id": {
			  "id": "0xf8f80c0d569fb076adb5fdc3a717dcb9ac14f7fd7512dc17efbf0f80a8b7fa8a"
			},
			"name": {
			  "type": "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::token_registry::Key<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>",
			  "fields": {
				"dummy_field": false
			  }
			},
			"value": {
			  "type": "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::wrapped_asset::WrappedAsset<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>",
			  "fields": {
				"decimals": 6,
				"info": {
				  "type": "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d::wrapped_asset::ForeignInfo<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>",
				  "fields": {
					"native_decimals": %d,
					"symbol": "USDC",
					"token_address": {
					  "type": "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::external_address::ExternalAddress",
					  "fields": {
						"value": {
						  "type": "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::bytes32::Bytes32",
						  "fields": {"data": [%s]
						  }
						}
					  }
					},
					"token_chain": %s
				  }
				},
				"treasury_cap": {
				  "type": "0x2::coin::TreasuryCap<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>",
				  "fields": {
					"id": {
					  "id": "0xa5085139fdeae133cf6ca58f1f1cee138f24ad6fc54d8e24a519dc24f3b2b974"
					},
					"total_supply": {
					  "type": "0x2::balance::Supply<0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf::coin::COIN>",
					  "fields": {
						"value": "%s"
					  }
					}
				  }
				},
				"upgrade_cap": {
				  "type": "0x2::package::UpgradeCap",
				  "fields": {
					"id": {
					  "id": "0x86ebd31cc715928671ac05e29e85b68ae1d96db02565b5413084fcb5afb695b1"
					},
					"package": "0x5d4b302506645c37ff133b98c4b50a5ae14841659738d6d733d59d0d217a93bf",
					"policy": 0,
					"version": "1"
				  }
				}
			  }
			}
		  }
		}
	  }`, objectId, version, objectType, objectType, decimals, tokenAddress, tokenChain, balance)
	return json_string_per_object

}
