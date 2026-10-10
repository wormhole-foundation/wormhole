package txverifier

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ipfslog "github.com/ipfs/go-log/v2"
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

// rpcMockClient is a configurable implementation of the evmClient interface.
// When fn is nil it behaves like mockClient (returns 8 decimals).
type rpcMockClient struct {
	fn func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func (m *rpcMockClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if m.fn != nil {
		return m.fn(ctx, msg, blockNumber)
	}
	return common.LeftPadBytes([]byte{0x08}, 32), nil
}

func setupWithClient(fn func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)) *TransferVerifier[*rpcMockClient, *mockConnector] {
	logger := ipfslog.Logger("wormhole-transfer-verifier-tests").Desugar()
	return &TransferVerifier[*rpcMockClient, *mockConnector]{
		Addresses: &TVAddresses{
			CoreBridgeAddr:    coreBridgeAddr,
			TokenBridgeAddr:   tokenBridgeAddr,
			WrappedNativeAddr: nativeAddrGeth,
		},
		chainIds:            &chainIds{evmChainId: 1, wormholeChainId: vaa.ChainIDEthereum},
		evmConnector:        &mockConnector{},
		client:              &rpcMockClient{fn: fn},
		logger:              *logger,
		evaluations:         make(map[common.Hash]*receiptEvaluation),
		isWrappedCache:      make(map[string]bool),
		chainIdCache:        make(map[string]vaa.ChainID),
		nativeContractCache: make(map[string]vaa.Address),
		decimalsCache:       make(map[common.Address]uint8),
	}
}

func TestGetDecimals(t *testing.T) {
	t.Run("zero address", func(t *testing.T) {
		tv := setupWithClient(nil)
		_, err := tv.getDecimals(ZERO_ADDRESS)
		require.Error(t, err)
	})

	t.Run("cache hit does not call the client", func(t *testing.T) {
		called := false
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			called = true
			return nil, errors.New("should not be called")
		})
		tv.decimalsCache[usdcAddrGeth] = 6
		decimals, err := tv.getDecimals(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, uint8(6), decimals)
		assert.False(t, called, "cache hit must not invoke the client")
	})

	t.Run("client error", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := tv.getDecimals(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("short result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return []byte{0x08}, nil
		})
		_, err := tv.getDecimals(usdcAddrGeth)
		require.ErrorIs(t, err, ErrFailedToGetDecimals)
	})

	t.Run("valid result is cached", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes([]byte{0x06}, 32), nil
		})
		decimals, err := tv.getDecimals(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, uint8(6), decimals)
		assert.Equal(t, uint8(6), tv.decimalsCache[usdcAddrGeth])
	})
}

func TestChainId(t *testing.T) {
	t.Run("zero address", func(t *testing.T) {
		tv := setupWithClient(nil)
		_, err := tv.chainId(ZERO_ADDRESS)
		require.Error(t, err)
	})

	t.Run("cache hit", func(t *testing.T) {
		called := false
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			called = true
			return nil, errors.New("should not be called")
		})
		tv.chainIdCache[usdcAddrGeth.Hex()] = vaa.ChainIDPolygon
		chainID, err := tv.chainId(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDPolygon, chainID)
		assert.False(t, called)
	})

	t.Run("client error", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := tv.chainId(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("short result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return []byte{0x02}, nil
		})
		_, err := tv.chainId(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("unknown chain number", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes(big.NewInt(999999).Bytes(), 32), nil
		})
		_, err := tv.chainId(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("valid result is cached", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes(big.NewInt(int64(vaa.ChainIDPolygon)).Bytes(), 32), nil
		})
		chainID, err := tv.chainId(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDPolygon, chainID)
		assert.Equal(t, vaa.ChainIDPolygon, tv.chainIdCache[usdcAddrGeth.Hex()])
	})
}

func TestNativeContract(t *testing.T) {
	t.Run("zero address", func(t *testing.T) {
		tv := setupWithClient(nil)
		_, err := tv.nativeContract(ZERO_ADDRESS)
		require.Error(t, err)
	})

	t.Run("cache hit", func(t *testing.T) {
		called := false
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			called = true
			return nil, errors.New("should not be called")
		})
		tv.nativeContractCache[usdcAddrGeth.Hex()] = usdcAddrVAA
		addr, err := tv.nativeContract(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, usdcAddrVAA, addr)
		assert.False(t, called)
	})

	t.Run("client error", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := tv.nativeContract(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("short result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return []byte{0x01}, nil
		})
		_, err := tv.nativeContract(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("valid result is cached", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes(usdcAddrGeth.Bytes(), 32), nil
		})
		addr, err := tv.nativeContract(usdcAddrGeth)
		require.NoError(t, err)
		assert.Equal(t, usdcAddrVAA, addr)
		assert.Equal(t, usdcAddrVAA, tv.nativeContractCache[usdcAddrGeth.Hex()])
	})
}

