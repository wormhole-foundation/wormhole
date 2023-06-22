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

func createQueryRequestForTesting(chainId vaa.ChainID) *QueryRequest {
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
	callRequest := &EthCallQueryRequest{
		BlockId:  block,
		CallData: callData,
	}

	perChainQuery := &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery},
	}

	return queryRequest
}

// A timestamp has nanos, but we only marshal down to micros, so trim our time to micros for testing purposes.
func timeForTest(t time.Time) time.Time {
	return time.UnixMicro(t.UnixMicro())
}

func TestQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
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
	queryRequest := createQueryRequestForTesting(vaa.ChainIDUnset)
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

func TestPostSignedQueryRequestShouldFailIfNoOneIsListening(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
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

func createQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	perChainResponses := []*PerChainQueryResponse{}
	for idx, pcr := range queryRequest.PerChainQueries {
		switch req := pcr.Query.(type) {
		case *EthCallQueryRequest:
			results := [][]byte{}
			for idx := range req.CallData {
				result := []byte([]byte(fmt.Sprintf("Result %d", idx)))
				results = append(results, result[:])
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &EthCallQueryResponse{
					BlockNumber: uint64(1000 + idx),
					Hash:        ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Time:        timeForTest(time.Now()),
					Results:     results,
				},
			})
		default:
			panic("invalid query type!")
		}

	}

	return &QueryResponsePublication{
		Request:           signedQueryRequest,
		PerChainResponses: perChainResponses,
	}
}

func TestQueryResponseMarshalUnmarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

func TestMarshalUnmarshalQueryResponseWithNoPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	respPub := &QueryResponsePublication{
		Request:           signedQueryRequest,
		PerChainResponses: []*PerChainQueryResponse{},
	}

	_, err = respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNilPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	respPub := &QueryResponsePublication{
		Request:           signedQueryRequest,
		PerChainResponses: nil,
	}

	_, err = respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithTooManyPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	for count := 0; count < 300; count++ {
		respPub.PerChainResponses = append(respPub.PerChainResponses, respPub.PerChainResponses[0])
	}

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithWrongNumberOfPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses = append(respPub.PerChainResponses, respPub.PerChainResponses[0])

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithInvalidChainIDShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses[0].ChainId = vaa.ChainIDUnset

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNilResponseShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses[0].Response = nil

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNoResultsShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	switch resp := respPub.PerChainResponses[0].Response.(type) {
	case *EthCallQueryResponse:
		resp.Results = [][]byte{}
	default:
		panic("invalid query type!")
	}

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNilResultsShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	switch resp := respPub.PerChainResponses[0].Response.(type) {
	case *EthCallQueryResponse:
		resp.Results = nil
	default:
		panic("invalid query type!")
	}

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithTooManyResultsShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	results := [][]byte{}
	for count := 0; count < 300; count++ {
		results = append(results, []byte{})
	}

	switch resp := respPub.PerChainResponses[0].Response.(type) {
	case *EthCallQueryResponse:
		resp.Results = results
	default:
		panic("invalid query type!")
	}

	_, err := respPub.Marshal()
	require.Error(t, err)
}
