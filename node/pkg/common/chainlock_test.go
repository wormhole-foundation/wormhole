package common

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// The following constants are used to calculate the offset of each field in the serialized message publication.
const (
	offsetTxIDLength = 0
	// Assumes a length of 32 bytes for the TxID.
	offsetTxID              = offsetTxIDLength + 1
	offsetTimestamp         = offsetTxID + 32
	offsetNonce             = offsetTimestamp + 8
	offsetSequence          = offsetNonce + 4
	offsetConsistencyLevel  = offsetSequence + 8
	offsetEmitterChain      = offsetConsistencyLevel + 1
	offsetEmitterAddress    = offsetEmitterChain + 2
	offsetIsReobservation   = offsetEmitterAddress + 32
	offsetUnreliable        = offsetIsReobservation + 1
	offsetVerificationState = offsetUnreliable + 1
	offsetPayloadLength     = offsetVerificationState + 1
	offsetPayload           = offsetPayloadLength + 8
)

func encodePayloadBytes(payload *vaa.TransferPayloadHdr) []byte {
	bz := make([]byte, 101)
	bz[0] = payload.Type

	amtBytes := payload.Amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}
	copy(bz[33-len(amtBytes):33], amtBytes)

	copy(bz[33:65], payload.OriginAddress.Bytes())
	binary.BigEndian.PutUint16(bz[65:67], uint16(payload.OriginChain))
	copy(bz[67:99], payload.TargetAddress.Bytes())
	binary.BigEndian.PutUint16(bz[99:101], uint16(payload.TargetChain))
	return bz
}

// makeTestMsgPub is a helper function that generates a Message Publication.
func makeTestMsgPub(t *testing.T) *MessagePublication {
	t.Helper()
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes := encodePayloadBytes(payload)

	// Use a non-default value for each field to ensure that the unmarshalled values are represented correctly.
	return &MessagePublication{
		TxID:              eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:         time.Unix(int64(1654516425), 0),
		Nonce:             123456,
		Sequence:          789101112131415,
		EmitterChain:      vaa.ChainIDEthereum,
		EmitterAddress:    tokenBridgeAddress,
		Payload:           payloadBytes,
		ConsistencyLevel:  32,
		Unreliable:        true,
		IsReobservation:   true,
		verificationState: Anomalous,
	}
}

func TestRoundTripMarshal(t *testing.T) {
	orig := makeTestMsgPub(t)
	var loaded MessagePublication

	bz, writeErr := orig.MarshalBinary()
	require.NoError(t, writeErr)
	t.Logf("marshaled bytes: %x", bz)

	readErr := loaded.UnmarshalBinary(bz)
	require.NoError(t, readErr)

	require.Equal(t, *orig, loaded)
}

