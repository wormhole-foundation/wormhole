package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
)

func GetDenomsMetadata(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) QueryDenomsMetadataResponse {
	node := chain.FullNodes[0]

	stdoutBz, _, err := node.ExecQuery(ctx, "bank", "denom-metadata")
	require.NoError(t, err)

	fmt.Println("Stdout: ", string(stdoutBz))
	res := QueryDenomsMetadataResponse{}
	err = json.Unmarshal(stdoutBz, &res)
	require.NoError(t, err)

	return res
}

// QueryDenomsMetadataResponse is the response type for the Query/DenomsMetadata RPC
// method.
type QueryDenomsMetadataResponse struct {
	// metadata provides the client information for all the registered tokens.
	Metadatas []banktypes.Metadata `json:"metadatas"`
}
