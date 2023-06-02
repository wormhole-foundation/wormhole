package common

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"google.golang.org/protobuf/proto"
)

func createQueryRequestForTesting() *gossipv1.QueryRequest {
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
	callData := []*gossipv1.EthCallQueryRequest_EthCallData{
		{
			To:   to,
			Data: data1,
		},
		{
			To:   to,
			Data: data2,
		},
	}
	callRequest := &gossipv1.EthCallQueryRequest{
		Block:    block,
		CallData: callData,
	}

	queryRequest := &gossipv1.QueryRequest{
		ChainId: 5,
		Nonce:   0,
		Message: &gossipv1.QueryRequest_EthCallQueryRequest{
			EthCallQueryRequest: callRequest,
		},
	}

	return queryRequest
}

// A timestamp has nanos, but we only marshal down to micros, so trim our time to micros for testing purposes.
func timeForTest(t time.Time) time.Time {
	return time.UnixMicro(t.UnixMicro())
}

func TestQueryRequestProtoMarshalUnMarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting()
	queryRequestBytes, err := proto.Marshal(queryRequest)
	require.NoError(t, err)

	var queryRequest2 gossipv1.QueryRequest
	err = proto.Unmarshal(queryRequestBytes, &queryRequest2)
	require.NoError(t, err)

	assert.True(t, QueryRequestEqual(queryRequest, &queryRequest2))
}

func TestQueryRequestMarshalUnMarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting()
	queryRequestBytes, err := MarshalQueryRequest(queryRequest)
	require.NoError(t, err)

	queryRequest2, err := UnmarshalQueryRequest(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, QueryRequestEqual(queryRequest, queryRequest2))
}

func TestQueryResponseMarshalUnMarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting()
	queryRequestBytes, err := proto.Marshal(queryRequest)
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	results, err := hex.DecodeString("010203040506070809")
	require.NoError(t, err)

	respPub := &QueryResponsePublication{
		Request: signedQueryRequest,
		Responses: []EthCallQueryResponse{
			{
				Number: big.NewInt(42),
				Hash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
				Time:   timeForTest(time.Now()),
				Result: results,
			},
			{
				Number: big.NewInt(43),
				Hash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef9deadbeef"),
				Time:   timeForTest(time.Now()),
				Result: results,
			},
		},
	}

	respPubBytes, err := MarshalQueryResponsePublication(respPub)
	require.NoError(t, err)

	respPub2, err := UnmarshalQueryResponsePublication(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(respPub2))
}

func TesMarshalUnMarshalQueryResponseWithNoResults(t *testing.T) {
	queryRequest := createQueryRequestForTesting()
	queryRequestBytes, err := proto.Marshal(queryRequest)
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	respPub := &QueryResponsePublication{
		Request:   signedQueryRequest,
		Responses: nil,
	}

	respPubBytes, err := MarshalQueryResponsePublication(respPub)
	require.NoError(t, err)

	respPub2, err := UnmarshalQueryResponsePublication(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(respPub2))
}
