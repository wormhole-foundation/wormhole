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
			vaa, err := GovMsgToVaa(govMsg, govGuardianSetIndex, govTimestamp)
			if tst.errText == "" {
				require.NoError(t, err)
				verifyGovernanceVAA(t, vaa, govMsg.Sequence, govMsg.Nonce)
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
	boundsCheckFailure := `the specified number of days falls outside the range of 1 to 7`
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
