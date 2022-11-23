package keeper

import (
	"encoding/json"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func NewCustomQueryHandler(keeper Keeper) *wasmkeeper.QueryPlugins {
	return &wasmkeeper.QueryPlugins{
		Custom: WormholeQuerier(keeper),
	}
}

type WormholeQuery struct {
	VerifyQuorum    *verifyQuorumParams    `json:"verify_quorum,omitempty"`
	VerifySignature *verifySignatureParams `json:"verify_signature,omitempty"`
	CalculateQuorum *calculateQuorumParams `json:"calculate_quorum,omitempty"`
}

type verifyQuorumParams struct {
	Data             []byte           `json:"data"`
	GuardianSetIndex uint32           `json:"guardian_set_index"`
	Signatures       []*vaa.Signature `json:"signatures"`
}

type verifySignatureParams struct {
	Data             []byte         `json:"data"`
	GuardianSetIndex uint32         `json:"guardian_set_index"`
	Signature        *vaa.Signature `json:"signature"`
}

type calculateQuorumParams struct {
	GuardianSetIndex uint32 `json:"guardian_set_index"`
}

func WormholeQuerier(keeper Keeper) func(ctx sdk.Context, data json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, data json.RawMessage) ([]byte, error) {
		var wormholeQuery WormholeQuery
		err := json.Unmarshal(data, &wormholeQuery)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
		}

		if wormholeQuery.VerifyQuorum != nil {
			// handle the verify quorum query
			digest := vaa.SigningMsg(wormholeQuery.VerifyQuorum.Data).Bytes()
			err := keeper.VerifyQuorum(ctx, digest, wormholeQuery.VerifyQuorum.GuardianSetIndex, wormholeQuery.VerifyQuorum.Signatures)
			if err != nil {
				return nil, err
			}
			return []byte("{}"), nil
		}
		if wormholeQuery.VerifySignature != nil {
			// handle the verify signature query
			digest := vaa.SigningMsg(wormholeQuery.VerifySignature.Data).Bytes()
			err := keeper.VerifySignature(ctx, digest, wormholeQuery.VerifySignature.GuardianSetIndex, wormholeQuery.VerifySignature.Signature)
			if err != nil {
				return nil, err
			}
			return []byte("{}"), nil
		}
		if wormholeQuery.CalculateQuorum != nil {
			// handle the calculate quorum query
			quorum, _, err := keeper.CalculateQuorum(ctx, wormholeQuery.CalculateQuorum.GuardianSetIndex)
			if err != nil {
				return nil, err
			}

			return json.Marshal(quorum)
		}

		// else we have an unrecognized request
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "custom"}
	}
}
