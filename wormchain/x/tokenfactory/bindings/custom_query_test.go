package bindings_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/app"
	bindings "github.com/wormhole-foundation/wormchain/x/tokenfactory/bindings/types"
)

/*func TestQueryFullDenom(t *testing.T) {
	actor := RandomAccountAddress()
	tokenz, ctx := SetupCustomApp(t, actor)

	reflect := instantiateReflectContract(t, ctx, tokenz, actor)
	require.NotEmpty(t, reflect)

	// query full denom
	query := bindings.TokenQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "ustart",
		},
	}
	resp := bindings.FullDenomResponse{}
	queryCustom(t, ctx, tokenz, reflect, query, &resp)

	expected := fmt.Sprintf("factory/%s/ustart", reflect.String())
	require.EqualValues(t, expected, resp.Denom)
}*/

type ReflectQuery struct {
	Chain *ChainRequest `json:"chain,omitempty"`
}

type ChainRequest struct {
	Request wasmvmtypes.QueryRequest `json:"request"`
}

type ChainResponse struct {
	Data []byte `json:"data"`
}

func queryCustom(t *testing.T, ctx sdk.Context, tokenz *app.App, contract sdk.AccAddress, request bindings.TokenQuery, response interface{}) {
	wrapped := bindings.TokenFactoryQuery{
		Token: &request,
	}
	msgBz, err := json.Marshal(wrapped)
	require.NoError(t, err)
	fmt.Println(string(msgBz))

	query := ReflectQuery{
		Chain: &ChainRequest{
			Request: wasmvmtypes.QueryRequest{Custom: msgBz},
		},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	fmt.Println(string(queryBz))

	resBz, err := tokenz.GetWasmKeeper().QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var resp ChainResponse
	err = json.Unmarshal(resBz, &resp)
	require.NoError(t, err)
	err = json.Unmarshal(resp.Data, response)
	require.NoError(t, err)
}
