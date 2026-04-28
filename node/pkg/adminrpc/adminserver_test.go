//nolint:unparam
package adminrpc

import (
	"bytes"
	"context"
	"testing"
	"time"

	wh_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	dgAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/delegated_guardians"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/prototext"
)

type mockEVMConnector struct {
	guardianAddrs    []common.Address
	guardianSetIndex uint32
}

func (m mockEVMConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return m.guardianSetIndex, nil
}

func (m mockEVMConnector) GetGuardianSet(ctx context.Context, index uint32) (ethabi.StructsGuardianSet, error) {
	return ethabi.StructsGuardianSet{
		Keys:           m.guardianAddrs,
		ExpirationTime: 0,
	}, nil
}

func (m mockEVMConnector) GetDelegatedGuardianConfig(ctx context.Context) ([]dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	panic("unimplemented")
}

func (m mockEVMConnector) NetworkName() string {
	panic("unimplemented")
}

func (m mockEVMConnector) ContractAddress() common.Address {
	panic("unimplemented")
}

func (m mockEVMConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethabi.AbiLogMessagePublished) (event.Subscription, error) {
	panic("unimplemented")
}

func (m mockEVMConnector) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	panic("unimplemented")
}

func (m mockEVMConnector) TimeOfBlockByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	panic("unimplemented")
}

func (m mockEVMConnector) ParseLogMessagePublished(log types.Log) (*ethabi.AbiLogMessagePublished, error) {
	panic("unimplemented")
}

func (m mockEVMConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *connectors.NewBlock) (ethereum.Subscription, error) {
	panic("unimplemented")
}

func (e mockEVMConnector) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	panic("unimplemented")
}

func (m mockEVMConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	panic("unimplemented")
}

func (m mockEVMConnector) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	panic("unimplemented")
}

func (c mockEVMConnector) Client() *ethclient.Client {
	panic("unimplemented")
}

func (c mockEVMConnector) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	panic("unimplemented")
}

func generateGuardianSigners(num int) (signers []guardiansigner.GuardianSigner, addrs []common.Address) {
	for i := 0; i < num; i++ {
		signer, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
		if err != nil {
			panic(err)
		}
		signers = append(signers, signer)
		addrs = append(addrs, ethcrypto.PubkeyToAddress(signer.PublicKey(context.Background())))
	}
	return
}

func addrsToHexStrings(addrs []common.Address) (out []string) {
	for _, addr := range addrs {
		out = append(out, addr.String())
	}
	return
}

func generateMockVAA(gsIndex uint32, signers []guardiansigner.GuardianSigner, t *testing.T) []byte {
	t.Helper()
	v := &vaa.VAA{
		Version:          1,
		GuardianSetIndex: gsIndex,
		Signatures:       nil,
		Timestamp:        time.Now(),
		Nonce:            3,
		Sequence:         79,
		ConsistencyLevel: 1,
		EmitterChain:     1,
		EmitterAddress:   vaa.Address{},
		Payload:          []byte("test"),
	}
	for i, signer := range signers {
		sig, err := signer.Sign(context.Background(), v.SigningDigest().Bytes())
		if err != nil {
			require.NoError(t, err)
		}

		signature := [ecdsaSignatureLength]byte{}
		copy(signature[:], sig)

		v.Signatures = append(v.Signatures, &vaa.Signature{
			Index:     uint8(i), // #nosec G115 -- This conversion is safe based on the constants used
			Signature: signature,
		})

	}

	vBytes, err := v.Marshal()
	if err != nil {
		panic(err)
	}
	return vBytes
}

func setupAdminServerForVAASigning(gsIndex uint32, gsAddrs []common.Address) *nodePrivilegedService {
	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	if err != nil {
		panic(err)
	}

	connector := mockEVMConnector{
		guardianAddrs:    gsAddrs,
		guardianSetIndex: gsIndex,
	}

	return &nodePrivilegedService{
		db:              nil,
		injectC:         nil,
		obsvReqSendC:    nil,
		logger:          zap.L(),
		signedInC:       nil,
		governor:        nil,
		evmConnector:    connector,
		guardianSigner:  guardianSigner,
		guardianAddress: ethcrypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background())),
	}
}

