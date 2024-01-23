package query

import (
	"fmt"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func createQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)
	return createQueryResponseFromRequestWithRequestBytes(t, queryRequest, queryRequestBytes)
}

func createQueryResponseFromRequestWithRequestBytes(t *testing.T, queryRequest *QueryRequest, queryRequestBytes []byte) *QueryResponsePublication {
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
					Time:        timeForTest(t, time.Now()),
					Results:     results,
				},
			})
		case *EthCallByTimestampQueryRequest:
			results := [][]byte{}
			for idx := range req.CallData {
				result := []byte([]byte(fmt.Sprintf("Result %d", idx)))
				results = append(results, result[:])
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &EthCallByTimestampQueryResponse{
					TargetBlockNumber:    uint64(1000 + idx),
					TargetBlockHash:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					TargetBlockTime:      timeForTest(t, time.Now()),
					FollowingBlockNumber: uint64(1000 + idx + 1),
					FollowingBlockHash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
					FollowingBlockTime:   timeForTest(t, time.Now().Add(10*time.Second)),
					Results:              results,
				},
			})
		case *EthCallWithFinalityQueryRequest:
			results := [][]byte{}
			for idx := range req.CallData {
				result := []byte([]byte(fmt.Sprintf("Result %d", idx)))
				results = append(results, result[:])
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &EthCallWithFinalityQueryResponse{
					BlockNumber: uint64(1000 + idx),
					Hash:        ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Time:        timeForTest(t, time.Now()),
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
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

func TestQueryResponseUnmarshalWithExtraBytesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	respWithExtraBytes := append(respPubBytes, []byte("Hello, World!")[:]...)
	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respWithExtraBytes)
	assert.EqualError(t, err, "excess bytes in unmarshal")
}

func TestQueryResponseMarshalWithExtraRequestBytesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	requestWithExtraBytes := append(queryRequestBytes, []byte("Hello, World!")[:]...)
	respPub := createQueryResponseFromRequestWithRequestBytes(t, queryRequest, requestWithExtraBytes)

	// Marshal should fail because it calls Unmarshal on the request.
	_, err = respPub.Marshal()
	assert.EqualError(t, err, "failed to unmarshal query request: excess bytes in unmarshal")
}

func TestMarshalUnmarshalQueryResponseWithNoPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
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
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
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
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	for count := 0; count < 300; count++ {
		respPub.PerChainResponses = append(respPub.PerChainResponses, respPub.PerChainResponses[0])
	}

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithWrongNumberOfPerChainResponsesShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses = append(respPub.PerChainResponses, respPub.PerChainResponses[0])

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithInvalidChainIDShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses[0].ChainId = vaa.ChainIDUnset

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNilResponseShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPub.PerChainResponses[0].Response = nil

	_, err := respPub.Marshal()
	require.Error(t, err)
}

func TestMarshalUnmarshalQueryResponseWithNoResultsShouldFail(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
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
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
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
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
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

///////////// Solana Account Query tests /////////////////////////////////

func createSolanaAccountQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
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
		case *SolanaAccountQueryRequest:
			results := []SolanaAccountResult{}
			for idx := range req.Accounts {
				results = append(results, SolanaAccountResult{
					Lamports:   uint64(2000 + idx),
					RentEpoch:  uint64(3000 + idx),
					Executable: (idx%2 == 0),
					Owner:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Data:       []byte([]byte(fmt.Sprintf("Result %d", idx))),
				})
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &SolanaAccountQueryResponse{
					SlotNumber: uint64(1000 + idx),
					BlockTime:  timeForTest(t, time.Now()),
					BlockHash:  ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
					Results:    results,
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

func TestSolanaAccountQueryResponseMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaAccountQueryRequestForTesting(t)
	respPub := createSolanaAccountQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

///////////// End of Solana Account Query tests ///////////////////////////
