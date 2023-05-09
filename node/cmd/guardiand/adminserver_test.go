//nolint:unparam
package guardiand

import (
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
	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
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
