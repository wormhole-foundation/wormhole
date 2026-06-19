package sui

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	mystenbcs "github.com/block-vision/sui-go-sdk/mystenbcs"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/suiclient"
	"github.com/certusone/wormhole/node/pkg/testutils"
	txverifier "github.com/certusone/wormhole/node/pkg/txverifier"
)

// A `WormholeMessage` event captured from mainnet transaction
// 3dFwinvN8cxotrbHmisT1zMnFiFeR5nowo9zzZw7akin. The base64 string is the BCS-serialized
// event contents (the `contents` of the gRPC event), and the remaining constants are the
// values it decodes to, taken from the JSON-RPC `parsedJson` rendering of the same event.
const (
	sampleWormholeMessageBcsB64 = "zM7rKTSPcb3SL/70OioZwfW14XxcylQRUpEgGCZyreW/OAMAAAAAAEyGAQCFAQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACjm6FyXsyJ/MIvDRPeNoi5T6FtZKIgeRhqlBkUKAxnEB/3VCY8ABUAAAAAAAAAAAAAAABlqPB72ahZjhtbbAqI9HedvAd2dQAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMmMkaAAAAAA="
	sampleTxDigest              = "3dFwinvN8cxotrbHmisT1zMnFiFeR5nowo9zzZw7akin"
	sampleSenderHex             = "ccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5"
	samplePayloadHex            = "01000000000000000000000000000000000000000000000000000000a39ba1725ecc89fcc22f0d13de3688b94fa16d64a22079186a941914280c67101ff754263c001500000000000000000000000065a8f07bd9a8598e1b5b6c0a88f4779dbc07767500040000000000000000000000000000000000000000000000000000000000000000"
	sampleSequence              = uint64(211135)
	sampleNonce                 = uint32(99916)
	sampleConsistencyLevel      = uint8(0)
	sampleTimestamp             = uint64(1747215154)
)

// sampleWormholeMessageBcs is the BCS-serialized contents of the sample WormholeMessage event,
// decoded once from sampleWormholeMessageBcsB64.
var sampleWormholeMessageBcs = func() []byte {
	b, err := base64.StdEncoding.DecodeString(sampleWormholeMessageBcsB64)
	if err != nil {
		panic(err)
	}
	return b
}()

// mockSuiClient is a minimal suiclient.SuiClient implementation for exercising the watcher's
// gRPC-facing logic. Only the methods used by the tests return meaningful data. Object versions
// are served from `objects`, keyed by "objectId@version".
type mockSuiClient struct {
	transactions map[string]suiclient.SuiTransaction
	objects      map[string][]byte
}

func mockObjectKey(objectID string, version uint64) string {
	return fmt.Sprintf("%s@%d", objectID, version)
}

func (m *mockSuiClient) GetObject(ctx context.Context, objectID string, fields []string) (suiclient.SuiObject, error) {
	return m.GetObjectAtVersion(ctx, objectID, nil, fields)
}

func (m *mockSuiClient) GetObjectAtVersion(ctx context.Context, objectID string, version *uint64, fields []string) (suiclient.SuiObject, error) {
	if version == nil {
		return suiclient.SuiObject{}, errors.New("nil version")
	}
	contents, ok := m.objects[mockObjectKey(objectID, *version)]
	if !ok {
		return suiclient.SuiObject{}, fmt.Errorf("object %s@%d not found", objectID, *version)
	}
	return suiclient.SuiObject{ContentsBytes: contents}, nil
}

func (m *mockSuiClient) GetLatestCheckpoint(ctx context.Context, fields []string) (suiclient.SuiCheckpoint, error) {
	sn := uint64(145024142)
	return suiclient.SuiCheckpoint{SequenceNumber: &sn}, nil
}

