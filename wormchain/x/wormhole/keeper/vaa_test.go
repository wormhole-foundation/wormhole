package keeper_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
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

func TestKeeperCalculateQuorum(t *testing.T) {
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())

	tests := []struct {
		label            string
		guardianSets     []types.GuardianSet
		guardianSetIndex uint32
		quorum           int
		willError        bool
		err              error
	}{

		{label: "HappyPath",
			guardianSets:     []types.GuardianSet{{Index: 0, Keys: addrsBytes, ExpirationTime: 0}},
			guardianSetIndex: 0,
			quorum:           1,
			willError:        false},
		{label: "GuardianSetNotFound",
			guardianSets:     []types.GuardianSet{{Index: 0, Keys: addrsBytes, ExpirationTime: 0}},
			guardianSetIndex: 1,
			willError:        true,
			err:              types.ErrGuardianSetNotFound},
		{label: "GuardianSetExpired",
			guardianSets: []types.GuardianSet{
				{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
				{Index: 1, Keys: addrsBytes, ExpirationTime: 0},
			},
			guardianSetIndex: 0,
			willError:        true,
			err:              types.ErrGuardianSetExpired},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			keeper, ctx := keepertest.WormholeKeeper(t)
			for _, gs := range tc.guardianSets {
				keeper.AppendGuardianSet(ctx, gs)
			}
			quorum, _, err := keeper.CalculateQuorum(ctx, tc.guardianSetIndex)

			if tc.willError == true {
				assert.NotNil(t, err)
				assert.Equal(t, err, tc.err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, quorum, tc.quorum)
			}
		})
	}
}

func sign(data common.Hash, key *ecdsa.PrivateKey, index uint8) *vaa.Signature {
	sig, err := crypto.Sign(data.Bytes(), key)
	if err != nil {
		panic(err)
	}
	sigData := [65]byte{}
	copy(sigData[:], sig)

	return &vaa.Signature{
		Index:     index,
		Signature: sigData,
	}
}

func TestVerifyMessageSignature(t *testing.T) {
	prefix := [32]byte{}
	payload := []byte{97, 97, 97, 97, 97, 97}
	digest, err := vaa.MessageSigningDigest(prefix[:], payload)
	require.NoError(t, err)
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())

	tests := []struct {
		label       string
		guardianSet types.GuardianSet
		signer      *ecdsa.PrivateKey
		setSigIndex bool
		sigIndex    uint8
		willError   bool
		err         error
	}{

		{label: "ValidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey1,
			willError:   false},
		{label: "IndexOutOfBounds",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey1,
			setSigIndex: true,
			sigIndex:    1,
			willError:   true,
			err:         types.ErrGuardianIndexOutOfBounds},
		{label: "InvalidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey2,
			willError:   true,
			err:         types.ErrSignaturesInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			keeper, ctx := keepertest.WormholeKeeper(t)
			keeper.AppendGuardianSet(ctx, tc.guardianSet)

			// build the signature
			signature := sign(digest, tc.signer, 0)
			if tc.setSigIndex {
				signature.Index = tc.sigIndex
			}

			// verify the signature
			err := keeper.VerifyMessageSignature(ctx, prefix[:], payload, tc.guardianSet.Index, signature)

			if tc.willError == true {
				assert.NotNil(t, err)
				assert.Equal(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyVaaSignature(t *testing.T) {
	v := &vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Now(),
		Nonce:            1,
		Sequence:         1,
		Payload:          []byte{97, 97, 97, 97, 97, 97},
	}
	digest := v.SigningDigest()
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	addr1 := crypto.PubkeyToAddress(privKey1.PublicKey)

	addrsBytes := [][]byte{}
	addrsBytes = append(addrsBytes, addr1.Bytes())

	tests := []struct {
		label       string
		guardianSet types.GuardianSet
		signer      *ecdsa.PrivateKey
		setSigIndex bool
		sigIndex    uint8
		willError   bool
		err         error
	}{

		{label: "ValidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey1,
			willError:   false},
		{label: "IndexOutOfBounds",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey1,
			setSigIndex: true,
			sigIndex:    1,
			willError:   true,
			// this out of bounds issue will trigger invalid signature from sdk.
			err: types.ErrSignaturesInvalid},
		{label: "InvalidSigner",
			guardianSet: types.GuardianSet{Index: 0, Keys: addrsBytes, ExpirationTime: 0},
			signer:      privKey2,
			willError:   true,
			err:         types.ErrSignaturesInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			keeper, ctx := keepertest.WormholeKeeper(t)
			keeper.AppendGuardianSet(ctx, tc.guardianSet)

			// build the signature
			signature := sign(digest, tc.signer, 0)
			if tc.setSigIndex {
				signature.Index = tc.sigIndex
			}
			v.Signatures = append(v.Signatures, signature)

			// verify the signature
			err := keeper.VerifyVAA(ctx, v)

			if tc.willError == true {
				assert.NotNil(t, err)
				assert.Equal(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var lastestSequence = 1

func signVaa(vaaToSign vaa.VAA, signers []*ecdsa.PrivateKey) vaa.VAA {
	for i, key := range signers {
		vaaToSign.AddSignature(key, uint8(i))
	}
	return vaaToSign
}
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
	return signVaa(v, signers)
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
