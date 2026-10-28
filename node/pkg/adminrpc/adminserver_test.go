//nolint:unparam
package adminrpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	wh_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/prototext"
)

const govGuardianSetIndex = uint32(4)

var govTimestamp = time.Now()

const govEmitterChain = vaa.ChainIDSolana

var govEmitterAddr vaa.Address = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

// verifyGovernanceVAA verifies the VAA fields of a generated governance VAA. Note that it doesn't verify the payload because that is
// already verified in `sdk/vaa/payload_test` and we don't want to duplicate all those arrays.
func verifyGovernanceVAA(t *testing.T, v *vaa.VAA, expectedSeqNo uint64, expectedNonce uint32) {
	t.Helper()
	require.NotNil(t, v)
	assert.Equal(t, uint8(vaa.SupportedVAAVersion), v.Version)
	assert.Equal(t, govGuardianSetIndex, v.GuardianSetIndex)
	assert.Nil(t, v.Signatures)
	assert.Equal(t, govTimestamp, v.Timestamp)
	assert.Equal(t, expectedNonce, v.Nonce)
	assert.Equal(t, expectedSeqNo, v.Sequence)
	assert.Equal(t, uint8(32), v.ConsistencyLevel)
	assert.Equal(t, govEmitterChain, v.EmitterChain)
	assert.True(t, bytes.Equal(govEmitterAddr[:], v.EmitterAddress[:]))
}

// Test_adminCommands executes all of the tests in prototext_test.go, unmarshaling the prototext and feeding it into `GovMsgToVaa`.
func Test_adminCommands(t *testing.T) {
	for _, tst := range adminCommandTest {
		t.Run(tst.label, func(t *testing.T) {
			var msg nodev1.InjectGovernanceVAARequest
			err := prototext.Unmarshal([]byte(tst.prototext), &msg)
			require.NoError(t, err)
			require.Equal(t, 1, len(msg.Messages))
			govMsg := msg.Messages[0]
			govVAA, err := GovMsgToVaa(govMsg, govGuardianSetIndex, govTimestamp)
			if tst.errText == "" {
				require.NoError(t, err)
				verifyGovernanceVAA(t, govVAA, govMsg.Sequence, govMsg.Nonce)
			} else {
				require.ErrorContains(t, err, tst.errText)
			}
		})
	}
}

func newNodePrivilegedServiceForGovernorTests() *nodePrivilegedService {
	gov := governor.NewChainGovernor(zap.NewNop(), &db.MockGovernorDB{}, wh_common.GoTest, false, "")

	return &nodePrivilegedService{
		db:           nil,
		injectC:      nil,
		obsvReqSendC: nil,
		logger:       nil,
		signedInC:    nil,
		governor:     gov,
	}
}

func TestChainGovernorResetReleaseTimer(t *testing.T) {
	service := newNodePrivilegedServiceForGovernorTests()

	// governor has no VAAs enqueued, so if we receive this error we know the input validation passed
	success := `vaa not found in the pending list`
	boundsCheckFailure := `the specified number of days falls outside the range of 1 to 90`
	vaaIdLengthFailure := `the VAA id must be specified as "chainId/emitterAddress/seqNum"`

	tests := map[string]struct {
		vaaId          string
		numDays        uint32
		expectedResult string
	}{
		"EmptyVaaId": {
			vaaId:          "",
			numDays:        1,
			expectedResult: vaaIdLengthFailure,
		},
		"NumDaysEqualsLowerBoundary": {
			vaaId:          "valid",
			numDays:        1,
			expectedResult: success,
		},
		"NumDaysLowerThanLowerBoundary": {
			vaaId:          "valid",
			numDays:        0,
			expectedResult: boundsCheckFailure,
		},
		"NumDaysEqualsUpperBoundary": {
			vaaId:          "valid",
			numDays:        maxResetReleaseTimerDays,
			expectedResult: success,
		},
		"NumDaysExceedsUpperBoundary": {
			vaaId:          "valid",
			numDays:        maxResetReleaseTimerDays + 1,
			expectedResult: boundsCheckFailure,
		},
		"EmptyVaaIdAndNumDaysExceedsUpperBoundary": {
			vaaId:          "",
			numDays:        maxResetReleaseTimerDays + 1,
			expectedResult: vaaIdLengthFailure,
		},
		"NumDaysSignificantlyExceedsUpperBoundary": {
			vaaId:          "valid",
			numDays:        maxResetReleaseTimerDays + 1000,
			expectedResult: boundsCheckFailure,
		},
	}

	ctx := context.Background()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := nodev1.ChainGovernorResetReleaseTimerRequest{
				VaaId:   test.vaaId,
				NumDays: test.numDays,
			}

			_, err := service.ChainGovernorResetReleaseTimer(ctx, &req)
			assert.EqualError(t, err, test.expectedResult)
		})
	}

}

