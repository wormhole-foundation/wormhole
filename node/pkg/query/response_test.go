package query

import (
	"bytes"
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
					BlockNumber: uint64(1000 + idx), // #nosec G115 -- This is safe in this test suite
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
					TargetBlockNumber:    uint64(1000 + idx), // #nosec G115 -- This is safe in this test suite
					TargetBlockHash:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					TargetBlockTime:      timeForTest(t, time.Now()),
					FollowingBlockNumber: uint64(1000 + idx + 1), // #nosec G115 -- This is safe in this test suite
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
					BlockNumber: uint64(1000 + idx), // #nosec G115 -- This is safe in this test suite
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
					Lamports:   uint64(2000 + idx), // #nosec G115 -- This is safe in this test suite
					RentEpoch:  uint64(3000 + idx), // #nosec G115 -- This is safe in this test suite
					Executable: (idx%2 == 0),
					Owner:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Data:       []byte([]byte(fmt.Sprintf("Result %d", idx))),
				})
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &SolanaAccountQueryResponse{
					SlotNumber: uint64(1000 + idx), // #nosec G115 -- This is safe in this test suite
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

///////////// Solana PDA Query tests /////////////////////////////////

func createSolanaPdaQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
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
		case *SolanaPdaQueryRequest:
			results := []SolanaPdaResult{}
			for idx := range req.PDAs {
				results = append(results, SolanaPdaResult{
					Account:    ethCommon.HexToHash("4fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773e"),
					Bump:       uint8(255 - idx),   // #nosec G115 -- This is safe in this test suite
					Lamports:   uint64(2000 + idx), // #nosec G115 -- This is safe in this test suite
					RentEpoch:  uint64(3000 + idx), // #nosec G115 -- This is safe in this test suite
					Executable: (idx%2 == 0),
					Owner:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Data:       []byte([]byte(fmt.Sprintf("Result %d", idx))),
				})
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &SolanaPdaQueryResponse{
					SlotNumber: uint64(1000 + idx), // #nosec G115 -- This is safe in this test suite
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

func TestSolanaPdaQueryResponseMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaPdaQueryRequestForTesting(t)
	respPub := createSolanaPdaQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

///////////// End of Solana PDA Query tests ///////////////////////////

// marshalV1QueryRequestForResponseTest builds v1 wire-format bytes from a v2 QueryRequest.
// v1 format: [version=1][nonce][numQueries][per-chain-queries...] — no timestamp or staker address.
func marshalV1QueryRequestForResponseTest(t *testing.T, qr *QueryRequest) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	buf.WriteByte(MSG_VERSION_V1)
	b := make([]byte, 4)
	b[0] = byte(qr.Nonce >> 24)
	b[1] = byte(qr.Nonce >> 16)
	b[2] = byte(qr.Nonce >> 8)
	b[3] = byte(qr.Nonce)
	buf.Write(b)
	buf.WriteByte(uint8(len(qr.PerChainQueries))) //nolint:gosec // test code with controlled input
	for _, pcq := range qr.PerChainQueries {
		pcqBytes, err := pcq.Marshal()
		require.NoError(t, err)
		buf.Write(pcqBytes)
	}
	return buf.Bytes()
}

func TestResponseWithEmbeddedV1RequestUnmarshal(t *testing.T) {
	// Simulate the real-world scenario: a guardian responds to a v1 request
	// from another CCQ server. The response embeds the v1 request bytes.
	// Our v2 CCQ server must be able to unmarshal this response.
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	v1RequestBytes := marshalV1QueryRequestForResponseTest(t, queryRequest)

	respPub := createQueryResponseFromRequestWithRequestBytes(t, queryRequest, v1RequestBytes)
	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2.Request)
	assert.Equal(t, v1RequestBytes, respPub2.Request.QueryRequest)
}

func TestResponseWithEmbeddedV2RequestUnmarshal(t *testing.T) {
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)

	assert.True(t, respPub.Equal(&respPub2))
}

func TestResponseWithCorruptedEmbeddedRequestIsRejected(t *testing.T) {
	// Verify that Validate() (via Unmarshal) rejects responses where the
	// embedded request bytes are garbage, regardless of version.
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	// The embedded request starts at offset 68 (1 version + 2 chainID + 65 signature)
	// followed by 4 bytes of length. Corrupt the request bytes after the length field.
	requestOffset := 1 + 2 + 65 + 4
	require.Greater(t, len(respPubBytes), requestOffset+5)
	for i := requestOffset; i < requestOffset+5; i++ {
		respPubBytes[i] = 0xFF
	}

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	assert.Error(t, err)
}

func TestResponseValidateRejectsCountMismatch(t *testing.T) {
	// Build a valid response, then add an extra per-chain response to
	// create a count mismatch with the embedded request's queries.
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	// Duplicate the last response to create a count mismatch.
	extra := *respPub.PerChainResponses[len(respPub.PerChainResponses)-1]
	respPub.PerChainResponses = append(respPub.PerChainResponses, &extra)

	err := respPub.Validate()
	assert.ErrorContains(t, err, "number of responses does not match number of queries")
}

func TestResponseValidateRejectsTypeMismatch(t *testing.T) {
	// Build a valid response, then swap a response type so it doesn't
	// match the corresponding query type.
	queryRequest := createQueryRequestForTesting(t, vaa.ChainIDPolygon)
	respPub := createQueryResponseFromRequest(t, queryRequest)

	// The first query is EthCallQueryRequest. Replace its response with
	// an EthCallByTimestampQueryResponse to create a type mismatch.
	respPub.PerChainResponses[0] = &PerChainQueryResponse{
		ChainId: vaa.ChainIDPolygon,
		Response: &EthCallByTimestampQueryResponse{
			TargetBlockNumber:    1000,
			TargetBlockHash:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
			TargetBlockTime:      timeForTest(t, time.Now()),
			FollowingBlockNumber: 1001,
			FollowingBlockHash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
			FollowingBlockTime:   timeForTest(t, time.Now()),
			Results:              [][]byte{{0x01}},
		},
	}

	err := respPub.Validate()
	assert.ErrorContains(t, err, "type of response 0 does not match the query")
}