func TestIsWrappedAsset(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		called := false
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			called = true
			return nil, errors.New("should not be called")
		})
		tv.isWrappedCache[usdcAddrGeth.Hex()] = true
		wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
		require.NoError(t, err)
		assert.True(t, wrapped)
		assert.False(t, called)
	})

	t.Run("client error", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		_, err := tv.isWrappedAsset(usdcAddrGeth)
		require.Error(t, err)
	})

	t.Run("short result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return []byte{0x01}, nil
		})
		_, err := tv.isWrappedAsset(usdcAddrGeth)
		require.ErrorIs(t, err, ErrWrappedAssetResultBadLength)
	})

	t.Run("true result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes([]byte{0x01}, 32), nil
		})
		wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
		require.NoError(t, err)
		assert.True(t, wrapped)
	})

	t.Run("false result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return common.LeftPadBytes([]byte{0x00}, 32), nil
		})
		wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
		require.NoError(t, err)
		assert.False(t, wrapped)
	})
}

// depositOnlyReceipt builds a structurally valid receipt containing a single
// native deposit and one message publication.
func depositOnlyReceipt() *TransferReceipt {
	deposits := []*NativeDeposit{
		{
			TokenAddress: nativeAddrGeth,
			TokenChain:   vaa.ChainIDEthereum,
			Receiver:     tokenBridgeAddr,
			Amount:       big.NewInt(1),
		},
	}
	transfers := []*ERC20Transfer{}
	messages := []*LogMessagePublished{
		{
			EventEmitter: coreBridgeAddr,
			MsgSender:    tokenBridgeAddr,
			TransferDetails: &TransferDetails{
				PayloadType:   TransferTokens,
				TokenChain:    vaa.ChainIDEthereum,
				TargetAddress: eoaAddrVAA,
				Amount:        big.NewInt(1),
				OriginAddress: nativeAddrVAA,
			},
		},
	}
	return &TransferReceipt{Deposits: &deposits, Transfers: &transfers, MessagePublications: &messages}
}

// transferOnlyReceipt builds a structurally valid receipt containing a single
// ERC20 transfer and one message publication.
func transferOnlyReceipt() *TransferReceipt {
	deposits := []*NativeDeposit{}
	transfers := []*ERC20Transfer{
		{
			From:         eoaAddrGeth,
			To:           tokenBridgeAddr,
			TokenAddress: usdcAddrGeth,
			TokenChain:   vaa.ChainIDEthereum,
			Amount:       big.NewInt(1),
		},
	}
	messages := []*LogMessagePublished{
		{
			EventEmitter: coreBridgeAddr,
			MsgSender:    tokenBridgeAddr,
			TransferDetails: &TransferDetails{
				PayloadType:   TransferTokens,
				TokenChain:    vaa.ChainIDEthereum,
				TargetAddress: eoaAddrVAA,
				Amount:        big.NewInt(1),
				OriginAddress: usdcAddrVAA,
			},
		},
	}
	return &TransferReceipt{Deposits: &deposits, Transfers: &transfers, MessagePublications: &messages}
}

func TestUpdateReceiptDetailsDepositDecimalsError(t *testing.T) {
	tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
		return nil, errors.New("rpc down")
	})
	err := tv.updateReceiptDetails(depositOnlyReceipt())
	require.Error(t, err)
}

func TestUpdateReceiptDetailsIsWrappedError(t *testing.T) {
	tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
		return nil, errors.New("rpc down")
	})
	err := tv.updateReceiptDetails(transferOnlyReceipt())
	require.Error(t, err)
}

func TestUpdateReceiptDetailsNativeTransfer(t *testing.T) {
	// isWrappedAsset returns false, so the native path is taken.
	tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
		return common.LeftPadBytes([]byte{0x00}, 32), nil
	})
	receipt := transferOnlyReceipt()
	err := tv.updateReceiptDetails(receipt)
	require.NoError(t, err)
	transfer := (*receipt.Transfers)[0]
	assert.Equal(t, usdcAddrVAA, transfer.OriginAddr)
	assert.Equal(t, vaa.ChainIDEthereum, transfer.TokenChain)
}

