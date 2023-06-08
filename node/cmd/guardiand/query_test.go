package guardiand

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	testSigner = "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
)

func createPerChainQueryForTesting(
	chainId vaa.ChainID,
) *gossipv1.PerChainQueryRequest {
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

	return &gossipv1.PerChainQueryRequest{
		ChainId: uint32(chainId),
		Message: &gossipv1.PerChainQueryRequest_EthCallQueryRequest{
			EthCallQueryRequest: callRequest,
		},
	}
}

func createSignedQueryRequestForTesting(
	sk *ecdsa.PrivateKey,
	perChainQueries []*gossipv1.PerChainQueryRequest,
) (*gossipv1.SignedQueryRequest, *gossipv1.QueryRequest, []common.PerChainQueryResponse) {
	queryRequest := &gossipv1.QueryRequest{
		Nonce:           1,
		PerChainQueries: perChainQueries,
	}

	queryRequestBytes, err := proto.Marshal(queryRequest)
	if err != nil {
		panic(err)
	}

	digest := common.QueryRequestDigest(common.UnsafeDevNet, queryRequestBytes)
	sig, err := ethCrypto.Sign(digest.Bytes(), sk)
	if err != nil {
		panic(err)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig,
	}

	expectedResults := createExpectedResultsForTest(queryRequest.PerChainQueries)
	return signedQueryRequest, queryRequest, expectedResults
}

func createExpectedResultsForTest(perChainQueries []*gossipv1.PerChainQueryRequest) []common.PerChainQueryResponse {
	expectedResults := []common.PerChainQueryResponse{}
	for _, pcq := range perChainQueries {
		switch req := pcq.Message.(type) {
		case *gossipv1.PerChainQueryRequest_EthCallQueryRequest:
			now := time.Now()
			blockNum, err := strconv.ParseInt(strings.TrimPrefix(req.EthCallQueryRequest.Block, "0x"), 16, 64)
			if err != nil {
				panic("failed to parse block number!")
			}
			resp := []common.EthCallQueryResponse{}
			for _, cd := range req.EthCallQueryRequest.CallData {
				resp = append(resp, common.EthCallQueryResponse{
					Number: big.NewInt(blockNum),
					Hash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Time:   timeForTest(timeForTest(now)),
					Result: []byte(hex.EncodeToString(cd.To) + ":" + hex.EncodeToString(cd.Data)),
				})
			}
			expectedResults = append(expectedResults, common.PerChainQueryResponse{
				ChainID:   pcq.ChainId,
				Responses: resp,
			})

		default:
			panic("Invalid call data type!")
		}
	}

	return expectedResults
}

// A timestamp has nanos, but we only marshal down to micros, so trim our time to micros for testing purposes.
func timeForTest(t time.Time) time.Time {
	return time.UnixMicro(t.UnixMicro())
}

func TestCcqParseAllowedRequestersSuccess(t *testing.T) {
	ccqAllowedRequestersList, err := ccqParseAllowedRequesters(testSigner)
	require.NoError(t, err)
	require.NotNil(t, ccqAllowedRequestersList)
	require.Equal(t, 1, len(ccqAllowedRequestersList))

	_, exists := ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes(testSigner))]
	require.True(t, exists)
	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf"))]
	require.False(t, exists)

	ccqAllowedRequestersList, err = ccqParseAllowedRequesters("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe,beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf")
	require.NoError(t, err)
	require.NotNil(t, ccqAllowedRequestersList)
	require.Equal(t, 2, len(ccqAllowedRequestersList))

	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes(testSigner))]
	require.True(t, exists)
	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf"))]
	require.True(t, exists)
}

func TestCcqParseAllowedRequestersFailsIfParameterEmpty(t *testing.T) {
	ccqAllowedRequestersList, err := ccqParseAllowedRequesters("")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)

	ccqAllowedRequestersList, err = ccqParseAllowedRequesters(",")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)
}

func TestCcqParseAllowedRequestersFailsIfInvalidParameter(t *testing.T) {
	ccqAllowedRequestersList, err := ccqParseAllowedRequesters("Hello")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)
}

type mockData struct {
	sk *ecdsa.PrivateKey

	signedQueryReqReadC  <-chan *gossipv1.SignedQueryRequest
	signedQueryReqWriteC chan<- *gossipv1.SignedQueryRequest

	chainQueryReqC map[vaa.ChainID]chan *common.PerChainQueryInternal

	queryResponseReadC  <-chan *common.PerChainQueryResponseInternal
	queryResponseWriteC chan<- *common.PerChainQueryResponseInternal

	queryResponsePublicationReadC  <-chan *common.QueryResponsePublication
	queryResponsePublicationWriteC chan<- *common.QueryResponsePublication

	chainQueryResponseC map[vaa.ChainID]chan *common.PerChainQueryResponseInternal

	mutex                    sync.Mutex
	queryResponsePublication *common.QueryResponsePublication
	expectedResults          []common.PerChainQueryResponse
	requestsPerChain         map[vaa.ChainID]int
}

