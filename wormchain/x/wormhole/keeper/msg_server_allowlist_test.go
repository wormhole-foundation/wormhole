package keeper_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/ante"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var _ sdk.Tx = &MockTx{}

type MockTx struct {
	Msgs []sdk.Msg
}

func (tx *MockTx) GetMsgs() []sdk.Msg {
	return tx.Msgs
}

func (tx *MockTx) ValidateBasic() error {
	return nil
}
func MockNext(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
	return ctx, nil
}

func getSigner(guardianValidator *types.GuardianValidator) string {
	return sdk.AccAddress(guardianValidator.ValidatorAddr).String()
}

func getMsgWithSigner(signer string) sdk.Msg {
	// Use any msg, picking on MsgExecuteGovernanceVAA arbitrarily.
	return &types.MsgExecuteGovernanceVAA{
		Signer: signer,
	}
}

func getTxWithSigner(signer string) sdk.Tx {
	return &MockTx{
		Msgs: []sdk.Msg{getMsgWithSigner(signer)},
	}
}

func getRandomAddress() string {
	privKeyValidator, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	validatorAddr := crypto.PubkeyToAddress(privKeyValidator.PublicKey)
	return sdk.AccAddress(validatorAddr[:]).String()
}

func TestAllowlist(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, _ := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})

	createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{
		Index: 0,
	})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// Test creating allowlist works using a validator
	new_address := getRandomAddress()
	_, err := msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)
	// Test creating the same address again is rejected
	for _, g := range guardians {
		_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
			Signer:  getSigner(&g),
			Address: new_address,
		})
		assert.Error(t, err)
	}

	// Test address can be Deleted
	_, err = msgServer.DeleteAllowlist(context, &types.MsgDeleteAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)
	// Can't be deleted again since it doesn't exist
	_, err = msgServer.DeleteAllowlist(context, &types.MsgDeleteAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.Error(t, err)
	// Can be added again
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)

	// another guardian cannot delete an allowlist they did not create
	for _, g := range guardians[1:] {
		_, err = msgServer.DeleteAllowlist(context, &types.MsgDeleteAllowlistRequest{
			Signer:  getSigner(&g),
			Address: new_address,
		})
		assert.Error(t, err)
	}

	// Cannot make allowlist if not a validator
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getRandomAddress(),
		Address: getRandomAddress(),
	})
	assert.Error(t, err)

	// Cannot make allowlist if the guardian set changes
	oldGuardian := guardians[0]
	guardians, _ = createNGuardianValidator(k, ctx, 10)
	createNewGuardianSet(k, ctx, guardians)
	err = k.TrySwitchToNewConsensusGuardianSet(ctx)
	assert.NoError(t, err)
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&oldGuardian),
		Address: getRandomAddress(),
	})
	assert.Error(t, err)

	// still works with new guardian set
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: getRandomAddress(),
	})
	assert.NoError(t, err)

	// Anyone can remove stale allowlists
	// (new_address list is now stale as it's validator is no longer in validator set)
	_, err = msgServer.DeleteAllowlist(context, &types.MsgDeleteAllowlistRequest{
		Signer:  getSigner(&guardians[9]),
		Address: new_address,
	})
	assert.NoError(t, err)

	_ = msgServer
	_ = context
}

func TestAllowlistAnteHandler(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	_ = privateKeys
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})

	createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{
		Index: 0,
	})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	anteHandler := ante.NewWormholeAllowlistDecorator(*k)

	// Test ante handler works with validate validator address
	for _, g := range guardians {
		msgs := []sdk.Msg{}
		for i := 0; i < 5; i += 1 {
			msgs = append(msgs, getMsgWithSigner(getSigner(&g)))
		}
		tx := MockTx{
			Msgs: msgs,
		}
		_, err := anteHandler.AnteHandle(ctx, &tx, false, MockNext)
		assert.NoError(t, err)
	}

	// Test ante handler rejects new address
	new_address := getRandomAddress()
	_, err := anteHandler.AnteHandle(ctx, getTxWithSigner(new_address), false, MockNext)
	assert.Error(t, err)

	// Test ante handler accepts new address when whitelisted
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)
	_, err = anteHandler.AnteHandle(ctx, getTxWithSigner(new_address), false, MockNext)
	assert.NoError(t, err)

	// Test ante handler rejects when allowlist is removed
	_, err = msgServer.DeleteAllowlist(context, &types.MsgDeleteAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)

	_, err = anteHandler.AnteHandle(ctx, getTxWithSigner(new_address), false, MockNext)
	assert.Error(t, err)

	// (add back the allowlist)
	_, err = msgServer.CreateAllowlist(context, &types.MsgCreateAllowlistRequest{
		Signer:  getSigner(&guardians[0]),
		Address: new_address,
	})
	assert.NoError(t, err)
	_, err = anteHandler.AnteHandle(ctx, getTxWithSigner(new_address), false, MockNext)
	assert.NoError(t, err)

	// test that the ante handler rejects when a 2nd not-allowed msg is snuck in >:)
	_, err = anteHandler.AnteHandle(ctx, &MockTx{
		Msgs: []sdk.Msg{
			// good
			getMsgWithSigner(new_address),
			// bad
			getMsgWithSigner(getRandomAddress()),
		},
	}, false, MockNext)
	assert.Error(t, err)

	// test ante handler rejects address that is no longer valid
	// due to validator set advancing
	// 1. new guardian set
	guardians, _ = createNGuardianValidator(k, ctx, 10)
	createNewGuardianSet(k, ctx, guardians)
	err = k.TrySwitchToNewConsensusGuardianSet(ctx)
	assert.NoError(t, err)
	// 2. expect reject
	_, err = anteHandler.AnteHandle(ctx, getTxWithSigner(new_address), false, MockNext)
	assert.Error(t, err)
}