func TestSignExistingVAA_NoVAA(t *testing.T) {
	s := setupAdminServerForVAASigning(0, []common.Address{})

	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 nil,
		NewGuardianAddrs:    nil,
		NewGuardianSetIndex: 0,
	})
	require.ErrorContains(t, err, "failed to unmarshal VAA")
}

func TestSignExistingVAA_NotGuardian(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is not a member of the new guardian set")
}

func TestSignExistingVAA_InvalidVAA(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers[:2], t)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "failed to verify existing VAA")
}

func TestSignExistingVAA_DuplicateGuardian(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "duplicate guardians in the guardian set")
}

func TestSignExistingVAA_AlreadyGuardian(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)
	s.evmConnector = mockEVMConnector{
		guardianAddrs:    append(gsAddrs, s.guardianAddress),
		guardianSetIndex: 0,
	}

	v := generateMockVAA(0, append(signers, s.guardianSigner), t)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is already on the old set")
}

func TestSignExistingVAA_NotAFutureGuardian(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is not a member of the new guardian set")
}

func TestSignExistingVAA_CantReachQuorum(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(append(gsAddrs, common.Address{0, 1}, common.Address{3, 1}, common.Address{8, 1})),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "cannot reach quorum on new guardian set with the local signature")
}

func TestSignExistingVAA_Valid(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	res, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})

	require.NoError(t, err)
	v2 := generateMockVAA(1, append(signers, s.guardianSigner), t)
	require.Equal(t, v2, res.Vaa)
}

func TestSignExistingVAA_ValidMutatedSet(t *testing.T) {
	signers, gsAddrs := generateGuardianSigners(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, signers, t)

	gsAddrs = append(gsAddrs[1:], s.guardianAddress) // We lose the first Guardian so the index changes for every Guardian
	res, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})

	require.NoError(t, err)
	v2 := generateMockVAA(1, append(signers[1:], s.guardianSigner), t)
	require.Equal(t, v2, res.Vaa)
}

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
		db:              nil,
		injectC:         nil,
		obsvReqSendC:    nil,
		logger:          nil,
		signedInC:       nil,
		governor:        gov,
		evmConnector:    nil,
		guardianSigner:  nil,
		guardianAddress: common.Address{},
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

// =================================================================================================
// DelegatedPauser SetConfigEvm / SetConfigSolana, BridgeSetPauserAddresses{Evm,Solana}
// See whitepapers/0018_pauser.md.
// =================================================================================================

func TestDelegatedPauserSetConfigEvmToVaa(t *testing.T) {
	req := &nodev1.DelegatedPauserSetConfigEvm{
		ChainId:        uint32(vaa.ChainIDEthereum),
		Index:          1,
		Threshold:      2,
		ExpiryDuration: 3600,
		Signers: []string{
			"1111111111111111111111111111111111111111",
			"2222222222222222222222222222222222222222",
			"3333333333333333333333333333333333333333",
		},
	}

	nonce := uint32(11)
	sequence := uint64(22)
	v, err := delegatedPauserSetConfigEvmToVaa(req, govTimestamp, govGuardianSetIndex, nonce, sequence)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, sequence, nonce)
	assert.NotEmpty(t, v.Payload)
}