func (m *mockSuiClient) GetTransaction(ctx context.Context, digest string, fields []string) (suiclient.SuiTransaction, error) {
	txn, ok := m.transactions[digest]
	if !ok {
		return suiclient.SuiTransaction{}, fmt.Errorf("transaction not found: %s", digest)
	}
	return txn, nil
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

func NewSuiWatcherForTest(msgChan chan *common.MessagePublication, suiTxVerifier *txverifier.SuiTransferVerifier, suiEventType string) *Watcher {
	return &Watcher{
		msgChan:           msgChan,
		suiMoveEventType:  suiEventType,
		suiTxVerifier:     suiTxVerifier,
		txVerifierEnabled: true,
	}
}

// Test_DecodeWormholeMessage checks that the BCS-serialized contents of a `WormholeMessage`
// gRPC event decode into the expected fields.
func Test_DecodeWormholeMessage(t *testing.T) {
	bcsBytes := sampleWormholeMessageBcs

	msg, err := suiclient.DecodeBcs[txverifier.WormholeMessage](bcsBytes)
	require.NoError(t, err)
	require.NotNil(t, msg)

	require.Equal(t, sampleSenderHex, hex.EncodeToString(msg.Sender[:]))
	require.Equal(t, sampleSequence, msg.Sequence)
	require.Equal(t, sampleNonce, msg.Nonce)
	require.Equal(t, decodeStringNoError(samplePayloadHex), msg.Payload)
	require.Equal(t, sampleConsistencyLevel, msg.ConsistencyLevel)
	require.Equal(t, sampleTimestamp, msg.Timestamp)
}

// Test_processEvent checks that a `WormholeMessage` gRPC event is decoded and published as a
// MessagePublication with the expected fields when no transfer verifier is configured.
func Test_processEvent(t *testing.T) {
	eventType := "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage"

	msgChan := make(chan *common.MessagePublication, 1)
	// No verifier configured, so processEvent should publish the message directly.
	watcher := &Watcher{
		msgChan:          msgChan,
		suiMoveEventType: eventType,
	}

	event := suiclient.SuiEvent{
		EventType: eventType,
		BcsBytes:  sampleWormholeMessageBcs,
	}

	err := watcher.processEvent(context.TODO(), zap.NewNop(), event, sampleTxDigest, false)
	require.NoError(t, err)

	published := <-msgChan
	expectedTxID, err := base58.Decode(sampleTxDigest)
	require.NoError(t, err)
	require.Len(t, expectedTxID, 32)

	require.Equal(t, expectedTxID, published.TxID)
	require.Equal(t, vaa.ChainIDSui, published.EmitterChain)
	require.Equal(t, sampleSenderHex, published.EmitterAddress.String())
	require.Equal(t, sampleSequence, published.Sequence)
	require.Equal(t, sampleNonce, published.Nonce)
	require.Equal(t, decodeStringNoError(samplePayloadHex), published.Payload)
	require.Equal(t, sampleConsistencyLevel, published.ConsistencyLevel)
	require.Equal(t, testutils.MustTimeFromUnix(t, sampleTimestamp), published.Timestamp)
	require.False(t, published.IsReobservation)
}

// Test_processEvent_IgnoresOtherEventTypes checks that events whose type does not match the
// configured Wormhole message event type are skipped without publishing.
func Test_processEvent_IgnoresOtherEventTypes(t *testing.T) {
	msgChan := make(chan *common.MessagePublication, 1)
	watcher := &Watcher{
		msgChan:          msgChan,
		suiMoveEventType: "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage",
	}

	event := suiclient.SuiEvent{
		EventType: "0xabc::some_module::SomeOtherEvent",
		BcsBytes:  sampleWormholeMessageBcs,
	}

	err := watcher.processEvent(context.TODO(), zap.NewNop(), event, sampleTxDigest, false)
	require.NoError(t, err)
	require.Empty(t, msgChan)
}

// Test_handleReobservation checks that a re-observation request fetches the transaction via the
// gRPC client and publishes its Wormhole message events with IsReobservation set.
func Test_handleReobservation(t *testing.T) {
	eventType := "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage"

	txHash, err := base58.Decode(sampleTxDigest)
	require.NoError(t, err)

	mockClient := &mockSuiClient{
		transactions: map[string]suiclient.SuiTransaction{
			sampleTxDigest: {
				Events: []suiclient.SuiEvent{
					{
						EventType: eventType,
						BcsBytes:  sampleWormholeMessageBcs,
					},
				},
			},
		},
	}

	msgChan := make(chan *common.MessagePublication, 1)
	watcher := &Watcher{
		msgChan:          msgChan,
		suiMoveEventType: eventType,
		suiClient:        mockClient,
	}

	watcher.handleReobservation(context.TODO(), zap.NewNop(), mockClient, &gossipv1.ObservationRequest{
		ChainId: uint32(vaa.ChainIDSui),
		TxHash:  txHash,
	})

	published := <-msgChan
	require.Equal(t, txHash, published.TxID)
	require.Equal(t, sampleSequence, published.Sequence)
	require.True(t, published.IsReobservation)
}

func decodeStringNoError(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

// hexTo32 decodes a 32-byte hex string (with or without a 0x prefix) into a [32]byte.
func hexTo32(s string) [32]byte {
	raw := decodeStringNoError(strings.TrimPrefix(s, "0x"))
	var out [32]byte
	copy(out[:], raw)
	return out
}

func strPtr(s string) *string { return &s }
func u64Ptr(v uint64) *uint64 { return &v }

// Local BCS mirrors of the on-chain native asset dynamic field, used to synthesize object
// contents for the verifier. The layout mirrors the txverifier package's internal structs
// (BCS only depends on field order/types, not the declaring package).
type bcsBytes32T struct{ Data []byte }
type bcsExternalAddressT struct{ Value bcsBytes32T }
type bcsNativeAssetT struct {
	Custody      uint64
	TokenAddress bcsExternalAddressT
	Decimals     uint8
}
type bcsNativeAssetFieldT struct {
	ID    [32]byte
	Name  bool
	Value bcsNativeAssetT
}

func bcsNativeObjectForTest(custody uint64, tokenAddress []byte, decimals uint8) []byte {
	return mystenbcs.MustMarshal(bcsNativeAssetFieldT{
		Value: bcsNativeAssetT{
			Custody:      custody,
			TokenAddress: bcsExternalAddressT{Value: bcsBytes32T{Data: tokenAddress}},
			Decimals:     decimals,
		},
	})
}

// transferPayloadForTest builds a minimal Wormhole token-transfer payload (type 1).
func transferPayloadForTest(amount *big.Int, originAddress []byte, originChain uint16) []byte {
	p := make([]byte, 0, 101)
	p = append(p, 1) // payload type 1 == transfer
	p = append(p, amount.FillBytes(make([]byte, 32))...)
	p = append(p, originAddress...)
	p = append(p, byte(originChain>>8), byte(originChain&0xff))
	p = append(p, make([]byte, 101-len(p))...)
	return p
}

// TestVerifyAndPublish_Samples checks that verifyAndPublish runs the transfer verifier and
// propagates the resulting verification state to the published message. The verifier is driven
// by a mock gRPC client serving a synthesized native-asset transfer; the exhaustive verifier
// scenario coverage lives in the txverifier package.
func TestVerifyAndPublish_Samples(t *testing.T) {
	// Mainnet values.
	suiCoreContract := "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a"
	//nolint:gosec
	suiTokenBridgeEmitter := "0xccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5"
	//nolint:gosec
	suiTokenBridgeContract := "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8eba0697695e3d"
	eventType := fmt.Sprintf("%s::publish_message::WormholeMessage", suiCoreContract)

	emitter32 := hexTo32(suiTokenBridgeEmitter)
	tokenAddress := decodeStringNoError("9258181f5ceac8dbffb7030890243caed69a9599d2886d957a9cb7656af3bdb3")
	nativeObjectType := fmt.Sprintf("0x2::dynamic_field::Field<%s::token_registry::Key<0x2::sui::SUI>, %s::native_asset::NativeAsset<0x2::sui::SUI>>", suiTokenBridgeContract, suiTokenBridgeContract)
	objectId := "0x831c45a8d512c9cf46e7a8a947f7cbbb5e0a59829aa72450ff26fb1873fd0e94"
	txDigest := "3dFwinvN8cxotrbHmisT1zMnFiFeR5nowo9zzZw7akin"
	curVersion := uint64(100)
	prevVersion := uint64(99)

	tests := []struct {
		description   string
		expectedState common.VerificationState
		sequence      uint64
		amount        *big.Int
		custodyAfter  uint64
		custodyBefore uint64
	}{
		// Deposit (custody delta 990) covers the 990 requested out of the bridge.
		{"NativeStandard", common.Valid, 1, big.NewInt(990), 1000, 10},
		// Deposit (custody delta 50) is far short of the 100000 requested out of the bridge.
		{"NativeInsufficientDeposit", common.Anomalous, 2, big.NewInt(100000), 1050, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload := transferPayloadForTest(tt.amount, tokenAddress, uint16(vaa.ChainIDSui))

			event := suiclient.SuiEvent{
				EventType: eventType,
				BcsBytes:  mystenbcs.MustMarshal(txverifier.WormholeMessage{Sender: emitter32, Sequence: tt.sequence, Payload: payload}),
			}
			change := suiclient.SuiObjectChange{
				ObjectID:      strPtr(objectId),
				ObjectType:    strPtr(nativeObjectType),
				OutputVersion: u64Ptr(curVersion),
				InputVersion:  u64Ptr(prevVersion),
			}

			mock := &mockSuiClient{
				transactions: map[string]suiclient.SuiTransaction{
					txDigest: {Events: []suiclient.SuiEvent{event}, ObjectChanges: []suiclient.SuiObjectChange{change}},
				},
				objects: map[string][]byte{
					mockObjectKey(objectId, curVersion):  bcsNativeObjectForTest(tt.custodyAfter, tokenAddress, 8),
					mockObjectKey(objectId, prevVersion): bcsNativeObjectForTest(tt.custodyBefore, tokenAddress, 8),
				},
			}

			suiTxVerifier := txverifier.NewSuiTransferVerifier(suiCoreContract, suiTokenBridgeEmitter, suiTokenBridgeContract, mock)

			msgChan := make(chan *common.MessagePublication, 1)
			testWatcher := NewSuiWatcherForTest(msgChan, suiTxVerifier, eventType)

			msg := &common.MessagePublication{
				TxID:             []byte(txDigest),
				Timestamp:        testutils.MustTimeFromUnix(t, 1747215154),
				Sequence:         tt.sequence,
				EmitterChain:     vaa.ChainIDSui,
				EmitterAddress:   vaa.Address(emitter32),
				Payload:          payload,
				ConsistencyLevel: 0,
			}

			err := testWatcher.verifyAndPublish(context.TODO(), msg, txDigest, zap.NewNop())
			require.NoError(t, err)

			published := <-msgChan
			require.Equal(t, tt.expectedState, published.VerificationState())
		})
	}
}
