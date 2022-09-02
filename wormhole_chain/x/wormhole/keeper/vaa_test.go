package keeper_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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

func getVaa() vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	return vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(0),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}
}

func TestVerifyVAA(t *testing.T) {

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
			vaa := getVaa()

			for i, key := range tc.signers {
				vaa.AddSignature(key, uint8(i))
			}

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