func TestMessagePublicationUnmarshalBinaryErrors(t *testing.T) {
	orig := makeTestMsgPub(t)
	validBytes, err := orig.MarshalBinary()
	require.Greater(t, len(validBytes), marshaledMsgLenMin)
	require.NoError(t, err)

	tests := []struct {
		name         string
		data         []byte
		expectedErr  error
		errorChecker func(t *testing.T, err error)
		setupData    func() []byte
	}{
		{
			name: "data too short - empty data",
			data: []byte{},
			errorChecker: func(t *testing.T, err error) {
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "data too short")
			},
		},
		{
			name: "data too short - less than minimum size",
			data: make([]byte, marshaledMsgLenMin-1),
			errorChecker: func(t *testing.T, err error) {
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "data too short")
			},
		},
		{
			name:        "invalid IsReobservation boolean - value 0x02",
			expectedErr: ErrInvalidBinaryBool,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetIsReobservation] = 0x02
				return data
			},
		},
		{
			name:        "invalid IsReobservation boolean - value 0xFF",
			expectedErr: ErrInvalidBinaryBool,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetIsReobservation] = 0xFF
				return data
			},
		},
		{
			name:        "invalid Unreliable boolean - value 0x02",
			expectedErr: ErrInvalidBinaryBool,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetUnreliable] = 0x02
				return data
			},
		},
		{
			name:        "invalid Unreliable boolean - value 0xFF",
			expectedErr: ErrInvalidBinaryBool,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetUnreliable] = 0xFF
				return data
			},
		},
		{
			name:        "invalid verification state - at boundary",
			expectedErr: ErrInvalidVerificationState,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetVerificationState] = NumVariantsVerificationState
				return data
			},
		},
		{
			name:        "invalid verification state - above boundary",
			expectedErr: ErrInvalidVerificationState,
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				data[offsetVerificationState] = NumVariantsVerificationState + 1
				return data
			},
		},
		{
			name: "invalid payload length - truncated at payload length",
			errorChecker: func(t *testing.T, err error) {
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "invalid payload length")
			},
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				// Set payload length to be larger than remaining data
				// #nosec G115 -- intentionally creating invalid data for testing
				binary.BigEndian.PutUint64(data[offsetPayloadLength:offsetPayloadLength+8], uint64(len(data)-offsetPayload+1))
				return data
			},
		},
		{
			name: "invalid payload length - no payload data",
			errorChecker: func(t *testing.T, err error) {
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "invalid payload length")
			},
			setupData: func() []byte {
				// Create data that ends right before payload
				data := make([]byte, offsetPayload)
				copy(data, validBytes[:offsetPayload])
				// Set payload length to 1 but provide no payload data
				binary.BigEndian.PutUint64(data[offsetPayloadLength:offsetPayloadLength+8], 1)
				return data
			},
		},
		{
			name: "unexpected end of read - extra bytes",
			errorChecker: func(t *testing.T, err error) {
				var endOfReadErr ErrUnexpectedEndOfRead
				require.ErrorAs(t, err, &endOfReadErr)
				assert.Greater(t, endOfReadErr.expected, endOfReadErr.got)
			},
			setupData: func() []byte {
				data := make([]byte, len(validBytes)+1)
				copy(data, validBytes)
				data[len(validBytes)] = 0xFF // extra byte
				return data
			},
		},
		{
			name: "unexpected end of read - missing bytes",
			errorChecker: func(t *testing.T, err error) {
				// This case actually triggers invalid payload length error first
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "invalid payload length")
			},
			setupData: func() []byte {
				// Create data that's shorter than expected but has valid payload length
				data := make([]byte, len(validBytes)-1)
				copy(data, validBytes[:len(validBytes)-1])
				return data
			},
		},
		{
			name: "payload length overflow - makeslice panic",
			errorChecker: func(t *testing.T, err error) {
				var inputSizeErr ErrInputSize
				require.ErrorAs(t, err, &inputSizeErr)
				assert.Contains(t, inputSizeErr.Error(), "payload length too large")
			},
			setupData: func() []byte {
				data := make([]byte, len(validBytes))
				copy(data, validBytes)
				// Set payload length to maximum uint64 value which would cause makeslice to panic
				binary.BigEndian.PutUint64(data[offsetPayloadLength:offsetPayloadLength+8], math.MaxUint64)
				return data
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			if tt.setupData != nil {
				data = tt.setupData()
			} else {
				data = tt.data
			}

			var mp MessagePublication
			err := mp.UnmarshalBinary(data)

			require.Error(t, err, "expected error for test case: %s", tt.name)

			if tt.errorChecker != nil {
				tt.errorChecker(t, err)
			} else if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "expected specific error type for test case: %s", tt.name)
			}
		})
	}
}

