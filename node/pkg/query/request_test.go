package query

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

func createQueryRequestForTesting(t *testing.T, chainId vaa.ChainID) *QueryRequest {
	t.Helper()
	// Create a query request.
	wethAbi, err := abi.JSON(strings.NewReader("[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"))
	if err != nil {
		panic(err)
	}

	data1, err := wethAbi.Pack("name")
	if err != nil {
		panic(err)
	}
	data2, err := wethAbi.Pack("totalSupply")
	if err != nil {
		panic(err)
	}

	to, _ := hex.DecodeString("0d500b1d8e8ef31e21c99d1db9a6444d3adf1270")
	block := "0x28d9630"
	callData := []*EthCallData{
		{
			To:   to,
			Data: data1,
		},
		{
			To:   to,
			Data: data2,
		},
	}
	callRequest1 := &EthCallQueryRequest{
		BlockId:  block,
		CallData: callData,
	}

	perChainQuery1 := &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest1,
	}

	callRequest2 := &EthCallByTimestampQueryRequest{
		TargetTimestamp:      1697216322000000,
		TargetBlockIdHint:    "0x28d9630",
		FollowingBlockIdHint: "0x28d9631",
		CallData:             callData,
	}

	perChainQuery2 := &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest2,
	}

	callRequest3 := &EthCallWithFinalityQueryRequest{
		BlockId:  "0x28d9630",
		Finality: "finalized",
		CallData: callData,
	}

	perChainQuery3 := &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest3,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery1, perChainQuery2, perChainQuery3},
	}

	return queryRequest
}

// A timestamp has nanos, but we only marshal down to micros, so trim our time to micros for testing purposes.
func timeForTest(t *testing.T, ts time.Time) time.Time {
	t.Helper()
	return time.UnixMicro(ts.UnixMicro())
}

func TestQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
}

func TestQueryRequestUnmarshalWithExtraBytesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	withExtraBytes := append(queryRequestBytes, []byte("Hello, World!")[:]...)
	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(withExtraBytes)
	assert.EqualError(t, err, "excess bytes in unmarshal")
}

