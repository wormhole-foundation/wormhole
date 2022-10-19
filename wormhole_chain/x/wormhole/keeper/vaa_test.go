package keeper_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	keepertest "github.com/wormhole-foundation/wormhole-chain/testutil/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestCalculateQuorum(t *testing.T) {

	tests := []struct {
		guardians int
		quorum    int
	}{
		{guardians: 0, quorum: 1},
		{guardians: 1, quorum: 1},
		{guardians: 2, quorum: 2},
		{guardians: 3, quorum: 3},
		{guardians: 4, quorum: 3},
		{guardians: 5, quorum: 4},
		{guardians: 6, quorum: 5},
		{guardians: 7, quorum: 5},
		{guardians: 8, quorum: 6},
		{guardians: 9, quorum: 7},
		{guardians: 10, quorum: 7},
		{guardians: 19, quorum: 13},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v", tc.guardians), func(t *testing.T) {
			quorum := keeper.CalculateQuorum(tc.guardians)
			assert.Equal(t, tc.quorum, quorum)
		})
	}
}

var lastestSequence = 1

func generateVaa(index uint32, signers []*ecdsa.PrivateKey, emitterChain vaa.ChainID, payload []byte) vaa.VAA {
	v := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: index,
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(lastestSequence),
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   vaa.Address(vaa.GovernanceEmitter),
		Payload:          payload,
	}
	lastestSequence = lastestSequence + 1
	for i, key := range signers {
		v.AddSignature(key, uint8(i))
	}
	return v
}
func resignVaa(v vaa.VAA, signers []*ecdsa.PrivateKey) vaa.VAA {
	v.Signatures = []*vaa.Signature{}
	for i, key := range signers {
		v.AddSignature(key, uint8(i))
	}
	return v
}

func TestVerifyVAA(t *testing.T) {

	payload := []byte{97, 97, 97, 97, 97, 97}
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())

	tests := []struct {
		label       string
		guardianSet types.GuardianSet
		signers     []*ecdsa.PrivateKey
		willError   bool
	}{

		{label: "ValidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signers:     []*ecdsa.PrivateKey{privKey1},
			willError:   false},
		{label: "InvalidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signers:     []*ecdsa.PrivateKey{privKey2},
			willError:   true},
		{label: "InvalidGuardianSetIndex",
			guardianSet: types.GuardianSet{Index: 1, Keys: addrsBytes, ExpirationTime: 0},
			signers:     []*ecdsa.PrivateKey{privKey1},
			willError:   true},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			keeper, ctx := keepertest.WormholeKeeper(t)
			vaa := generateVaa(tc.guardianSet.Index, tc.signers, vaa.ChainIDSolana, payload)

			keeper.AppendGuardianSet(ctx, tc.guardianSet)
			err := keeper.VerifyVAA(ctx, &vaa)

			if tc.willError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestVerifyVAA2(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(keeper, ctx, 25)
	set := createNewGuardianSet(keeper, ctx, guardians)

	// check verify works
	payload := []byte{97, 97, 97, 97, 97, 97}
	v := generateVaa(set.Index, privateKeys, vaa.ChainIDSolana, payload)
	err := keeper.VerifyVAA(ctx, &v)
	assert.NoError(t, err)

	// flip a bit in one of the signatures
	v = generateVaa(set.Index, privateKeys, vaa.ChainIDSolana, payload)
	v.Signatures[20].Signature[1] = v.Signatures[20].Signature[1] ^ 0x40
	err = keeper.VerifyVAA(ctx, &v)
	assert.Error(t, err)

	// generate for a non existing guardian set
	v = generateVaa(set.Index+1, privateKeys, vaa.ChainIDSolana, payload)
	err = keeper.VerifyVAA(ctx, &v)
	assert.Error(t, err)
}

func TestVerifyVAAGovernance(t *testing.T) {
	keeper, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(keeper, ctx, 25)
	set := createNewGuardianSet(keeper, ctx, guardians)
	config := types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	}
	keeper.SetConfig(ctx, config)

	action := byte(0x12)
	our_module := [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 01}
	payload := []byte{}
	// governance payload is {module_id, action, chain, payload}
	payload = append(payload, our_module[:]...)
	payload = append(payload, action)
	chain_bz := [2]byte{}
	binary.BigEndian.PutUint16(chain_bz[:], uint16(vaa.ChainIDWormchain))
	payload = append(payload, chain_bz[:]...)
	// custom payload
	custom_payload := []byte{1, 2, 3, 4, 5}
	payload = append(payload, custom_payload...)

	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	err := keeper.VerifyVAA(ctx, &v)
	assert.NoError(t, err)
	parsed_action, parsed_payload, err := keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.NoError(t, err)
	assert.Equal(t, action, parsed_action)
	assert.Equal(t, custom_payload, parsed_payload)

	// verifying a second time will return error because of replay protection
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.ErrorIs(t, err, types.ErrVAAAlreadyExecuted)

	// Expect error if module-id is different
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	bad_module := [32]byte{}
	bad_module[31] = 0xff
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, bad_module)
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)

	// Expect error if we're not using the right governance emitter address
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	v.EmitterAddress[5] = 0xff
	v = resignVaa(v, privateKeys)
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.ErrorIs(t, err, types.ErrInvalidGovernanceEmitter)

	// Expect error if we're not using the right governance emitter chain
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	v.EmitterChain = vaa.ChainIDEthereum
	v = resignVaa(v, privateKeys)
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.ErrorIs(t, err, types.ErrInvalidGovernanceEmitter)

	// Expect error if we're using a small payload
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload[:34])
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.ErrorIs(t, err, types.ErrGovernanceHeaderTooShort)

	// Expect error if we're using a different target chain
	payload[33] = 0xff
	payload[34] = 0xff
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	_, _, err = keeper.VerifyGovernanceVAA(ctx, &v, our_module)
	assert.ErrorIs(t, err, types.ErrInvalidGovernanceTargetChain)
}