func FuzzMessagePublicationUnmarshalBinary(f *testing.F) {
	// Create a valid message publication for seeding
	orig := &MessagePublication{
		TxID:              make([]byte, TxIDLenMin), // Use minimum valid TxID length
		Timestamp:         time.Unix(int64(1654516425), 0),
		Nonce:             123456,
		Sequence:          789101112131415,
		EmitterChain:      vaa.ChainIDEthereum,
		EmitterAddress:    vaa.Address{0x07, 0x07, 0xf9, 0x11, 0x8e, 0x33, 0xa9, 0xb8, 0x99, 0x8b, 0xea, 0x41, 0xdd, 0x0d, 0x46, 0xf3, 0x8b, 0xb9, 0x63, 0xfc, 0x80},
		Payload:           []byte("test payload"),
		ConsistencyLevel:  32,
		IsReobservation:   true,
		Unreliable:        true,
		verificationState: Valid,
	}

	// Seed with valid marshaled data
	validBytes, err := orig.MarshalBinary()
	if err == nil {
		f.Add(validBytes)
	}

	// Seed with some edge cases
	f.Add([]byte{})                           // empty data
	f.Add(make([]byte, marshaledMsgLenMin-1)) // too short
	f.Add(make([]byte, marshaledMsgLenMin))   // minimum size
	f.Add(make([]byte, 1000))                 // larger data
	// Previous inputs that caused panics
	f.Add([]byte(" 000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\x01\x01\x00\x7f\xff\xff\xff\xff\xff\xff\xed"))
	f.Add([]byte("x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))
	f.Add([]byte(" 000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\x00\x00\x00\xec0000000"))

	f.Add([]byte("\x000000000000000000000000000000000000000000000000000000000\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00 00000000000000000000000000000000"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Catch panics and report them as test failures
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalBinary panicked with input length %d: %v", len(data), r)
			}
		}()

		var mp MessagePublication
		err := mp.UnmarshalBinary(data)

		// The function should never panic, but may return an error
		// We don't assert anything about the error - just that it doesn't crash
		if err == nil {
			// If unmarshaling succeeded, the result should be valid
			// Basic sanity checks on the unmarshaled data
			if len(mp.TxID) > 255 {
				t.Errorf("TxID length %d exceeds maximum of 255", len(mp.TxID))
			}
			if len(mp.TxID) < TxIDLenMin && len(mp.TxID) > 0 {
				t.Errorf("TxID length %d is less than minimum of %d (unless empty)", len(mp.TxID), TxIDLenMin)
			}

			// Verify that a successful unmarshal can be marshaled back
			_, marshalErr := mp.MarshalBinary()
			if marshalErr != nil {
				t.Errorf("Successfully unmarshaled data cannot be marshaled back: %v", marshalErr)
			}

			// Additional invariant checks
			if mp.verificationState >= NumVariantsVerificationState {
				t.Errorf("Invalid verification state %d >= %d", mp.verificationState, NumVariantsVerificationState)
			}
		}
	})
}

// This tests a message publication using the deprecated [Marshal] and [UnmarshalMessagePublication] functions.
// The test and these functions can be removed once the message publication upgrade is complete.
func TestDeprecatedSerializeAndDeserializeOfMessagePublication(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
		// NOTE: these fields are not marshalled or unmarshalled. They are set to non-default values
		// here to prove that they will be unmarshalled into their defaults.
		Unreliable:        true,
		verificationState: Anomalous,
	}

	bz, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bz)
	require.NoError(t, err)

	require.Equal(t, msg1.TxID, msg2.TxID)
	require.Equal(t, msg1.Timestamp, msg2.Timestamp)
	require.Equal(t, msg1.Nonce, msg2.Nonce)
	require.Equal(t, msg1.Sequence, msg2.Sequence)
	require.Equal(t, msg1.EmitterChain, msg2.EmitterChain)
	require.Equal(t, msg1.EmitterAddress, msg2.EmitterAddress)
	require.Equal(t, msg1.ConsistencyLevel, msg2.ConsistencyLevel)
	// These fields are not currently marshalled or unmarshalled. Ensure that the unmarshalled values are equal
	// to the defaults for the types, even if the original struct instance had non-default values.
	require.Equal(t, NotVerified, msg2.verificationState)
	require.Equal(t, false, msg2.Unreliable)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, payload1, payload2)
}

// This tests a message publication using the deprecated [Marshal] and [UnmarshalMessagePublication] functions.
// The test and these functions can be removed once the message publication upgrade is complete.
func TestSerializeAndDeserializeOfMessagePublicationWithEmptyTxID(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxID:             []byte{},
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bz, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bz)
	require.NoError(t, err)
	assert.Equal(t, msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, payload1, payload2)
}

// This tests a message publication using the deprecated [Marshal] and [UnmarshalMessagePublication] functions.
// The test and these functions can be removed once the message publication upgrade is complete.
func TestSerializeAndDeserializeOfMessagePublicationWithArbitraryTxID(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxID:             []byte("This is some arbitrary string with just some random junk in it. This is to prove that the TxID does not have to be a ethCommon.Hash"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bz, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bz)
	require.NoError(t, err)
	assert.Equal(t, msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, payload1, payload2)
}

