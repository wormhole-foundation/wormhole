package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

// TestGenesis ensures genesis state can be initialiazed and exported correctly.
func TestGenesis(t *testing.T) {
	for _, tc := range []struct {
		dataInFlight map[string][]byte
	}{
		{
			dataInFlight: map[string][]byte{},
		},
		{
			dataInFlight: map[string][]byte{
				"key1": []byte("value1"),
			},
		},
		{
			dataInFlight: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
				"key3": []byte("value3"),
			},
		},
	} {
		genesisState := types.GenesisState{
			TransposedDataInFlight: tc.dataInFlight,
		}

		app, ctx := keepertest.SetupWormchainAndContext(t)
		keeper := app.IbcComposabilityMwKeeper

		keeper.InitGenesis(ctx, genesisState)

		outputState := keeper.ExportGenesis(ctx)

		require.Equal(t, genesisState, *outputState)
	}
}
