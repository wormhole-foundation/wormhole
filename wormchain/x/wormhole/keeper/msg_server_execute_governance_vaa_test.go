package keeper_test

import (
	"crypto/ecdsa"
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createExecuteGovernanceVaaPayload(k *keeper.Keeper, ctx sdk.Context, num_guardians byte) ([]byte, []*ecdsa.PrivateKey) {
	guardians, privateKeys := createNGuardianValidator(k, ctx, int(num_guardians))
	next_index := k.GetGuardianSetCount(ctx)
	set_update := make([]byte, 4)
	binary.BigEndian.PutUint32(set_update, next_index)
	set_update = append(set_update, num_guardians)
	// Add keys to set_update
	for _, guardian := range guardians {
		set_update = append(set_update, guardian.GuardianKey...)
	}
	// governance message with sha3 of wasmBytes as the payload
	module := [32]byte{}
	copy(module[:], vaa.CoreModule)
	gov_msg := types.NewGovernanceMessage(module, byte(vaa.ActionGuardianSetUpdate), uint16(vaa.ChainIDWormchain), set_update)

	return gov_msg.MarshalBinary(), privateKeys
}

func TestExecuteGovernanceVAA(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	_ = privateKeys
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to update guardian set with extra guardian
	payload, newPrivateKeys := createExecuteGovernanceVaaPayload(k, ctx, 11)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ := v.Marshal()
	_, err := msgServer.ExecuteGovernanceVAA(context, &types.MsgExecuteGovernanceVAA{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.NoError(t, err)

	// we should have a new set with 11 guardians now
	new_index := k.GetLatestGuardianSetIndex(ctx)
	assert.Equal(t, set.Index+1, new_index)
	new_set, _ := k.GetGuardianSet(ctx, new_index)
	assert.Len(t, new_set.Keys, 11)

	// Submitting another change with the old set doesn't work
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.ExecuteGovernanceVAA(context, &types.MsgExecuteGovernanceVAA{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrGuardianSetNotSequential)

	// Invalid length
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload[:len(payload)-1])
	vBz, _ = v.Marshal()
	_, err = msgServer.ExecuteGovernanceVAA(context, &types.MsgExecuteGovernanceVAA{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidGovernancePayloadLength)

	// Include a guardian address twice in an update
	payload_bad, _ := createExecuteGovernanceVaaPayload(k, ctx, 11)
	copy(payload_bad[len(payload_bad)-20:], payload_bad[len(payload_bad)-40:len(payload_bad)-20])
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload_bad)
	vBz, _ = v.Marshal()
	_, err = msgServer.ExecuteGovernanceVAA(context, &types.MsgExecuteGovernanceVAA{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrDuplicateGuardianAddress)

	// Change set again with new set update
	payload, _ = createExecuteGovernanceVaaPayload(k, ctx, 12)
	v = generateVaa(new_set.Index, newPrivateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.ExecuteGovernanceVAA(context, &types.MsgExecuteGovernanceVAA{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.NoError(t, err)
	new_index2 := k.GetLatestGuardianSetIndex(ctx)
	assert.Equal(t, new_set.Index+1, new_index2)
}