// This tests a message publication using the deprecated [Marshal] and [UnmarshalMessagePublication] functions.
// The test and these functions can be removed once the message publication upgrade is complete.
func TestTxIDStringTooLongShouldFail(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// This is limited to 255. Make it 256 and the marshal should fail.
	txID := []byte("0123456789012345678901234567890123456789012345678901234567890123012345678901234567890123456789012345678901234567890123456789012301234567890123456789012345678901234567890123456789012345678901230123456789012345678901234567890123456789012345678901234567890123")

	msg := &MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          []byte("Hello, World!"),
		ConsistencyLevel: 32,
	}

	_, err = msg.Marshal()
	assert.ErrorContains(t, err, "TxID too long")
}

// This tests a message publication using the deprecated [Marshal] and [UnmarshalMessagePublication] functions.
// The test and these functions can be removed once the message publication upgrade is complete.
func TestSerializeAndDeserializeOfMessagePublicationWithBigPayload(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// Create a payload of more than 1000 bytes.
	var payload1 []byte
	for i := 0; i < 2000; i++ {
		ch := i % 255
		payload1 = append(payload1, byte(ch))
	}

	msg1 := &MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payload1,
		ConsistencyLevel: 32,
	}

	bz, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bz)
	require.NoError(t, err)

	assert.Equal(t, msg1, msg2)
}

func TestMarshalUnmarshalJSONOfMessagePublication(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bz, err := msg1.MarshalJSON()
	require.NoError(t, err)

	var msg2 MessagePublication
	err = msg2.UnmarshalJSON(bz)
	require.NoError(t, err)
	assert.Equal(t, *msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, *payload1, *payload2)
}

func TestMarshalUnmarshalJSONOfMessagePublicationWithArbitraryTxID(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxID:             []byte("This is some arbitrary string with just some random junk in it. This is to prove that the TxID does not have to be a ethCommon.Hash"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bz, err := msg1.MarshalJSON()
	require.NoError(t, err)

	var msg2 MessagePublication
	err = msg2.UnmarshalJSON(bz)
	require.NoError(t, err)
	assert.Equal(t, *msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, *payload1, *payload2)
}

func TestMessageIDString(t *testing.T) {
	addr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	type test struct {
		label  string
		input  MessagePublication
		output string
	}

	tests := []test{
		{label: "simple",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1"},
		{label: "missing sequence",
			input:  MessagePublication{EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"},
		{label: "missing chain id",
			input:  MessagePublication{Sequence: 1, EmitterAddress: addr},
			output: "0/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1"},
		{label: "missing emitter address",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum},
			output: "2/0000000000000000000000000000000000000000000000000000000000000000/1"},
		{label: "empty message",
			input:  MessagePublication{},
			output: "0/0000000000000000000000000000000000000000000000000000000000000000/0"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.Equal(t, tc.output, tc.input.MessageIDString())
		})
	}
}

func TestMessageID(t *testing.T) {
	addr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	type test struct {
		label  string
		input  MessagePublication
		output []byte
	}

	tests := []test{
		{label: "simple",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: []byte("2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1")},
		{label: "missing sequence",
			input:  MessagePublication{EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: []byte("2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")},
		{label: "missing chain id",
			input:  MessagePublication{Sequence: 1, EmitterAddress: addr},
			output: []byte("0/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1")},
		{label: "missing emitter address",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum},
			output: []byte("2/0000000000000000000000000000000000000000000000000000000000000000/1")},
		{label: "empty message",
			input:  MessagePublication{},
			output: []byte("0/0000000000000000000000000000000000000000000000000000000000000000/0")},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.Equal(t, tc.output, tc.input.MessageID())
		})
	}
}

func TestTxIDStringMatchesHashToString(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	expectedHashID := "0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"

	msg := &MessagePublication{
		TxID:             eth_common.HexToHash(expectedHashID).Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          []byte("Hello, World!"),
		ConsistencyLevel: 32,
	}

	assert.Equal(t, expectedHashID, msg.TxIDString())
}