func TestUpdateReceiptDetailsWrappedTransfer(t *testing.T) {
	// isWrappedAsset returns true; chainId and nativeContract return valid values.
	tv := setupWithClient(func(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		switch {
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE):
			return common.LeftPadBytes([]byte{0x01}, 32), nil
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(WRAPPED_ERC20_CHAIN_ID_SIGNATURE):
			return common.LeftPadBytes(big.NewInt(int64(vaa.ChainIDPolygon)).Bytes(), 32), nil
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(WRAPPED_ERC20_NATIVE_CONTRACT_SIGNATURE):
			return common.LeftPadBytes(usdcAddrGeth.Bytes(), 32), nil
		default:
			return common.LeftPadBytes([]byte{0x08}, 32), nil
		}
	})
	receipt := transferOnlyReceipt()
	err := tv.updateReceiptDetails(receipt)
	require.NoError(t, err)
	transfer := (*receipt.Transfers)[0]
	assert.Equal(t, usdcAddrVAA, transfer.OriginAddr)
	assert.Equal(t, vaa.ChainIDPolygon, transfer.TokenChain)
}

func TestUpdateReceiptDetailsWrappedChainIdError(t *testing.T) {
	tv := setupWithClient(func(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		switch {
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE):
			return common.LeftPadBytes([]byte{0x01}, 32), nil
		default:
			return nil, errors.New("rpc down")
		}
	})
	err := tv.updateReceiptDetails(transferOnlyReceipt())
	require.Error(t, err)
}

func TestUpdateReceiptDetailsWrappedNativeContractError(t *testing.T) {
	tv := setupWithClient(func(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		switch {
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE):
			return common.LeftPadBytes([]byte{0x01}, 32), nil
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(WRAPPED_ERC20_CHAIN_ID_SIGNATURE):
			return common.LeftPadBytes(big.NewInt(int64(vaa.ChainIDPolygon)).Bytes(), 32), nil
		default:
			return nil, errors.New("rpc down")
		}
	})
	err := tv.updateReceiptDetails(transferOnlyReceipt())
	require.Error(t, err)
}

func TestUpdateReceiptDetailsWrappedZeroOriginAddress(t *testing.T) {
	tv := setupWithClient(func(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		switch {
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE):
			return common.LeftPadBytes([]byte{0x01}, 32), nil
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(WRAPPED_ERC20_CHAIN_ID_SIGNATURE):
			return common.LeftPadBytes(big.NewInt(int64(vaa.ChainIDPolygon)).Bytes(), 32), nil
		case len(msg.Data) >= 4 && string(msg.Data[:4]) == string(WRAPPED_ERC20_NATIVE_CONTRACT_SIGNATURE):
			return common.LeftPadBytes(ZERO_ADDRESS.Bytes(), 32), nil
		default:
			return common.LeftPadBytes([]byte{0x08}, 32), nil
		}
	})
	err := tv.updateReceiptDetails(transferOnlyReceipt())
	require.Error(t, err)
}

