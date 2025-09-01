package txverifier

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// Mainnet values
	WETH_ADDRESS                = common.HexToAddress("c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	NATIVE_CHAIN_ID vaa.ChainID = 2
)

func TestRelevantDeposit(t *testing.T) {
	t.Parallel()

	// The expected return values for relevant()
	type result struct {
		key      string
		relevant bool
	}

	mocks := setup()

	deposits := map[string]struct {
		input    NativeDeposit
		expected result
	}{
		"relevant, deposit": {
			input: NativeDeposit{
				TokenAddress: nativeAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       big.NewInt(500),
			},
			expected: result{"000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2-2", true},
		},
		"irrelevant, deposit from non-native contract": {
			input: NativeDeposit{
				TokenAddress: usdcAddrGeth, // not Native
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       big.NewInt(500),
			},
			expected: result{"", false},
		},
		"irrelevant, deposit not sent to token bridge": {
			input: NativeDeposit{
				TokenAddress: nativeAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     eoaAddrGeth, // not token bridge
				Amount:       big.NewInt(500),
			},
			expected: result{"", false},
		},
		"irrelevant, sanity check for zero-address deposits": {
			input: NativeDeposit{
				TokenAddress: ZERO_ADDRESS, // zero address
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       big.NewInt(500),
			},
			expected: result{"", false},
		},
	}

	transfers := map[string]struct {
		input    ERC20Transfer
		expected result
	}{
		"relevant transfer": {
			input: ERC20Transfer{
				TokenAddress: nativeAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         eoaAddrGeth,
				To:           tokenBridgeAddr,
				Amount:       big.NewInt(500),
				OriginAddr:   nativeAddrVAA,
			},
			expected: result{"000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2-2", true},
		},
		"irrelevant transfer: destination is not token bridge": {
			input: ERC20Transfer{
				TokenAddress: nativeAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         eoaAddrGeth,
				To:           eoaAddrGeth,
				Amount:       big.NewInt(500),
				OriginAddr:   nativeAddrVAA,
			},
			expected: result{"", false},
		},
	}

	messages := map[string]struct {
		input    LogMessagePublished
		expected result
	}{
		"relevant LogMessagePublished": {
			input: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					TokenChain:    NATIVE_CHAIN_ID,
					PayloadType:   TransferTokens,
					OriginAddress: nativeAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
			expected: result{"000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2-2", true},
		},
		"irrelevant LogMessagePublished: sender not equal to token bridge": {
			input: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    eoaAddrGeth,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: nativeAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
			expected: result{"", false},
		},
		"irrelevant LogMessagePublished: not emitted by core bridge": {
			input: LogMessagePublished{
				EventEmitter: tokenBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: nativeAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
			expected: result{"", false},
		},
		"irrelevant LogMessagePublished: does not have a PayloadType corresponding to a Transfer": {
			input: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   2,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: nativeAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
			expected: result{"", false},
		},
	}

	for name, test := range deposits {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			key, relevant := relevant[*NativeDeposit](&test.input, mocks.transferVerifier.Addresses)
			assert.Equal(t, test.expected.key, key)
			assert.Equal(t, test.expected.relevant, relevant)

			if key == "" {
				assert.False(t, relevant, "key must be empty for irrelevant transfers, but got ", key)
			} else {
				assert.True(t, relevant, "relevant must be true for non-empty keys")
			}
		})
	}

	for name, test := range transfers {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			key, relevant := relevant[*ERC20Transfer](&test.input, mocks.transferVerifier.Addresses)
			assert.Equal(t, test.expected.key, key)
			assert.Equal(t, test.expected.relevant, relevant)

			if key == "" {
				assert.False(t, relevant, "key must be empty for irrelevant transfers, but got ", key)
			} else {
				assert.True(t, relevant, "relevant must be true for non-empty keys")
			}
		})
	}

	for name, test := range messages {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			key, relevant := relevant[*LogMessagePublished](&test.input, mocks.transferVerifier.Addresses)
			assert.Equal(t, test.expected.key, key)
			assert.Equal(t, test.expected.relevant, relevant)

			if key == "" {
				assert.False(t, relevant, "key must be empty for irrelevant transfers, but got ", key)
			} else {
				assert.True(t, relevant, "relevant must be true for non-empty keys")
			}
		})
	}
}