func TestDelegatedGuardiansConfigToVaa(t *testing.T) {
	configJSON := `{
		"5": {
			"keys": [
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333"
			],
			"threshold": 2
		},
		"4": {
			"keys": [
				"0x4444444444444444444444444444444444444444",
				"0x5555555555555555555555555555555555555555"
			],
			"threshold": 1
		}
	}`

	req := &nodev1.DelegatedGuardiansConfig{
		ConfigIndex: 5,
		Config:      configJSON,
	}

	nonce := uint32(12345)
	sequence := uint64(67890)

	v, err := delegatedGuardiansConfigToVaa(req, govTimestamp, govGuardianSetIndex, nonce, sequence)
	require.NoError(t, err)
	require.NotNil(t, v)

	verifyGovernanceVAA(t, v, sequence, nonce)
	assert.NotEmpty(t, v.Payload)
}

func TestDelegatedGuardiansConfigToVaa_InvalidJSON(t *testing.T) {
	configJSON := `not valid json`

	req := &nodev1.DelegatedGuardiansConfig{
		ConfigIndex: 5,
		Config:      configJSON,
	}

	_, err := delegatedGuardiansConfigToVaa(req, time.Now(), 4, 12345, 67890)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestDelegatedGuardiansConfigToVaa_InvalidChainID(t *testing.T) {
	configJSON := `{
		"not_a_number": {
			"keys": ["0x1111111111111111111111111111111111111111"],
			"threshold": 1
		}
	}`

	req := &nodev1.DelegatedGuardiansConfig{
		ConfigIndex: 5,
		Config:      configJSON,
	}

	_, err := delegatedGuardiansConfigToVaa(req, time.Now(), 4, 12345, 67890)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chain ID")
}

func TestCoreBridgeTransferFeesToVaa(t *testing.T) {
	const recipientHex = "00000000000000000000000000000000000000000000000000000000deadbeef"
	const expectedAmount = "0000000000000000000000000000000000000000000000000000000000002710"
	const moduleAndAction = "00000000000000000000000000000000000000000000000000000000436f726504"

	t.Run("Solana spec layout", func(t *testing.T) {
		req := &nodev1.CoreBridgeTransferFees{
			ChainId:   uint32(vaa.ChainIDSolana),
			Amount:    "10000",
			Recipient: recipientHex,
		}
		v, err := coreBridgeTransferFeesToVaa(req, time.Now(), 4, 7, 7)
		require.NoError(t, err)
		// chain (uint16 = 1) || amount || recipient
		expected := moduleAndAction + "0001" + expectedAmount + recipientHex
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})

	t.Run("Injective cosmwasm reversed layout", func(t *testing.T) {
		req := &nodev1.CoreBridgeTransferFees{
			ChainId:   uint32(vaa.ChainIDInjective),
			Amount:    "10000",
			Recipient: recipientHex,
		}
		v, err := coreBridgeTransferFeesToVaa(req, time.Now(), 4, 7, 7)
		require.NoError(t, err)
		// chain (uint16 = 19 = 0x0013) || recipient || amount
		expected := moduleAndAction + "0013" + recipientHex + expectedAmount
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})
}

func TestCoreBridgeTransferFeesToVaa_Errors(t *testing.T) {
	const recipientHex = "00000000000000000000000000000000000000000000000000000000deadbeef"
	base := func() *nodev1.CoreBridgeTransferFees {
		return &nodev1.CoreBridgeTransferFees{
			ChainId:   uint32(vaa.ChainIDSolana),
			Amount:    "1",
			Recipient: recipientHex,
		}
	}
	tests := []struct {
		name    string
		mutate  func(r *nodev1.CoreBridgeTransferFees)
		errText string
	}{
		{"UnknownChain", func(r *nodev1.CoreBridgeTransferFees) { r.ChainId = 0xffff }, "convert chain id"},
		{"InvalidAmount", func(r *nodev1.CoreBridgeTransferFees) { r.Amount = "not-a-number" }, "invalid amount"},
		{"NegativeAmount", func(r *nodev1.CoreBridgeTransferFees) { r.Amount = "-1" }, "amount cannot be negative"},
		{"AmountOverflow", func(r *nodev1.CoreBridgeTransferFees) {
			r.Amount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
		}, "amount overflow"},
		{"ZeroAmount", func(r *nodev1.CoreBridgeTransferFees) { r.Amount = "0" }, "amount must be non-zero"},
		{"BadRecipientHex", func(r *nodev1.CoreBridgeTransferFees) { r.Recipient = "zz" }, "invalid recipient"},
		{"ShortRecipient", func(r *nodev1.CoreBridgeTransferFees) { r.Recipient = "01" }, "invalid recipient"},
		{"LongRecipient", func(r *nodev1.CoreBridgeTransferFees) {
			r.Recipient = recipientHex + "00"
		}, "invalid recipient"},
		{"ZeroRecipient", func(r *nodev1.CoreBridgeTransferFees) {
			r.Recipient = "0000000000000000000000000000000000000000000000000000000000000000"
		}, "recipient must be non-zero"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := coreBridgeTransferFeesToVaa(req, time.Now(), 4, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestCoreBridgeSetMessageFeeToVaa(t *testing.T) {
	const coreModuleAndAction = "00000000000000000000000000000000000000000000000000000000436f726503"
	const expectedFee = "0000000000000000000000000000000000000000000000000000000000002710"

	nonce := uint32(12345)
	sequence := uint64(67890)

	req := &nodev1.CoreBridgeSetMessageFee{
		ChainId:    uint32(vaa.ChainIDSolana),
		MessageFee: "10000",
	}
	v, err := coreBridgeSetMessageFeeToVaa(req, govTimestamp, govGuardianSetIndex, nonce, sequence)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, sequence, nonce)
	// module || action || chain (uint16 = 1) || fee (uint256)
	assert.Equal(t, coreModuleAndAction+"0001"+expectedFee, common.Bytes2Hex(v.Payload))
}

func TestCoreBridgeSetMessageFeeToVaa_Errors(t *testing.T) {
	base := func() *nodev1.CoreBridgeSetMessageFee {
		return &nodev1.CoreBridgeSetMessageFee{
			ChainId:    uint32(vaa.ChainIDSolana),
			MessageFee: "1",
		}
	}
	tests := []struct {
		name    string
		mutate  func(r *nodev1.CoreBridgeSetMessageFee)
		errText string
	}{
		{"UnknownChain", func(r *nodev1.CoreBridgeSetMessageFee) { r.ChainId = 0xffff }, "failed to convert chain id"},
		{"InvalidFee", func(r *nodev1.CoreBridgeSetMessageFee) { r.MessageFee = "not-a-number" }, "invalid new fee"},
		{"FeeOverflow", func(r *nodev1.CoreBridgeSetMessageFee) {
			r.MessageFee = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
		}, "new fee overflow"},
		{"NegativeFee", func(r *nodev1.CoreBridgeSetMessageFee) { r.MessageFee = "-1" }, "new fee cannot be negative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := coreBridgeSetMessageFeeToVaa(req, time.Now(), 4, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestDelegatedManagerSetUpdateToVaa(t *testing.T) {
	const delegatedManagerModuleAndAction = "0000000000000000000000000000000044656c6567617465644d616e61676572" + "01"

	nonce := uint32(12345)
	sequence := uint64(67890)

	req := &nodev1.DelegatedManagerSetUpdate{
		ManagerChainId:  65,
		ManagerSetIndex: 2,
		ManagerSet:      "aabbcc",
	}
	v, err := delegatedManagerSetUpdateToVaa(req, govTimestamp, govGuardianSetIndex, nonce, sequence)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, sequence, nonce)
	// module || action || chain (0 = universal) || manager chain (uint16 = 65) || manager set index (uint32 = 2) || manager set
	assert.Equal(t, delegatedManagerModuleAndAction+"0000"+"0041"+"00000002"+"aabbcc", common.Bytes2Hex(v.Payload))
}

func TestDelegatedManagerSetUpdateToVaa_Errors(t *testing.T) {
	base := func() *nodev1.DelegatedManagerSetUpdate {
		return &nodev1.DelegatedManagerSetUpdate{
			ManagerChainId:  65,
			ManagerSetIndex: 2,
			ManagerSet:      "aabbcc",
		}
	}
	tests := []struct {
		name    string
		mutate  func(r *nodev1.DelegatedManagerSetUpdate)
		errText string
	}{
		{"ManagerChainIdOutOfRange", func(r *nodev1.DelegatedManagerSetUpdate) { r.ManagerChainId = 0x10000 }, "invalid manager_chain_id"},
		{"BadManagerSetHex", func(r *nodev1.DelegatedManagerSetUpdate) { r.ManagerSet = "zz" }, "invalid manager_set encoding"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := delegatedManagerSetUpdateToVaa(req, time.Now(), 4, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestTokenBridgeSetPauserAddresses(t *testing.T) {
	evmPauser := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evmFreezer := "dddddddddddddddddddddddddddddddddddddddd"
	evmUnpauser := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	svmAddr := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	t.Run("EVM all three addresses set", func(t *testing.T) {
		req := &nodev1.BridgeSetPauserAddresses{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDEthereum),
			Pauser:        evmPauser,
			Freezer:       evmFreezer,
			Unpauser:      evmUnpauser,
		}
		v, err := tokenBridgeSetPauserAddresses(req, time.Unix(0, 0), 4, 1, 1)
		require.NoError(t, err)
		// module(32) || action(04) || chain(0002) || pauserLen(14) || pauser || freezerLen(14) || freezer || unpauserLen(14) || unpauser
		expected := "000000000000000000000000000000000000000000546f6b656e427269646765" +
			"04" + "0002" + "14" + evmPauser + "14" + evmFreezer + "14" + evmUnpauser
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})

	t.Run("Solana single address (freezer and unpauser unassigned)", func(t *testing.T) {
		req := &nodev1.BridgeSetPauserAddresses{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDSolana),
			Pauser:        svmAddr,
		}
		v, err := tokenBridgeSetPauserAddresses(req, time.Unix(0, 0), 4, 1, 1)
		require.NoError(t, err)
		expected := "000000000000000000000000000000000000000000546f6b656e427269646765" +
			"04" + "0001" + "20" + svmAddr + "00" + "00"
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})

	t.Run("Freezer set, pauser and unpauser unassigned", func(t *testing.T) {
		req := &nodev1.BridgeSetPauserAddresses{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDEthereum),
			Freezer:       evmFreezer,
		}
		v, err := tokenBridgeSetPauserAddresses(req, time.Unix(0, 0), 4, 1, 1)
		require.NoError(t, err)
		expected := "000000000000000000000000000000000000000000546f6b656e427269646765" +
			"04" + "0002" + "00" + "14" + evmFreezer + "00"
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})

	t.Run("All unassigned", func(t *testing.T) {
		req := &nodev1.BridgeSetPauserAddresses{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDEthereum),
		}
		v, err := tokenBridgeSetPauserAddresses(req, time.Unix(0, 0), 4, 1, 1)
		require.NoError(t, err)
		expected := "000000000000000000000000000000000000000000546f6b656e427269646765" +
			"04" + "0002" + "00" + "00" + "00"
		assert.Equal(t, expected, common.Bytes2Hex(v.Payload))
	})

	t.Run("Error cases", func(t *testing.T) {
		base := func() *nodev1.BridgeSetPauserAddresses {
			return &nodev1.BridgeSetPauserAddresses{
				Module:        "TokenBridge",
				TargetChainId: uint32(vaa.ChainIDEthereum),
				Pauser:        evmPauser,
			}
		}
		cases := []struct {
			name    string
			mutate  func(r *nodev1.BridgeSetPauserAddresses)
			errText string
		}{
			{"UnknownModule", func(r *nodev1.BridgeSetPauserAddresses) { r.Module = "TokemBridge" }, "unknown module"},
			{"EmptyModule", func(r *nodev1.BridgeSetPauserAddresses) { r.Module = "" }, "unknown module"},
			{"NFTBridgeRejected", func(r *nodev1.BridgeSetPauserAddresses) { r.Module = "NFTBridge" }, "unknown module"},
			{"UnknownChain", func(r *nodev1.BridgeSetPauserAddresses) { r.TargetChainId = 0xffff }, "convert target_chain_id"},
			{"ChainOutOfRange", func(r *nodev1.BridgeSetPauserAddresses) { r.TargetChainId = 0x10000 }, "convert target_chain_id"},
			{"BadPauserHex", func(r *nodev1.BridgeSetPauserAddresses) { r.Pauser = "zz" }, "invalid pauser address encoding"},
			{"BadFreezerHex", func(r *nodev1.BridgeSetPauserAddresses) { r.Freezer = "zz" }, "invalid freezer address encoding"},
			{"BadUnpauserHex", func(r *nodev1.BridgeSetPauserAddresses) { r.Unpauser = "zz" }, "invalid unpauser address encoding"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := base()
				tc.mutate(req)
				_, err := tokenBridgeSetPauserAddresses(req, time.Unix(0, 0), 4, 0, 0)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errText)
			})
		}
	})
}

func TestGovMsgToVaa_DispatchesBridgeSetPauserAddresses(t *testing.T) {
	msg := &nodev1.GovernanceMessage{
		Sequence: 33,
		Nonce:    44,
		Payload: &nodev1.GovernanceMessage_BridgeSetPauserAddresses{
			BridgeSetPauserAddresses: &nodev1.BridgeSetPauserAddresses{
				Module:        "TokenBridge",
				TargetChainId: uint32(vaa.ChainIDEthereum),
				Pauser:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Freezer:       "dddddddddddddddddddddddddddddddddddddddd",
				Unpauser:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}
	v, err := GovMsgToVaa(msg, 4, time.Now())
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, uint64(33), v.Sequence)
	assert.Equal(t, uint32(44), v.Nonce)
}

func TestGovMsgToVaa_DispatchesCoreBridgeTransferFees(t *testing.T) {
	msg := &nodev1.GovernanceMessage{
		Sequence: 11,
		Nonce:    22,
		Payload: &nodev1.GovernanceMessage_CoreBridgeTransferFees{
			CoreBridgeTransferFees: &nodev1.CoreBridgeTransferFees{
				ChainId:   uint32(vaa.ChainIDSolana),
				Amount:    "1",
				Recipient: "00000000000000000000000000000000000000000000000000000000deadbeef",
			},
		},
	}
	v, err := GovMsgToVaa(msg, 4, time.Now())
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, uint64(11), v.Sequence)
	assert.Equal(t, uint32(22), v.Nonce)
}

// TestGetAndObserveMissingVAAs_RequestsCarryCurrentTimestamp checks that the observation requests built from the
// missing-VAA list are timestamped now. Receiving guardians drop requests whose timestamp is more than 15 minutes
// from their clock, so a request without one only reaches the local node's watchers.
func TestGetAndObserveMissingVAAs_RequestsCarryCurrentTimestamp(t *testing.T) {
	txHash := "0x2d449abf3d03edeb6c0c8f980a1608207958c5165cde1053e76a1a361150cc62"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"chain":2,"vaaKey":"2/0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585/7","txhash":"` + txHash + `"}]`))
	}))
	defer srv.Close()

	guardianDB := db.OpenDb(zap.NewNop(), nil)
	defer guardianDB.Close()
	obsvReqSendC := make(chan *gossipv1.ObservationRequest, 1)
	service := &nodePrivilegedService{db: guardianDB, obsvReqSendC: obsvReqSendC, logger: zap.NewNop()}

	before := time.Now()
	_, err := service.GetAndObserveMissingVAAs(context.Background(), &nodev1.GetAndObserveMissingVAAsRequest{Url: srv.URL, ApiKey: "key"})
	require.NoError(t, err)
	after := time.Now()

	require.Len(t, obsvReqSendC, 1)
	req := <-obsvReqSendC
	assert.Equal(t, uint32(vaa.ChainIDEthereum), req.ChainId)
	assert.Equal(t, txHash[2:], hex.EncodeToString(req.TxHash))
	sent := time.Unix(0, req.Timestamp)
	assert.False(t, sent.Before(before), "request timestamp %v predates the call (%v)", sent, before)
	assert.False(t, sent.After(after), "request timestamp %v postdates the call (%v)", sent, after)
}