func TestUpdateReceiptDetailsInvalidReceipt(t *testing.T) {
	tv := setupWithClient(nil)
	// A receipt with nil fields fails SanityCheck.
	err := tv.updateReceiptDetails(&TransferReceipt{})
	require.ErrorIs(t, err, ErrInvalidReceiptArgument)
}
func TestStringMethods(t *testing.T) {
	deposit := &NativeDeposit{
		TokenAddress: nativeAddrGeth,
		TokenChain:   vaa.ChainIDEthereum,
		Receiver:     tokenBridgeAddr,
		Amount:       big.NewInt(1),
	}
	assert.Contains(t, deposit.String(), "Deposit:")

	transfer := &ERC20Transfer{
		TokenAddress: usdcAddrGeth,
		TokenChain:   vaa.ChainIDEthereum,
		From:         eoaAddrGeth,
		To:           tokenBridgeAddr,
		Amount:       big.NewInt(1),
	}
	assert.Contains(t, transfer.String(), "ERC20Transfer:")

	message := &LogMessagePublished{
		EventEmitter: coreBridgeAddr,
		MsgSender:    tokenBridgeAddr,
		TransferDetails: &TransferDetails{
			PayloadType:   TransferTokens,
			TokenChain:    vaa.ChainIDEthereum,
			OriginAddress: usdcAddrVAA,
			TargetAddress: eoaAddrVAA,
			Amount:        big.NewInt(1),
		},
	}
	assert.Contains(t, message.String(), "LogMessagePublished:")

	details := message.TransferDetails
	assert.Contains(t, details.String(), "PayloadType:")

	summary := NewReceiptSummary()
	assert.Contains(t, summary.String(), "receipt summary:")

	inv := &InvariantError{Msg: "boom"}
	assert.Contains(t, inv.Error(), "boom")
}
func TestTransferReceiptStringNilFields(t *testing.T) {
	// All fields nil.
	empty := &TransferReceipt{}
	assert.Contains(t, empty.String(), "receipt:")

	// Non-nil slices containing nil elements.
	deposits := []*NativeDeposit{nil}
	transfers := []*ERC20Transfer{nil}
	messages := []*LogMessagePublished{nil}
	receipt := &TransferReceipt{Deposits: &deposits, Transfers: &transfers, MessagePublications: &messages}
	assert.Contains(t, receipt.String(), "receipt:")
}
func TestReceiptSummaryIsSafeEmpty(t *testing.T) {
	summary := NewReceiptSummary()
	assert.False(t, summary.isSafe(), "an empty summary must not be considered safe")
	assert.Equal(t, 0, summary.invalidMessageCount())
	assert.True(t, summary.allMsgsSafe())
}
func TestIsWrappedAssetCacheFalse(t *testing.T) {
	called := false
	tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
		called = true
		return nil, errors.New("should not be called")
	})
	tv.isWrappedCache[usdcAddrGeth.Hex()] = false
	wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
	require.NoError(t, err)
	assert.False(t, wrapped)
	assert.False(t, called)
}
func TestIsWrappedAssetErrorReturnsFalse(t *testing.T) {
	t.Run("client error", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return nil, errors.New("rpc down")
		})
		wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
		require.Error(t, err)
		assert.False(t, wrapped)
	})

	t.Run("short result", func(t *testing.T) {
		tv := setupWithClient(func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
			return []byte{0x01}, nil
		})
		wrapped, err := tv.isWrappedAsset(usdcAddrGeth)
		require.Error(t, err)
		assert.False(t, wrapped)
	})
}
func TestNewSubscriptionAndAccessors(t *testing.T) {
	sub := NewSubscription(nil, nil)
	require.NotNil(t, sub)
	assert.NotNil(t, sub.Events())
	assert.NotNil(t, sub.Errors())
	assert.NotNil(t, sub.quit)

	// Close must close the quit channel exactly once.
	sub.Close()
	select {
	case <-sub.quit:
	default:
		t.Fatal("quit channel should be closed after Close()")
	}
}

// mockEventSubscription implements event.Subscription for handleSubscription tests.
type mockEventSubscription struct {
	errC         chan error
	unsubscribed bool
}

func (m *mockEventSubscription) Err() <-chan error { return m.errC }
func (m *mockEventSubscription) Unsubscribe()      { m.unsubscribed = true }
func TestHandleSubscription(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		sub := NewSubscription(nil, nil)
		mock := &mockEventSubscription{errC: make(chan error)}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := sub.handleSubscription(ctx, mock)
		require.NoError(t, err)
		assert.True(t, mock.unsubscribed)
	})

	t.Run("quit channel", func(t *testing.T) {
		sub := NewSubscription(nil, nil)
		mock := &mockEventSubscription{errC: make(chan error)}
		sub.Close()

		err := sub.handleSubscription(context.Background(), mock)
		require.NoError(t, err)
		assert.True(t, mock.unsubscribed)
	})

	t.Run("subscription error", func(t *testing.T) {
		sub := NewSubscription(nil, nil)
		mock := &mockEventSubscription{errC: make(chan error, 1)}
		mock.errC <- errors.New("stream failed")

		err := sub.handleSubscription(context.Background(), mock)
		require.Error(t, err)
		assert.True(t, mock.unsubscribed)
	})
}
func TestValidateChainsBoundary(t *testing.T) {
	// 65535 is the maximum uint16 but is not a known chain ID. The boundary
	// comparison must not report it as exceeding MaxUint16.
	_, err := ValidateChains([]uint{65535})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "exceeds MaxUint16")
}
func TestIsSupported(t *testing.T) {
	assert.True(t, IsSupported(vaa.ChainIDEthereum))
	assert.False(t, IsSupported(vaa.ChainIDSolana))
}
func TestVAAAddrFrom(t *testing.T) {
	addr := VAAAddrFrom(usdcAddrGeth)
	assert.Equal(t, usdcAddrVAA, addr)
	assert.Equal(t, common.LeftPadBytes(usdcAddrGeth.Bytes(), 32), addr.Bytes())
}
