package governor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestFlowCancelPipesMainnetDeployment(t *testing.T) {
	tests := map[string][]pipe{
		"expected mainnet values": {
			{
				vaa.ChainIDEthereum,
				vaa.ChainIDSui,
			},
		},
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			got := FlowCancelPipes()
			require.Equal(t, 1, len(got))
			require.True(t, got[0].equals(&expected[0]))
		})
	}
}