func TestValidateDeposit(t *testing.T) {
	t.Parallel()

	invalidDeposits := map[string]struct {
		deposit NativeDeposit
	}{
		"invalid: zero-value for TokenAddress": {
			deposit: NativeDeposit{
				// TokenAddress:
				TokenChain: NATIVE_CHAIN_ID,
				Receiver:   tokenBridgeAddr,
				Amount:     big.NewInt(1),
			},
		},
		"invalid: zero-value for TokenChain": {
			deposit: NativeDeposit{
				TokenAddress: usdcAddrGeth,
				// TokenChain:
				Receiver: tokenBridgeAddr,
				Amount:   big.NewInt(1),
			},
		},
		"invalid: zero-value for Receiver": {
			deposit: NativeDeposit{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				// Receiver:
				Amount: big.NewInt(1),
			},
		},
		"invalid: nil Amount": {
			deposit: NativeDeposit{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       nil,
			},
		},
		"invalid: negative Amount": {
			deposit: NativeDeposit{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       big.NewInt(-1),
			},
		},
	}

	for name, test := range invalidDeposits {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*NativeDeposit](&test.deposit)
			require.Error(t, err)
		})
	}

	validDeposits := map[string]struct {
		deposit NativeDeposit
	}{
		"valid": {
			deposit: NativeDeposit{
				TokenAddress: nativeAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				Receiver:     tokenBridgeAddr,
				Amount:       big.NewInt(500),
			},
		},
	}

	for name, test := range validDeposits {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*NativeDeposit](&test.deposit)
			require.NoError(t, err)

			// Test the interface
			assert.Equal(t, test.deposit.TokenAddress, test.deposit.Emitter())
			assert.NotEqual(t, ZERO_ADDRESS, test.deposit.OriginAddress())
		})
	}
}

func TestValidateERC20Transfer(t *testing.T) {
	t.Parallel()

	invalidTransfers := map[string]struct {
		input ERC20Transfer
	}{
		"invalid: zero-value for TokenAddress": {
			input: ERC20Transfer{
				// TokenAddress:
				TokenChain: NATIVE_CHAIN_ID,
				To:         tokenBridgeAddr,
				From:       eoaAddrGeth,
				Amount:     big.NewInt(1),
			},
		},
		"invalid: zero-value for TokenChain": {
			input: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				// TokenChain:
				To:     tokenBridgeAddr,
				From:   eoaAddrGeth,
				Amount: big.NewInt(1),
			},
		},
		// Note: transfer's To and From values are allowed to be the zero address.
		"invalid: nil Amount": {
			input: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         eoaAddrGeth,
				To:           tokenBridgeAddr,
				Amount:       nil,
			},
		},
		"invalid: negative Amount": {
			input: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         eoaAddrGeth,
				To:           tokenBridgeAddr,
				Amount:       big.NewInt(-1),
			},
		},
	}

	for name, test := range invalidTransfers {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*ERC20Transfer](&test.input)
			require.Error(t, err)
			assert.ErrorContains(t, err, "invalid log")
		})
	}

	validTransfers := map[string]struct {
		transfer ERC20Transfer
	}{
		"valid": {
			transfer: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				To:           tokenBridgeAddr,
				From:         eoaAddrGeth,
				Amount:       big.NewInt(100),
				OriginAddr:   usdcAddrVAA,
			},
		},
		"valid: zero-value for From (possible Transfer event from non-ERC20 contract)": {
			transfer: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         ZERO_ADDRESS,
				To:           tokenBridgeAddr,
				Amount:       big.NewInt(1),
				OriginAddr:   usdcAddrVAA,
			},
		},
		"valid: zero-value for To (burning funds)": {
			transfer: ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				TokenChain:   NATIVE_CHAIN_ID,
				From:         tokenBridgeAddr,
				To:           ZERO_ADDRESS,
				Amount:       big.NewInt(1),
				OriginAddr:   usdcAddrVAA,
			},
		},
	}

	for name, test := range validTransfers {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*ERC20Transfer](&test.transfer)
			require.NoError(t, err)

			// Test interface
			assert.Equal(t, test.transfer.TokenAddress, test.transfer.Emitter())
			assert.NotEqual(t, ZERO_ADDRESS, test.transfer.OriginAddress())
		})
	}
}

