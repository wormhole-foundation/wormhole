package ante_test

import (
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/app/apptesting"
	"github.com/wormhole-foundation/wormchain/x/wormhole/ante"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func TestAnteHandle(t *testing.T) {

	// Setup app & ctx
	app := apptesting.Setup(t, false, 0)

	ctx := app.BaseApp.NewContext(false, tmtypes.Header{
		Height:  1,
		ChainID: apptesting.SimAppChainID,
		Time:    time.Now().UTC(),
	})

	// Create the decorator
	decorator := ante.NewWormholeWasmdDecorator(app.WormholeKeeper, *app.GetWasmKeeper())

	// Register a contract on the allowlist
	app.WormholeKeeper.SetWasmInstantiateAllowlist(ctx, types.WasmInstantiateAllowedContractCodeId{
		ContractAddress: "contract",
		CodeId:          1,
	})

	for _, tc := range []struct {
		name      string
		msg       sdk.Msg
		shouldErr bool
	}{
		{
			"MsgInstantiateContract - Valid Sender & Code ID",
			&wasmtypes.MsgInstantiateContract{
				Sender: "contract",
				CodeID: 1,
			},
			false,
		},
		{
			"MsgInstantiateContract - Sender Not On Allowlist",
			&wasmtypes.MsgInstantiateContract{
				Sender: "contract_invalid",
				CodeID: 1,
			},
			true,
		},
		{
			"MsgInstantiateContract - Code ID Not On Allowlist",
			&wasmtypes.MsgInstantiateContract{
				Sender: "contract",
				CodeID: 2,
			},
			true,
		},
		{
			"MsgInstantiateContract2 - Valid Sender & Code ID",
			&wasmtypes.MsgInstantiateContract2{
				Sender: "contract",
				CodeID: 1,
			},
			false,
		},
		{
			"MsgInstantiateContract - Sender Not On Allowlist",
			&wasmtypes.MsgInstantiateContract2{
				Sender: "contract_invalid",
				CodeID: 1,
			},
			true,
		},
		{
			"MsgInstantiateContract - Code ID Not On Allowlist",
			&wasmtypes.MsgInstantiateContract2{
				Sender: "contract",
				CodeID: 2,
			},
			true,
		},
		{
			"MsgStoreCode",
			&wasmtypes.MsgStoreCode{},
			true,
		},
		{
			"MsgMigrateContract",
			&wasmtypes.MsgMigrateContract{},
			true,
		},
		{
			"MsgUpdateAdmin",
			&wasmtypes.MsgUpdateAdmin{},
			true,
		},
		{
			"MsgClearAdmin",
			&wasmtypes.MsgClearAdmin{},
			true,
		},
		{
			"MsgUpdateInstantiateConfig",
			&wasmtypes.MsgUpdateInstantiateConfig{},
			true,
		},
		{
			"MsgUpdateParams",
			&wasmtypes.MsgUpdateParams{},
			true,
		},
		{
			"MsgPinCodes",
			&wasmtypes.MsgPinCodes{},
			true,
		},
		{
			"MsgUnpinCodes",
			&wasmtypes.MsgUnpinCodes{},
			true,
		},
		{
			"MsgSudoContract",
			&wasmtypes.MsgSudoContract{},
			true,
		},
		{
			"MsgStoreAndInstantiateContract",
			&wasmtypes.MsgStoreAndInstantiateContract{},
			true,
		},
		{
			"MsgAddCodeUploadParamsAddresses",
			&wasmtypes.MsgAddCodeUploadParamsAddresses{},
			true,
		},
		{
			"MsgRemoveCodeUploadParamsAddresses",
			&wasmtypes.MsgRemoveCodeUploadParamsAddresses{},
			true,
		},
		{
			"MsgStoreAndMigrateContract",
			&wasmtypes.MsgStoreAndMigrateContract{},
			true,
		},
		{
			"MsgUpdateContractLabel",
			&wasmtypes.MsgUpdateContractLabel{},
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Handle the tx
			_, err := decorator.AnteHandle(ctx, NewMockTx(tc.msg), false, EmptyAnte)

			// Check the result
			if tc.shouldErr {
				require.ErrorIs(t, err, ante.ErrNotSupported())
			} else {
				require.Nil(t, err)
			}
		})

	}
}