func (md *mockData) setExpectedResults(expectedResults []common.PerChainQueryResponse) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.expectedResults = expectedResults
}

func (md *mockData) getQueryResponsePublication() *common.QueryResponsePublication {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	return md.queryResponsePublication
}

func (md *mockData) incrementRequestsPerChainAlreadyLocked(chainId vaa.ChainID) {
	if val, exists := md.requestsPerChain[chainId]; exists {
		md.requestsPerChain[chainId] = val + 1
	} else {
		md.requestsPerChain[chainId] = 1
	}
}

func (md *mockData) getRequestsPerChain(chainId vaa.ChainID) int {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	if ret, exists := md.requestsPerChain[chainId]; exists {
		return ret
	}
	return 0
}

func createQueryHandlerForTest(t *testing.T, ctx context.Context, logger *zap.Logger) *mockData {
	var md mockData
	var err error

	*unsafeDevMode = true
	md.sk, err = loadGuardianKey("../../hack/query/dev.guardian.key")
	require.NoError(t, err)
	require.NotNil(t, md.sk)

	ccqAllowedRequestersList, err := ccqParseAllowedRequesters(testSigner)
	require.NoError(t, err)

	// Inbound observation requests from the p2p service (for all chains)
	md.signedQueryReqReadC, md.signedQueryReqWriteC = makeChannelPair[*gossipv1.SignedQueryRequest](common.SignedQueryRequestChannelSize)

	// Per-chain query requests
	chainQueryReqPolygon := make(chan *common.PerChainQueryInternal)
	md.chainQueryReqC = make(map[vaa.ChainID]chan *common.PerChainQueryInternal)
	md.chainQueryReqC[vaa.ChainIDPolygon] = chainQueryReqPolygon

	// Query responses from watchers to query handler aggregated across all chains
	md.queryResponseReadC, md.queryResponseWriteC = makeChannelPair[*common.PerChainQueryResponseInternal](0)

	// Query responses from query handler to p2p
	md.queryResponsePublicationReadC, md.queryResponsePublicationWriteC = makeChannelPair[*common.QueryResponsePublication](0)

	// Per-chain query response channel
	md.chainQueryResponseC = make(map[vaa.ChainID]chan *common.PerChainQueryResponseInternal)
	md.chainQueryResponseC[vaa.ChainIDPolygon] = make(chan *common.PerChainQueryResponseInternal)

	md.requestsPerChain = make(map[vaa.ChainID]int)

	go handleQueryRequests(ctx, logger, md.signedQueryReqReadC, md.chainQueryReqC, ccqAllowedRequestersList, md.queryResponseReadC, md.queryResponsePublicationWriteC, common.GoTest)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case pcqr := <-chainQueryReqPolygon:
				require.Equal(t, vaa.ChainIDPolygon, pcqr.ChainID)
				md.mutex.Lock()
				md.incrementRequestsPerChainAlreadyLocked(vaa.ChainIDPolygon)
				results := md.expectedResults[pcqr.RequestIdx].Responses
				queryResponse := common.CreatePerChainQueryResponseInternal(pcqr.RequestID, pcqr.RequestIdx, pcqr.ChainID, common.QuerySuccess, results)
				md.queryResponseWriteC <- queryResponse
				md.mutex.Unlock()
			case qrp := <-md.queryResponsePublicationReadC:
				md.mutex.Lock()
				md.queryResponsePublication = qrp
				md.mutex.Unlock()
			}
		}
	}()

	return &md
}

func (md *mockData) waitForResponse() *common.QueryResponsePublication {
	for count := 0; count < 100; count++ {
		time.Sleep(10 * time.Millisecond)
		ret := md.getQueryResponsePublication()
		if ret != nil {
			return ret
		}
	}
	return nil
}

func TestSimpleQueryResponse(t *testing.T) {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	md := createQueryHandlerForTest(t, ctx, logger)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*gossipv1.PerChainQueryRequest{createPerChainQueryForTesting(vaa.ChainIDPolygon)}
	signedQueryRequest, queryRequest, expectedResults := createSignedQueryRequestForTesting(md.sk, perChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Wait until we receive a response or timeout.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateTestResponse(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

// validateTestResponse performs validation on the responses generated by these tests. Note that it is not a generalized validate function.
func validateTestResponse(
	t *testing.T,
	response *common.QueryResponsePublication,
	signedRequest *gossipv1.SignedQueryRequest,
	queryRequest *gossipv1.QueryRequest,
	expectedResults []common.PerChainQueryResponse,
) bool {
	require.NotNil(t, response)
	require.True(t, common.SignedQueryRequestEqual(signedRequest, response.Request))
	require.Equal(t, len(queryRequest.PerChainQueries), len(response.PerChainResponses))
	require.True(t, bytes.Equal(response.Request.Signature, signedRequest.Signature))
	require.Equal(t, len(response.PerChainResponses), len(expectedResults))
	for idx := range response.PerChainResponses {
		require.True(t, response.PerChainResponses[idx].Equal(&expectedResults[idx]))
	}

	return true
}
