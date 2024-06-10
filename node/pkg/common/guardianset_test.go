package common

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestNewGuardianSet(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"),
		common.HexToAddress("0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"),
		common.HexToAddress("0x58076F561CC62A47087B567C86f986426dFCD000"),
		common.HexToAddress("0xBd6e9833490F8fA87c733A183CD076a6cBD29074"),
		common.HexToAddress("0xb853FCF0a5C78C1b56D15fCE7a154e6ebe9ED7a2"),
		common.HexToAddress("0xAF3503dBD2E37518ab04D7CE78b630F98b15b78a"),
		common.HexToAddress("0x785632deA5609064803B1c8EA8bB2c77a6004Bd1"),
		common.HexToAddress("0x09a281a698C0F5BA31f158585B41F4f33659e54D"),
		common.HexToAddress("0x3178443AB76a60E21690DBfB17f7F59F09Ae3Ea1"),
		common.HexToAddress("0x647ec26ae49b14060660504f4DA1c2059E1C5Ab6"),
		common.HexToAddress("0x810AC3D8E1258Bd2F004a94Ca0cd4c68Fc1C0611"),
		common.HexToAddress("0x80610e96d645b12f47ae5cf4546b18538739e90F"),
		common.HexToAddress("0x2edb0D8530E31A218E72B9480202AcBaeB06178d"),
		common.HexToAddress("0xa78858e5e5c4705CdD4B668FFe3Be5bae4867c9D"),
		common.HexToAddress("0x5Efe3A05Efc62D60e1D19fAeB56A80223CDd3472"),
		common.HexToAddress("0xD791b7D32C05aBB1cc00b6381FA0c4928f0c56fC"),
		common.HexToAddress("0x14Bc029B8809069093D712A3fd4DfAb31963597e"),
		common.HexToAddress("0x246Ab29FC6EBeDf2D392a51ab2Dc5C59d0902A03"),
		common.HexToAddress("0x132A84dFD920b35a3D0BA5f7A0635dF298F9033e"),
	}
	gs := NewGuardianSet(keys, 1)
	assert.True(t, reflect.DeepEqual(keys, gs.Keys))
	assert.Equal(t, uint32(1), gs.Index)
	assert.Equal(t, vaa.CalculateQuorum(len(keys)), gs.Quorum())
}

func TestKeyIndex(t *testing.T) {
	type test struct {
		guardianSet GuardianSet
		address     string
		result      bool
		keyIndex    int
	}

	guardianSet := *NewGuardianSet(
		[]common.Address{
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
		},
		1,
	)

	tests := []test{
		{guardianSet: guardianSet, address: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", result: true, keyIndex: 0},
		{guardianSet: guardianSet, address: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee", result: true, keyIndex: 1},
		{guardianSet: guardianSet, address: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaef", result: false, keyIndex: -1},
	}

	for _, testCase := range tests {
		t.Run(testCase.address, func(t *testing.T) {
			gs := testCase.guardianSet
			keyIndex, result := gs.KeyIndex(common.HexToAddress(testCase.address))
			assert.Equal(t, result, testCase.result)
			assert.Equal(t, keyIndex, testCase.keyIndex)
		})
	}
}

func TestKeysAsHexStrings(t *testing.T) {
	gs := GuardianSet{
		Keys: []common.Address{
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
		},
		Index: 1,
	}

	keyStrings := make([]string, len(gs.Keys))
	// Note that go-ethereum common.HexToAddress() will force a supplied all-lower
	// address to mixedcase for a valid checksum, which is why these are different
	keyStrings[0] = "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
	keyStrings[1] = "0x5Aaeb6053f3e94c9B9a09F33669435e7EF1BeAee"
	assert.Equal(t, keyStrings, gs.KeysAsHexStrings())
}

func TestNewGuardianSetState(t *testing.T) {
	gss := NewGuardianSetState(nil)
	assert.NotNil(t, gss)
	assert.Nil(t, gss.current)
	assert.Nil(t, gss.Get())
}

func TestSet(t *testing.T) {
	var gs GuardianSet = GuardianSet{
		Keys: []common.Address{
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
		},
		Index: 1,
	}

	gss := NewGuardianSetState(nil)
	assert.Nil(t, gss.current)
	gss.Set(&gs)
	assert.Equal(t, gss.current, &gs)
}

func TestGet(t *testing.T) {
	var gs GuardianSet = GuardianSet{
		Keys: []common.Address{
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
		},
		Index: 1,
	}

	gss := NewGuardianSetState(nil)
	assert.Nil(t, gss.Get())
	gss.Set(&gs)
	assert.Equal(t, gss.Get(), &gs)
}
