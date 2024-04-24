//nolint:unparam
package adminrpc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

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

func generateGS(num int) (keys []*ecdsa.PrivateKey, addrs []common.Address) {
	for i := 0; i < num; i++ {
		key, err := ethcrypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		keys = append(keys, key)
		addrs = append(addrs, ethcrypto.PubkeyToAddress(key.PublicKey))
	}
	return
}

func addrsToHexStrings(addrs []common.Address) (out []string) {
	for _, addr := range addrs {
		out = append(out, addr.String())
	}
	return
}

func generateMockVAA(gsIndex uint32, gsKeys []*ecdsa.PrivateKey) []byte {
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
	for i, key := range gsKeys {
		v.AddSignature(key, uint8(i))
	}

	vBytes, err := v.Marshal()
	if err != nil {
		panic(err)
	}
	return vBytes
}

func setupAdminServerForVAASigning(gsIndex uint32, gsAddrs []common.Address) *nodePrivilegedService {
	gk, err := ethcrypto.GenerateKey()
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
		gk:              gk,
		guardianAddress: ethcrypto.PubkeyToAddress(gk.PublicKey),
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
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys)

	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is not a member of the new guardian set")
}

func TestSignExistingVAA_InvalidVAA(t *testing.T) {
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys[:2])

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "failed to verify existing VAA")
}

func TestSignExistingVAA_DuplicateGuardian(t *testing.T) {
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys)

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
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)
	s.evmConnector = mockEVMConnector{
		guardianAddrs:    append(gsAddrs, s.guardianAddress),
		guardianSetIndex: 0,
	}

	v := generateMockVAA(0, append(gsKeys, s.gk))

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is already on the old set")
}

func TestSignExistingVAA_NotAFutureGuardian(t *testing.T) {
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys)

	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "local guardian is not a member of the new guardian set")
}

func TestSignExistingVAA_CantReachQuorum(t *testing.T) {
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	_, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(append(gsAddrs, common.Address{0, 1}, common.Address{3, 1}, common.Address{8, 1})),
		NewGuardianSetIndex: 1,
	})
	require.ErrorContains(t, err, "cannot reach quorum on new guardian set with the local signature")
}

func TestSignExistingVAA_Valid(t *testing.T) {
	gsKeys, gsAddrs := generateGS(5)
	s := setupAdminServerForVAASigning(0, gsAddrs)

	v := generateMockVAA(0, gsKeys)

	gsAddrs = append(gsAddrs, s.guardianAddress)
	res, err := s.SignExistingVAA(context.Background(), &nodev1.SignExistingVAARequest{
		Vaa:                 v,
		NewGuardianAddrs:    addrsToHexStrings(gsAddrs),
		NewGuardianSetIndex: 1,
	})

	require.NoError(t, err)
	v2 := generateMockVAA(1, append(gsKeys, s.gk))
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
