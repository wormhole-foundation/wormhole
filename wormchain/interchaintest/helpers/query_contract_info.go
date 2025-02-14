package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// QueryContractInfo queries the information about a contract like the admin and code_id.
func QueryContractInfo(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, contractAddress string) ContractInfoResponse {
	stdout, _, err := chain.FullNodes[0].ExecQuery(ctx,
		"wasm", "contract", contractAddress,
	)
	require.NoError(t, err)

	res := new(ContractInfoResponse)
	err = json.Unmarshal(stdout, res)
	require.NoError(t, err)

	return *res
}

type ContractInfoResponse struct {
	Address      string `json:"address"`
	ContractInfo struct {
		CodeID  string `json:"code_id"`
		Creator string `json:"creator"`
		Admin   string `json:"admin"`
		Label   string `json:"label"`
		Created struct {
			BlockHeight string `json:"block_height"`
			TxIndex     string `json:"tx_index"`
		} `json:"created"`
		IbcPortID string `json:"ibc_port_id"`
		Extension any    `json:"extension"`
	} `json:"contract_info"`
}
