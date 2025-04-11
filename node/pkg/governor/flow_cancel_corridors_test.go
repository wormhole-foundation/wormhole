package governor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestFlowCancelCorridorsMainnetDeployment(t *testing.T) {

	tests := map[string][]corridor{
		"expected mainnet values": {
			{
				vaa.ChainIDEthereum,
				vaa.ChainIDSui,
			},
		},
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			got := FlowCancelCorridors()
			// Basic validity check.
			require.True(t, ValidateCorridors(got))

			// Check that values are what we expected.
			require.Equal(t, 1, len(got))
			require.True(t, got[0].equals(&expected[0]))
		})
	}
}

func TestValidateCorridors(t *testing.T) {
	tests := []struct {
		name string
		args []corridor
		want bool
	}{
		{
			name: "error: duplicate pipes",
			args: []corridor{
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDSolana,
				},
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDSolana,
				},
			},
			want: false,
		},
		{
			name: "error: duplicate pipes, different order",
			args: []corridor{
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDSolana,
				},
				{
					first:  vaa.ChainIDSolana,
					second: vaa.ChainIDEthereum,
				},
			},
			want: false,
		},
		{
			name: "error: invalid pipe (ends are the same)",
			args: []corridor{
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDEthereum,
				},
			},
			want: false,
		},
		{
			name: "error: invalid pipe (unset)",
			args: []corridor{
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDUnset,
				},
			},
			want: false,
		},
		{
			name: "happy path",
			args: []corridor{
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDSui,
				},
				{
					first:  vaa.ChainIDEthereum,
					second: vaa.ChainIDSolana,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCorridors(tt.args); got != tt.want {
				require.Equal(t, tt.want, got, "want %v got %v value %v", tt.want, got, tt.args)
			}
		})
	}
}
