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
	// This is deprecated and will be removed in a subsequent release
	// because it uses an error-prone verification interface.
	VerifyQuorum *verifyQuorumParams `json:"verify_quorum,omitempty"`

	// Verify the signatures on a VAA.
	// Successor to `VerifyQuorum` as the verification uses a safe interface.
	VerifyVaa *verifyVaaParams `json:"verify_vaa,omitempty"`

	// Verify the signatures on a message with a given message prefix.
	// The caller should take care not to allow outside sources to choose the prefix.
	VerifyMessageSignature *verifyMessageSignatureParams `json:"verify_message_signature,omitempty"`

	// Calculate the minimum number of participants required in quorum for the latest guardian set.
	CalculateQuorum *calculateQuorumParams `json:"calculate_quorum,omitempty"`
}

// deprecated
type verifyQuorumParams struct {
	Data             []byte           `json:"data"`
	GuardianSetIndex uint32           `json:"guardian_set_index"`
	Signatures       []*vaa.Signature `json:"signatures"`
}

type verifyVaaParams struct {
	Vaa []byte
}

type verifyMessageSignatureParams struct {
	Prefix           []byte         `json:"prefix"`
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
			// verify vaa using deprecated method
			err := keeper.DeprecatedVerifyVaa(ctx, wormholeQuery.VerifyQuorum.Data, wormholeQuery.VerifyQuorum.GuardianSetIndex, wormholeQuery.VerifyQuorum.Signatures)
			if err != nil {
				return nil, err
			}
			return []byte("{}"), nil
		}
		if wormholeQuery.VerifyVaa != nil {
			// verify vaa using recommended method
			v, err := vaa.Unmarshal(wormholeQuery.VerifyVaa.Vaa)
			if err != nil {
				return nil, err
			}
			err = keeper.VerifyVAA(ctx, v)
			if err != nil {
				return nil, err
			}
			return []byte("{}"), nil
		}
		if wormholeQuery.VerifyMessageSignature != nil {
			// handle the verify message signature query
			err := keeper.VerifyMessageSignature(
				ctx,
				wormholeQuery.VerifyMessageSignature.Prefix,
				wormholeQuery.VerifyMessageSignature.Data,
				wormholeQuery.VerifyMessageSignature.GuardianSetIndex,
				wormholeQuery.VerifyMessageSignature.Signature,
			)
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