func TestDelegatedPauserSetConfigEvmToVaa_Errors(t *testing.T) {
	base := func() *nodev1.DelegatedPauserSetConfigEvm {
		return &nodev1.DelegatedPauserSetConfigEvm{
			ChainId:        uint32(vaa.ChainIDEthereum),
			Index:          1,
			Threshold:      1,
			ExpiryDuration: 60,
			Signers:        []string{"1111111111111111111111111111111111111111"},
		}
	}

	tests := []struct {
		name    string
		mutate  func(r *nodev1.DelegatedPauserSetConfigEvm)
		errText string
	}{
		{"ChainIdZero", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.ChainId = 0 }, "convert chain id"},
		{"ChainIdTooLarge", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.ChainId = 1 << 17 }, "convert chain id"},
		{"ChainIdUnregistered", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.ChainId = 0xffff }, "convert chain id"},
		{"IndexZero", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Index = 0 }, "invalid index"},
		{"IndexTooLarge", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Index = 1 << 17 }, "invalid index"},
		{"ThresholdZero", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Threshold = 0 }, "invalid threshold"},
		{"ThresholdTooLarge", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Threshold = 1 << 9 }, "invalid threshold"},
		{"ExpiryZero", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.ExpiryDuration = 0 }, "invalid expiry_duration"},
		{"NoSigners", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Signers = nil }, "at least one signer"},
		{"ThresholdGreaterThanSigners", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Threshold = 5 }, "threshold exceeds number of signers"},
		{"BadSignerHex", func(r *nodev1.DelegatedPauserSetConfigEvm) { r.Signers = []string{"not-an-address"} }, "invalid EVM signer"},
		{"SignerWith0xPrefix", func(r *nodev1.DelegatedPauserSetConfigEvm) {
			r.Signers = []string{"0x1111111111111111111111111111111111111111"}
		}, "invalid EVM signer"},
		{"ZeroSigner", func(r *nodev1.DelegatedPauserSetConfigEvm) {
			r.Signers = []string{"0000000000000000000000000000000000000000"}
		}, "zero address"},
		{"DuplicateSigners", func(r *nodev1.DelegatedPauserSetConfigEvm) {
			r.Threshold = 1
			r.Signers = []string{
				"1111111111111111111111111111111111111111",
				"1111111111111111111111111111111111111111",
			}
		}, "duplicate"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := delegatedPauserSetConfigEvmToVaa(req, govTimestamp, govGuardianSetIndex, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestDelegatedPauserSetConfigSolanaToVaa(t *testing.T) {
	req := &nodev1.DelegatedPauserSetConfigSolana{
		ChainId:        uint32(vaa.ChainIDSolana),
		Index:          1,
		Threshold:      1,
		ExpiryDuration: 3600,
		Signers: []string{
			"0000000000000000000000000000000000000000000000000000000000000001",
			"0000000000000000000000000000000000000000000000000000000000000002",
		},
	}
	v, err := delegatedPauserSetConfigSolanaToVaa(req, govTimestamp, govGuardianSetIndex, 1, 1)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, 1, 1)
	assert.NotEmpty(t, v.Payload)
}

func TestDelegatedPauserSetConfigSolanaToVaa_Errors(t *testing.T) {
	base := func() *nodev1.DelegatedPauserSetConfigSolana {
		return &nodev1.DelegatedPauserSetConfigSolana{
			ChainId:        uint32(vaa.ChainIDSolana),
			Index:          1,
			Threshold:      1,
			ExpiryDuration: 60,
			Signers:        []string{"0000000000000000000000000000000000000000000000000000000000000001"},
		}
	}

	tests := []struct {
		name    string
		mutate  func(r *nodev1.DelegatedPauserSetConfigSolana)
		errText string
	}{
		{"ChainIdZero", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.ChainId = 0 }, "convert chain id"},
		{"ChainIdTooLarge", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.ChainId = 1 << 17 }, "convert chain id"},
		{"ChainIdUnregistered", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.ChainId = 0xffff }, "convert chain id"},
		{"IndexZero", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Index = 0 }, "invalid index"},
		{"ThresholdZero", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Threshold = 0 }, "invalid threshold"},
		{"ExpiryZero", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.ExpiryDuration = 0 }, "invalid expiry_duration"},
		{"NoSigners", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Signers = nil }, "at least one signer"},
		{"ThresholdGreaterThanSigners", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Threshold = 5 }, "threshold exceeds number of signers"},
		{"BadSignerHex", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Signers = []string{"zz"} }, "invalid Solana signer"},
		{"ShortSigner", func(r *nodev1.DelegatedPauserSetConfigSolana) { r.Signers = []string{"01"} }, "invalid Solana signer"},
		{"ZeroSigner", func(r *nodev1.DelegatedPauserSetConfigSolana) {
			r.Signers = []string{"0000000000000000000000000000000000000000000000000000000000000000"}
		}, "zero pubkey"},
		{"DuplicateSigners", func(r *nodev1.DelegatedPauserSetConfigSolana) {
			r.Signers = []string{
				"0000000000000000000000000000000000000000000000000000000000000001",
				"0000000000000000000000000000000000000000000000000000000000000001",
			}
		}, "duplicate"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := delegatedPauserSetConfigSolanaToVaa(req, govTimestamp, govGuardianSetIndex, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestBridgeSetPauserAddressesEvmToVaa(t *testing.T) {
	req := &nodev1.BridgeSetPauserAddressesEvm{
		Module:        "TokenBridge",
		TargetChainId: uint32(vaa.ChainIDEthereum),
		Pauser:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Unpauser:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	v, err := bridgeSetPauserAddressesEvmToVaa(req, govTimestamp, govGuardianSetIndex, 7, 7)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, 7, 7)
	assert.NotEmpty(t, v.Payload)
}

func TestBridgeSetPauserAddressesEvmToVaa_Errors(t *testing.T) {
	base := func() *nodev1.BridgeSetPauserAddressesEvm {
		return &nodev1.BridgeSetPauserAddressesEvm{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDEthereum),
			Pauser:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Unpauser:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}
	}

	tests := []struct {
		name    string
		mutate  func(r *nodev1.BridgeSetPauserAddressesEvm)
		errText string
	}{
		{"EmptyModule", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.Module = "" }, "module is required"},
		{"ChainIdZero", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.TargetChainId = 0 }, "convert chain id"},
		{"ChainIdTooLarge", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.TargetChainId = 1 << 17 }, "convert chain id"},
		{"ChainIdUnregistered", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.TargetChainId = 0xffff }, "convert chain id"},
		{"BadPauser", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.Pauser = "not-an-address" }, "invalid pauser"},
		{"PauserWith0xPrefix", func(r *nodev1.BridgeSetPauserAddressesEvm) {
			r.Pauser = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, "invalid pauser"},
		{"BadUnpauser", func(r *nodev1.BridgeSetPauserAddressesEvm) { r.Unpauser = "not-an-address" }, "invalid unpauser"},
		{"UnpauserWith0xPrefix", func(r *nodev1.BridgeSetPauserAddressesEvm) {
			r.Unpauser = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, "invalid unpauser"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := bridgeSetPauserAddressesEvmToVaa(req, govTimestamp, govGuardianSetIndex, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestBridgeSetPauserAddressesSolanaToVaa(t *testing.T) {
	req := &nodev1.BridgeSetPauserAddressesSolana{
		Module:        "TokenBridge",
		TargetChainId: uint32(vaa.ChainIDSolana),
		Pauser:        "000000000000000000000000000000000000000000000000000000000000000a",
		Unpauser:      "000000000000000000000000000000000000000000000000000000000000000b",
	}
	v, err := bridgeSetPauserAddressesSolanaToVaa(req, govTimestamp, govGuardianSetIndex, 8, 8)
	require.NoError(t, err)
	verifyGovernanceVAA(t, v, 8, 8)
	assert.NotEmpty(t, v.Payload)
}

func TestBridgeSetPauserAddressesSolanaToVaa_Errors(t *testing.T) {
	base := func() *nodev1.BridgeSetPauserAddressesSolana {
		return &nodev1.BridgeSetPauserAddressesSolana{
			Module:        "TokenBridge",
			TargetChainId: uint32(vaa.ChainIDSolana),
			Pauser:        "000000000000000000000000000000000000000000000000000000000000000a",
			Unpauser:      "000000000000000000000000000000000000000000000000000000000000000b",
		}
	}

	tests := []struct {
		name    string
		mutate  func(r *nodev1.BridgeSetPauserAddressesSolana)
		errText string
	}{
		{"EmptyModule", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.Module = "" }, "module is required"},
		{"ChainIdZero", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.TargetChainId = 0 }, "convert chain id"},
		{"ChainIdTooLarge", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.TargetChainId = 1 << 17 }, "convert chain id"},
		{"ChainIdUnregistered", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.TargetChainId = 0xffff }, "convert chain id"},
		{"BadPauserHex", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.Pauser = "zz" }, "invalid pauser"},
		{"ShortPauser", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.Pauser = "01" }, "invalid pauser"},
		{"BadUnpauserHex", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.Unpauser = "zz" }, "invalid unpauser"},
		{"ShortUnpauser", func(r *nodev1.BridgeSetPauserAddressesSolana) { r.Unpauser = "01" }, "invalid unpauser"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			tc.mutate(req)
			_, err := bridgeSetPauserAddressesSolanaToVaa(req, govTimestamp, govGuardianSetIndex, 0, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