func TestMessagePublication_SetVerificationState(t *testing.T) {
	tests := []struct {
		name    string
		initial VerificationState
		arg     VerificationState
		wantErr bool
	}{
		{
			"can't overwrite existing status with default value",
			Valid,
			NotVerified,
			true,
		},
		{
			"can't overwrite with the same value",
			Valid,
			Valid,
			true,
		},
		{
			"happy path: default status to non-default",
			NotVerified,
			Valid,
			false,
		},
		{
			"happy path: non-default status to non-default",
			Rejected,
			Valid,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MessagePublication{
				verificationState: tt.initial,
			}
			if err := msg.SetVerificationState(tt.arg); (err != nil) != tt.wantErr {
				t.Errorf("MessagePublication.SetVerificationState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafeRead(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			"happy path",
			MaxSafeInputSize,
			false,
		},
		{
			"error: too big",
			MaxSafeInputSize + 1,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file and write bytes to it
			tmp := os.TempDir()

			f, err := os.CreateTemp(tmp, "tmpfile-")
			require.NoError(t, err)

			defer f.Close()
			defer os.Remove(f.Name())

			// Fill slice with zeroes.
			data := make([]byte, tt.size)
			if _, err := f.Write(data); err != nil {
				require.NoError(t, err)
			}

			// File pointer is at EOF at this point. Reset to the start.
			_, err = f.Seek(0, io.SeekStart)
			require.NoError(t, err)

			got, err := SafeRead(f)
			if tt.wantErr {
				require.Error(t, err, "SafeRead() should have returned an error")
				require.Nil(t, got, "got should be nil when error occurs")
			} else {
				require.NoError(t, err, "SafeRead() should not have returned an error")
				require.NotNil(t, got, "got should not be nil when no error occurs")
				require.True(t, bytes.Equal(got, data), "bytes read are not equal to bytes written")
			}
		})
	}
}

func TestMessagePublication_IsWTT(t *testing.T) {
	// Using real mainnet and testnet token bridge emitter addresses as hex strings
	const (
		ethTokenBridgeHex        = "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
		
		solanaTokenBridgeHex     = "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5" // #nosec G101 -- addresses, not secrets
		wrongEmitterHex          = "0000000000000000000000000000000000000000000000000000000000000001"
		ethTestnetTokenBridgeHex = "000000000000000000000000f890982f9310df57d00f659cf4fd87e65aded8d7"
	)

	tests := []struct {
		name           string
		emitterChain   vaa.ChainID
		emitterAddrHex string
		payload        []byte
		env            Environment
		want           bool
	}{
		{
			name:           "happy path - valid mainnet WTT from Ethereum",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01}, // Transfer payload type
			env:            MainNet,
			want:           true,
		},
		{
			name:           "happy path - valid mainnet WTT from Solana with payload type 3",
			emitterChain:   vaa.ChainIDSolana,
			emitterAddrHex: solanaTokenBridgeHex,
			payload:        []byte{0x03}, // Transfer with payload type
			env:            MainNet,
			want:           true,
		},
		{
			name:           "failure - wrong payload type",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x02}, // Not a transfer payload
			env:            MainNet,
			want:           false,
		},
		{
			name:           "failure - chain without token bridge in environment",
			emitterChain:   vaa.ChainIDCosmoshub,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            MainNet,
			want:           false,
		},
		{
			name:           "failure - emitter address doesn't match token bridge",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: solanaTokenBridgeHex,
			payload:        []byte{0x01},
			env:            MainNet,
			want:           false,
		},
		{
			name:           "failure - test environment (GoTest)",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            GoTest,
			want:           false,
		},
		{
			name:           "failure - mock environment (AccountantMock)",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            AccountantMock,
			want:           false,
		},
		{
			name:           "failure - empty payload",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{},
			env:            MainNet,
			want:           false,
		},
		{
			name:           "valid testnet WTT from Ethereum",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTestnetTokenBridgeHex,
			payload:        []byte{0x01},
			env:            TestNet,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitterAddr, err := vaa.StringToAddress(tt.emitterAddrHex)
			require.NoError(t, err)

			msg := &MessagePublication{
				EmitterChain:   tt.emitterChain,
				EmitterAddress: emitterAddr,
				Payload:        tt.payload,
			}

			got := msg.IsWTT(tt.env)
			assert.Equal(t, tt.want, got)
		})
	}
}
