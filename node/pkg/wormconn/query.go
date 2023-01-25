package wormconn

import (
	"context"
	"fmt"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// SubmitQuery submits a query to a smart contract and returns the result.
func (c *ClientConn) SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error) {
	req := wasmdtypes.QuerySmartContractStateRequest{Address: contractAddress, QueryData: query}
	qc := wasmdtypes.NewQueryClient(c.c)
	if qc == nil {
		return []byte{}, fmt.Errorf("failed to create query client")
	}

	resp, err := qc.SmartContractState(ctx, &req)
	if err != nil {
		return []byte{}, err
	}

	return resp.Data, nil
}