func TestMarshalOfQueryRequestWithNoPerChainQueriesShouldFail(t *testing.T) {
	queryRequest := &QueryRequest{
		Nonce: 1,
		PerChainQueries: []*PerChainQueryRequest{
			{
				ChainId: vaa.ChainIDPolygon,
				// Leave Query nil.
			},
		},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestWithTooManyPerChainQueriesShouldFail(t *testing.T) {
	perChainQueries := []*PerChainQueryRequest{}
	for count := 0; count < 300; count++ {
		callData := []*EthCallData{{

			To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", count))),
			Data: []byte(fmt.Sprintf("CallData for %d", count)),
		},
		}

		perChainQueries = append(perChainQueries, &PerChainQueryRequest{
			ChainId: vaa.ChainIDPolygon,
			Query: &EthCallQueryRequest{
				BlockId:  "0x28d9630",
				CallData: callData,
			},
		})
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: perChainQueries,
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestForInvalidChainIdShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDUnset)
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestWithInvalidBlockIdShouldFail(t *testing.T) {
	callData := []*EthCallData{{
		To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
		Data: []byte(fmt.Sprintf("CallData for %d", 0)),
	}}

	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId:  "latest",
			CallData: callData,
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestWithNoCallDataEntriesShouldFail(t *testing.T) {
	callData := []*EthCallData{}
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId:  "0x28d9630",
			CallData: callData,
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestWithNilCallDataEntriesShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId:  "0x28d9630",
			CallData: nil,
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfQueryRequestWithTooManyCallDataEntriesShouldFail(t *testing.T) {
	callData := []*EthCallData{}
	for count := 0; count < 300; count++ {
		callData = append(callData, &EthCallData{
			To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", count))),
			Data: []byte(fmt.Sprintf("CallData for %d", count)),
		})
	}

	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId:  "0x28d9630",
			CallData: callData,
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithNilToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   nil,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithEmptyToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   []byte{},
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithWrongLengthToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   []byte("TooShort"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithNilDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: nil,
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithEmptyDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: []byte{},
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallQueryWithWrongToLengthShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallQueryRequest{
			BlockId: "0x28d9630",
			CallData: []*EthCallData{
				{
					To:   []byte("This is too short!"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

///////////// EthCallByTimestamp tests ////////////////////////////////////////

func TestMarshalOfEthCallByTimestampQueryWithNilToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   nil,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallByTimestampQueryWithEmptyToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   []byte{},
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallByTimestampQueryWithWrongLengthToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   []byte("TooShort"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallByTimestampQueryWithNilDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: nil,
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallByTimestampQueryWithEmptyDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: []byte{},
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallByTimestampQueryWithWrongToLengthShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallByTimestampQueryRequest{
			TargetTimestamp: 1697216322000000,
			CallData: []*EthCallData{
				{
					To:   []byte("This is too short!"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

///////////// EthCallWithFinality tests ////////////////////////////////////////

func TestMarshalOfEthCallWithFinalityQueryWithNilToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "finalized",
			CallData: []*EthCallData{
				{
					To:   nil,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithEmptyToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "safe",
			CallData: []*EthCallData{
				{
					To:   []byte{},
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithWrongLengthToShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "finalized",
			CallData: []*EthCallData{
				{
					To:   []byte("TooShort"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithNilDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "safe",
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: nil,
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithEmptyDataShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "finalized",
			CallData: []*EthCallData{
				{
					To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d", 0))),
					Data: []byte{},
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithWrongToLengthShouldFail(t *testing.T) {
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "safe",
			CallData: []*EthCallData{
				{
					To:   []byte("This is too short!"),
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err := queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithBadFinality(t *testing.T) {
	to, err := hex.DecodeString("0d500b1d8e8ef31e21c99d1db9a6444d3adf1270")
	require.NoError(t, err)
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "HelloWorld",
			CallData: []*EthCallData{
				{
					To:   to,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err = queryRequest.Marshal()
	require.Error(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithFinalizedShouldSucceed(t *testing.T) {
	to, err := hex.DecodeString("0d500b1d8e8ef31e21c99d1db9a6444d3adf1270")
	require.NoError(t, err)
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "finalized",
			CallData: []*EthCallData{
				{
					To:   to,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err = queryRequest.Marshal()
	require.NoError(t, err)
}

func TestMarshalOfEthCallWithFinalityQueryWithSafeShouldSucceed(t *testing.T) {
	to, err := hex.DecodeString("0d500b1d8e8ef31e21c99d1db9a6444d3adf1270")
	require.NoError(t, err)
	perChainQuery := &PerChainQueryRequest{
		ChainId: vaa.ChainIDPolygon,
		Query: &EthCallWithFinalityQueryRequest{
			BlockId:  "0x28d9630",
			Finality: "safe",
			CallData: []*EthCallData{
				{
					To:   to,
					Data: []byte("This can't be zero length"),
				},
			},
		},
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}
	_, err = queryRequest.Marshal()
	require.NoError(t, err)
}

///////////// Solana Account Query tests /////////////////////////////////

func createSolanaAccountQueryRequestForTesting(t *testing.T) *QueryRequest {
	t.Helper()

	callRequest1 := &SolanaAccountQueryRequest{
		Commitment: "finalized",
		Accounts: [][SolanaPublicKeyLength]byte{
			ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
			ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
		},
	}

	perChainQuery1 := &PerChainQueryRequest{
		ChainId: vaa.ChainIDSolana,
		Query:   callRequest1,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery1},
	}

	return queryRequest
}

func TestSolanaAccountQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaAccountQueryRequestForTesting(t)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
}

func TestSolanaAccountQueryRequestMarshalUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("0000000966696e616c697a656400000000000000000000000000000000000000000000000002165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7")
	require.NoError(t, err)

	var solAccountReq SolanaAccountQueryRequest
	err = solAccountReq.Unmarshal(serialized)
	require.NoError(t, err)
}

func TestSolanaQueryMarshalUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("010000002a01000104000000660000000966696e616c697a65640000000000000000000000000000000000000000000000000202c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491")
	require.NoError(t, err)

	var solQuery QueryRequest
	err = solQuery.Unmarshal(serialized)
	require.NoError(t, err)
}

func TestSolanaPublicKeyLengthIsAsExpected(t *testing.T) {
	// It will break the spec if this ever changes!
	require.Equal(t, 32, SolanaPublicKeyLength)
}

///////////// Solana PDA Query tests /////////////////////////////////

func TestSolanaSeedConstsAreAsExpected(t *testing.T) {
	// It might break the spec if these ever changes!
	require.Equal(t, 16, SolanaMaxSeeds)
	require.Equal(t, 32, SolanaMaxSeedLen)
}

func createSolanaPdaQueryRequestForTesting(t *testing.T) *QueryRequest {
	t.Helper()

	callRequest1 := &SolanaPdaQueryRequest{
		Commitment: "finalized",
		PDAs: []SolanaPDAEntry{
			SolanaPDAEntry{
				ProgramAddress: ethCommon.HexToHash("0x02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"), // Devnet core bridge
				Seeds: [][]byte{
					[]byte("GuardianSet"),
					make([]byte, 4),
				},
			},
		},
	}

	perChainQuery1 := &PerChainQueryRequest{
		ChainId: vaa.ChainIDSolana,
		Query:   callRequest1,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery1},
	}

	return queryRequest
}

func TestSolanaPdaQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaPdaQueryRequestForTesting(t)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
}

func TestSolanaPdaQueryUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("010000002b010001050000005e0000000966696e616c697a656400000000000008ff000000000000000c00000000000000140102c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e5365740000000400000000")
	require.NoError(t, err)

	var solQuery QueryRequest
	err = solQuery.Unmarshal(serialized)
	require.NoError(t, err)
}

///////////// End of Solana PDA Query tests ///////////////////////////

func TestPostSignedQueryRequestShouldFailIfNoOneIsListening(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	var signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest
	assert.Error(t, PostSignedQueryRequest(signedQueryReqSendC, signedQueryRequest))
}