func TestValidateLogMessagePublished(t *testing.T) {
	t.Parallel()

	invalidMessages := map[string]struct {
		logMessagePublished LogMessagePublished
	}{
		"invalid: zero-value for EventEmitter": {
			logMessagePublished: LogMessagePublished{
				// EventEmitter: coreBridgeAddr,
				MsgSender: tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"invalid: zero-value for MsgSender": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				// MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"invalid: zero-value for TransferDetails": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				// TransferDetails: &TransferDetails{
				// 	PayloadType:     TransferTokens,
				// 	TokenChain:      NATIVE_CHAIN_ID,
				// 	OriginAddress:   eoaAddrGeth,
				// 	TargetAddress:   eoaAddrVAA,
				// 	Amount:          big.NewInt(7),
				// },
			},
		},
		"invalid: zero-value for PayloadType": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					// PayloadType:     TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"invalid: zero-value for TokenChain": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType: TransferTokens,
					// TokenChain:      NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"invalid: zero-value for OriginAddress": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType: TransferTokens,
					TokenChain:  NATIVE_CHAIN_ID,
					// OriginAddress:   usdcAddr,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"invalid: zero-value for TargetAddress": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					// TargetAddress:   eoaAddrVAA,
					Amount: big.NewInt(7),
				},
			},
		},
		"invalid: nil Amount": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					// Amount:          big.NewInt(7),
				},
			},
		},
		"invalid: negative Amount": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(-1),
				},
			},
		},
		"invalid: msg.sender cannot be equal to emitter": {
			logMessagePublished: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    coreBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: usdcAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(1),
				},
			},
		},
	}

	for name, test := range invalidMessages {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*LogMessagePublished](&test.logMessagePublished)
			require.Error(t, err)
			var invalidErr *InvalidLogError
			ok := errors.As(err, &invalidErr)
			assert.True(t, ok, "wrong error type: ", err.Error())
		})
	}

	validTransfers := map[string]struct {
		input LogMessagePublished
	}{
		"valid and relevant": {
			input: LogMessagePublished{
				EventEmitter: coreBridgeAddr,
				MsgSender:    tokenBridgeAddr,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokens,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: eoaAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
		"valid and irrelevant": {
			input: LogMessagePublished{
				EventEmitter: usdcAddrGeth,
				MsgSender:    eoaAddrGeth,
				TransferDetails: &TransferDetails{
					PayloadType:   TransferTokensWithPayload,
					TokenChain:    NATIVE_CHAIN_ID,
					OriginAddress: eoaAddrVAA,
					TargetAddress: eoaAddrVAA,
					Amount:        big.NewInt(7),
				},
			},
		},
	}

	for name, test := range validTransfers {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			err := validate[*LogMessagePublished](&test.input)
			require.NoError(t, err)
		})
	}
}

func TestCmp(t *testing.T) {

	t.Parallel()

	// Table-driven tests were not used here because the function takes generic types which are awkward to declare
	// in that format.

	// Test identity
	assert.Zero(t, Cmp(ZERO_ADDRESS, ZERO_ADDRESS))
	assert.Zero(t, Cmp(ZERO_ADDRESS_VAA, ZERO_ADDRESS))

	// Test mixed types
	assert.Zero(t, Cmp(ZERO_ADDRESS, ZERO_ADDRESS_VAA))
	assert.Zero(t, Cmp(ZERO_ADDRESS_VAA, ZERO_ADDRESS_VAA))

	vaaAddr, err := vaa.BytesToAddress([]byte{0x01})
	require.NoError(t, err)
	assert.Zero(t, Cmp(vaaAddr, common.BytesToAddress([]byte{0x01})))

	vaaAddr, err = vaa.BytesToAddress([]byte{0xff, 0x02})
	require.NoError(t, err)
	assert.Zero(t, Cmp(common.BytesToAddress([]byte{0xff, 0x02}), vaaAddr))
}

func TestVAAFromAddr(t *testing.T) {

	t.Parallel()

	// Test values. Declared here in order to silence error values from the vaa functions.
	vaa1, _ := vaa.BytesToAddress([]byte{0xff, 0x02})
	vaa2, _ := vaa.StringToAddress("0000000000000000000000002260fac5e5542a773aa44fbcfedf7c193bc2c599")

	tests := map[string]struct {
		input    common.Address
		expected vaa.Address
	}{
		"valid, arbitrary": {
			input:    common.BytesToAddress([]byte{0xff, 0x02}),
			expected: vaa1,
		},
		"valid, zero values": {
			input:    ZERO_ADDRESS,
			expected: ZERO_ADDRESS_VAA,
		},
		"valid, string-based": {
			input:    common.HexToAddress("0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"),
			expected: vaa2,
		},
	}

	for name, test := range tests {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			res := VAAAddrFrom(test.input)
			assert.Equal(t, test.expected, res)
			assert.Zero(t, bytes.Compare(res[:], common.LeftPadBytes(test.input.Bytes(), EVM_WORD_LENGTH)))
		})
	}

}

func TestDepositFrom(t *testing.T) {

	t.Parallel()

	tests := map[string]struct {
		log      types.Log
		expected *NativeDeposit
	}{
		"valid deposit": {
			log: types.Log{
				Address: WETH_ADDRESS,
				Topics: []common.Hash{
					common.HexToHash(EVENTHASH_WETH_DEPOSIT),
					// Receiver
					common.HexToHash(tokenBridgeAddr.String()),
				},
				TxHash: common.BytesToHash([]byte{0x01}),
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), EVM_WORD_LENGTH),
			},
			expected: &NativeDeposit{
				Receiver:     tokenBridgeAddr,
				TokenAddress: WETH_ADDRESS,
				// Default token chain for a transfer.
				TokenChain: NATIVE_CHAIN_ID,
				Amount:     big.NewInt(100),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			deposit, err := DepositFromLog(&test.log, NATIVE_CHAIN_ID)
			assert.Equal(t, test.expected, deposit)
			require.NoError(t, err)
		})
	}

}

func TestParseERC20TransferFrom(t *testing.T) {

	t.Parallel()

	tests := map[string]struct {
		log      types.Log
		expected *ERC20Transfer
	}{
		"valid transfer": {
			log: types.Log{
				Address: usdcAddrGeth,
				Topics: []common.Hash{
					common.HexToHash(EVENTHASH_ERC20_TRANSFER),
					// From
					common.HexToHash(eoaAddrGeth.String()),
					// To
					common.HexToHash(tokenBridgeAddr.String()),
				},
				TxHash: common.BytesToHash([]byte{0x01}),
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), EVM_WORD_LENGTH),
			},
			expected: &ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				// Default token chain for a transfer.
				TokenChain: NATIVE_CHAIN_ID,
				From:       eoaAddrGeth,
				To:         tokenBridgeAddr,
				Amount:     big.NewInt(100),
			},
		},
		"valid transfer: burn action": {
			log: types.Log{
				Address: usdcAddrGeth,
				Topics: []common.Hash{
					common.HexToHash(EVENTHASH_ERC20_TRANSFER),
					// From
					common.HexToHash(eoaAddrGeth.String()),
					// To is equal to the zero-address for burn transfers
					common.HexToHash(ZERO_ADDRESS.String()),
				},
				TxHash: common.BytesToHash([]byte{0x01}),
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), EVM_WORD_LENGTH),
			},
			expected: &ERC20Transfer{
				TokenAddress: usdcAddrGeth,
				// Default token chain for a transfer.
				TokenChain: NATIVE_CHAIN_ID,
				From:       eoaAddrGeth,
				To:         ZERO_ADDRESS,
				Amount:     big.NewInt(100),
			},
		},
	}

	for name, test := range tests {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			transfer, err := ERC20TransferFromLog(&test.log, NATIVE_CHAIN_ID)
			assert.Equal(t, test.expected, transfer)
			require.NoError(t, err)
		})
	}

	invalidTests := map[string]struct {
		log types.Log
	}{
		"invalid transfer: From and To are both equal to the zero address": {
			log: types.Log{
				Address: usdcAddrGeth,
				Topics: []common.Hash{
					common.HexToHash(EVENTHASH_ERC20_TRANSFER),
					// From
					common.HexToHash(ZERO_ADDRESS.String()),
					// To
					common.HexToHash(ZERO_ADDRESS.String()),
				},
				TxHash: common.BytesToHash([]byte{0x01}),
				Data:   common.LeftPadBytes(big.NewInt(100).Bytes(), EVM_WORD_LENGTH),
			},
		},
	}

	for name, invalidTest := range invalidTests {
		test := invalidTest // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			transfer, err := ERC20TransferFromLog(&test.log, NATIVE_CHAIN_ID)
			require.Error(t, err)
			assert.Nil(t, transfer)
		})
	}

}
