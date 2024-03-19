package query

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math"
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

	"go.uber.org/zap"
)

const (
	testSigner = "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"

	// Magic retry values used to cause special behavior in the watchers.
	fatalError  = math.MaxInt
	ignoreQuery = math.MaxInt - 1

	// Speed things up for testing purposes.
	requestTimeoutForTest = 100 * time.Millisecond
	retryIntervalForTest  = 10 * time.Millisecond
	auditIntervalForTest  = 10 * time.Millisecond
	pollIntervalForTest   = 5 * time.Millisecond
)

var (
	nonce = uint32(0)

	watcherChainsForTest = []vaa.ChainID{vaa.ChainIDPolygon, vaa.ChainIDBSC, vaa.ChainIDArbitrum}
)

// createPerChainQueryForEthCall creates a per chain query for an eth_call for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCall(
	t *testing.T,
	chainId vaa.ChainID,
	block string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := 0; count < numCalls; count++ {
		ethCallData = append(ethCallData, &EthCallData{
			To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d:%d", chainId, count))),
			Data: []byte(fmt.Sprintf("CallData for %d:%d", chainId, count)),
		})
	}

	callRequest := &EthCallQueryRequest{
		BlockId:  block,
		CallData: ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest,
	}
}

// createPerChainQueryForEthCallByTimestamp creates a per chain query for an eth_call_by_timestamp for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCallByTimestamp(
	t *testing.T,
	chainId vaa.ChainID,
	targetBlock string,
	followingBlock string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := 0; count < numCalls; count++ {
		ethCallData = append(ethCallData, &EthCallData{
			To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d:%d", chainId, count))),
			Data: []byte(fmt.Sprintf("CallData for %d:%d", chainId, count)),
		})
	}

	callRequest := &EthCallByTimestampQueryRequest{
		TargetTimestamp:      1697216322000000,
		TargetBlockIdHint:    targetBlock,
		FollowingBlockIdHint: followingBlock,
		CallData:             ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest,
	}
}

// createPerChainQueryForEthCallWithFinality creates a per chain query for an eth_call_with_finality for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCallWithFinality(
	t *testing.T,
	chainId vaa.ChainID,
	blockId string,
	finality string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := 0; count < numCalls; count++ {
		ethCallData = append(ethCallData, &EthCallData{
			To:   []byte(fmt.Sprintf("%-20s", fmt.Sprintf("To for %d:%d", chainId, count))),
			Data: []byte(fmt.Sprintf("CallData for %d:%d", chainId, count)),
		})
	}

	callRequest := &EthCallWithFinalityQueryRequest{
		BlockId:  blockId,
		Finality: finality,
		CallData: ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainId,
		Query:   callRequest,
	}
}

// createSignedQueryRequestForTesting creates a query request object and signs it using the specified key.
func createSignedQueryRequestForTesting(
	t *testing.T,
	sk *ecdsa.PrivateKey,
	perChainQueries []*PerChainQueryRequest,
) (*gossipv1.SignedQueryRequest, *QueryRequest) {
	t.Helper()
	nonce += 1
	queryRequest := &QueryRequest{
		Nonce:           nonce,
		PerChainQueries: perChainQueries,
	}

	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		panic(err)
	}

	digest := QueryRequestDigest(common.UnsafeDevNet, queryRequestBytes)
	sig, err := ethCrypto.Sign(digest.Bytes(), sk)
	if err != nil {
		panic(err)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig,
	}

	return signedQueryRequest, queryRequest
}

// createExpectedResultsForTest generates an array of the results expected for a request. These results are returned by the watcher, and used to validate the response.
func createExpectedResultsForTest(t *testing.T, perChainQueries []*PerChainQueryRequest) []PerChainQueryResponse {
	t.Helper()
	expectedResults := []PerChainQueryResponse{}
	for _, pcq := range perChainQueries {
		switch req := pcq.Query.(type) {
		case *EthCallQueryRequest:
			now := time.Now()
			blockNum, err := strconv.ParseUint(strings.TrimPrefix(req.BlockId, "0x"), 16, 64)
			if err != nil {
				panic("invalid blockNum!")
			}
			resp := &EthCallQueryResponse{
				BlockNumber: blockNum,
				Hash:        ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
				Time:        timeForTest(t, now),
				Results:     [][]byte{},
			}
			for _, cd := range req.CallData {
				resp.Results = append(resp.Results, []byte(hex.EncodeToString(cd.To)+":"+hex.EncodeToString(cd.Data)))
			}
			expectedResults = append(expectedResults, PerChainQueryResponse{
				ChainId:  pcq.ChainId,
				Response: resp,
			})
		case *EthCallByTimestampQueryRequest:
			now := time.Now()
			blockNum, err := strconv.ParseUint(strings.TrimPrefix(req.TargetBlockIdHint, "0x"), 16, 64)
			if err != nil {
				panic("invalid blockNum!")
			}
			resp := &EthCallByTimestampQueryResponse{
				TargetBlockNumber:    blockNum,
				TargetBlockHash:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
				TargetBlockTime:      timeForTest(t, now),
				FollowingBlockNumber: blockNum + 1,
				FollowingBlockHash:   ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
				FollowingBlockTime:   timeForTest(t, time.Now().Add(10*time.Second)),
				Results:              [][]byte{},
			}
			for _, cd := range req.CallData {
				resp.Results = append(resp.Results, []byte(hex.EncodeToString(cd.To)+":"+hex.EncodeToString(cd.Data)))
			}
			expectedResults = append(expectedResults, PerChainQueryResponse{
				ChainId:  pcq.ChainId,
				Response: resp,
			})
		case *EthCallWithFinalityQueryRequest:
			now := time.Now()
			blockNum, err := strconv.ParseUint(strings.TrimPrefix(req.BlockId, "0x"), 16, 64)
			if err != nil {
				panic("invalid blockNum!")
			}
			resp := &EthCallQueryResponse{
				BlockNumber: blockNum,
				Hash:        ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
				Time:        timeForTest(t, now),
				Results:     [][]byte{},
			}
			for _, cd := range req.CallData {
				resp.Results = append(resp.Results, []byte(hex.EncodeToString(cd.To)+":"+hex.EncodeToString(cd.Data)))
			}
			expectedResults = append(expectedResults, PerChainQueryResponse{
				ChainId:  pcq.ChainId,
				Response: resp,
			})
		default:
			panic("Invalid call data type!")
		}
	}

	return expectedResults
}

// validateResponseForTest performs validation on the responses generated by these tests. Note that it is not a generalized validate function.
func validateResponseForTest(
	t *testing.T,
	response *QueryResponsePublication,
	signedRequest *gossipv1.SignedQueryRequest,
	queryRequest *QueryRequest,
	expectedResults []PerChainQueryResponse,
) bool {
	require.NotNil(t, response)
	require.True(t, SignedQueryRequestEqual(signedRequest, response.Request))
	require.Equal(t, len(queryRequest.PerChainQueries), len(response.PerChainResponses))
	require.True(t, bytes.Equal(response.Request.Signature, signedRequest.Signature))
	require.Equal(t, len(response.PerChainResponses), len(expectedResults))
	for idx := range response.PerChainResponses {
		require.True(t, response.PerChainResponses[idx].Equal(&expectedResults[idx]))
	}

	return true
}

func TestParseAllowedRequestersSuccess(t *testing.T) {
	ccqAllowedRequestersList, err := parseAllowedRequesters(testSigner)
	require.NoError(t, err)
	require.NotNil(t, ccqAllowedRequestersList)
	require.Equal(t, 1, len(ccqAllowedRequestersList))

	_, exists := ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes(testSigner))]
	require.True(t, exists)
	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf"))]
	require.False(t, exists)

	ccqAllowedRequestersList, err = parseAllowedRequesters("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe,beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf")
	require.NoError(t, err)
	require.NotNil(t, ccqAllowedRequestersList)
	require.Equal(t, 2, len(ccqAllowedRequestersList))

	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes(testSigner))]
	require.True(t, exists)
	_, exists = ccqAllowedRequestersList[ethCommon.BytesToAddress(ethCommon.Hex2Bytes("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBf"))]
	require.True(t, exists)
}

func TestParseAllowedRequestersFailsIfParameterEmpty(t *testing.T) {
	ccqAllowedRequestersList, err := parseAllowedRequesters("")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)

	ccqAllowedRequestersList, err = parseAllowedRequesters(",")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)
}

func TestParseAllowedRequestersFailsIfInvalidParameter(t *testing.T) {
	ccqAllowedRequestersList, err := parseAllowedRequesters("Hello")
	require.Error(t, err)
	require.Nil(t, ccqAllowedRequestersList)
}

// mockData is the data structure used to mock up the query handler environment.
type mockData struct {
	sk *ecdsa.PrivateKey

	signedQueryReqReadC  <-chan *gossipv1.SignedQueryRequest
	signedQueryReqWriteC chan<- *gossipv1.SignedQueryRequest

	chainQueryReqC map[vaa.ChainID]chan *PerChainQueryInternal

	queryResponseReadC  <-chan *PerChainQueryResponseInternal
	queryResponseWriteC chan<- *PerChainQueryResponseInternal

	queryResponsePublicationReadC  <-chan *QueryResponsePublication
	queryResponsePublicationWriteC chan<- *QueryResponsePublication

	mutex                    sync.Mutex
	queryResponsePublication *QueryResponsePublication
	expectedResults          []PerChainQueryResponse
	requestsPerChain         map[vaa.ChainID]int
	retriesPerChain          map[vaa.ChainID]int
}

// resetState() is used to reset mock data between queries in the same test.
func (md *mockData) resetState() {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.queryResponsePublication = nil
	md.expectedResults = nil
	md.requestsPerChain = make(map[vaa.ChainID]int)
	md.retriesPerChain = make(map[vaa.ChainID]int)
}

// setExpectedResults sets the results to be returned by the watchers.
func (md *mockData) setExpectedResults(expectedResults []PerChainQueryResponse) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.expectedResults = expectedResults
}

// setRetries allows a test to specify how many times a given watcher should retry before returning success.
// If the count is the special value `fatalError`, the watcher will return QueryFatalError.
func (md *mockData) setRetries(chainId vaa.ChainID, count int) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.retriesPerChain[chainId] = count
}

// incrementRequestsPerChainAlreadyLocked is used by the watchers to keep track of how many times they were invoked in a given test.
func (md *mockData) incrementRequestsPerChainAlreadyLocked(chainId vaa.ChainID) {
	if val, exists := md.requestsPerChain[chainId]; exists {
		md.requestsPerChain[chainId] = val + 1
	} else {
		md.requestsPerChain[chainId] = 1
	}
}

// getQueryResponsePublication returns the latest query response publication received by the mock.
func (md *mockData) getQueryResponsePublication() *QueryResponsePublication {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	return md.queryResponsePublication
}

// getRequestsPerChain returns the count of the number of times the given watcher was invoked in a given test.
func (md *mockData) getRequestsPerChain(chainId vaa.ChainID) int {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	if ret, exists := md.requestsPerChain[chainId]; exists {
		return ret
	}
	return 0
}

// shouldIgnoreAlreadyLocked is used by the watchers to see if they should ignore a query (causing a retry).
func (md *mockData) shouldIgnoreAlreadyLocked(chainId vaa.ChainID) bool {
	if val, exists := md.retriesPerChain[chainId]; exists {
		if val == ignoreQuery {
			delete(md.retriesPerChain, chainId)
			return true
		}
	}
	return false
}

// getStatusAlreadyLocked is used by the watchers to determine what query status they should return, based on the `retriesPerChain`.
func (md *mockData) getStatusAlreadyLocked(chainId vaa.ChainID) QueryStatus {
	if val, exists := md.retriesPerChain[chainId]; exists {
		if val == fatalError {
			return QueryFatalError
		}
		val -= 1
		if val > 0 {
			md.retriesPerChain[chainId] = val
		} else {
			delete(md.retriesPerChain, chainId)
		}
		return QueryRetryNeeded
	}
	return QuerySuccess
}

// createQueryHandlerForTest creates the query handler mock environment, including the set of watchers and the response listener.
// Most tests will use this function to set up the mock.
func createQueryHandlerForTest(t *testing.T, ctx context.Context, logger *zap.Logger, chains []vaa.ChainID) *mockData {
	md := createQueryHandlerForTestWithoutPublisher(t, ctx, logger, chains)
	md.startResponseListener(ctx)
	return md
}

// createQueryHandlerForTestWithoutPublisher creates the query handler mock environment, including the set of watchers but not the response listener.
// This function can be invoked directly to test retries of response publication (by delaying the start of the response listener).
func createQueryHandlerForTestWithoutPublisher(t *testing.T, ctx context.Context, logger *zap.Logger, chains []vaa.ChainID) *mockData {
	md := mockData{}
	var err error

	md.sk, err = common.LoadGuardianKey("dev.guardian.key", true)
	require.NoError(t, err)
	require.NotNil(t, md.sk)

	ccqAllowedRequestersList, err := parseAllowedRequesters(testSigner)
	require.NoError(t, err)

	// Inbound observation requests from the p2p service (for all chains)
	md.signedQueryReqReadC, md.signedQueryReqWriteC = makeChannelPair[*gossipv1.SignedQueryRequest](SignedQueryRequestChannelSize)

	// Per-chain query requests
	md.chainQueryReqC = make(map[vaa.ChainID]chan *PerChainQueryInternal)
	for _, chainId := range chains {
		md.chainQueryReqC[chainId] = make(chan *PerChainQueryInternal)
	}

	// Query responses from watchers to query handler aggregated across all chains
	md.queryResponseReadC, md.queryResponseWriteC = makeChannelPair[*PerChainQueryResponseInternal](0)

	// Query responses from query handler to p2p
	md.queryResponsePublicationReadC, md.queryResponsePublicationWriteC = makeChannelPair[*QueryResponsePublication](0)

	md.resetState()

	go func() {
		err := handleQueryRequestsImpl(ctx, logger, md.signedQueryReqReadC, md.chainQueryReqC, ccqAllowedRequestersList,
			md.queryResponseReadC, md.queryResponsePublicationWriteC, common.GoTest, requestTimeoutForTest, retryIntervalForTest, auditIntervalForTest)
		assert.NoError(t, err)
	}()

	// Create a routine for each configured watcher. It will take a per chain query and return the corresponding expected result.
	// It also pegs a counter of the number of requests the watcher received, for verification purposes.
	for chainId := range md.chainQueryReqC {
		go func(chainId vaa.ChainID, chainQueryReqC <-chan *PerChainQueryInternal) {
			for {
				select {
				case <-ctx.Done():
					return
				case pcqr := <-chainQueryReqC:
					require.Equal(t, chainId, pcqr.Request.ChainId)
					md.mutex.Lock()
					md.incrementRequestsPerChainAlreadyLocked(chainId)
					if md.shouldIgnoreAlreadyLocked(chainId) {
						logger.Info("watcher ignoring query", zap.String("chainId", chainId.String()), zap.Int("requestIdx", pcqr.RequestIdx))
					} else {
						results := md.expectedResults[pcqr.RequestIdx].Response
						status := md.getStatusAlreadyLocked(chainId)
						logger.Info("watcher returning", zap.String("chainId", chainId.String()), zap.Int("requestIdx", pcqr.RequestIdx), zap.Int("status", int(status)))
						queryResponse := CreatePerChainQueryResponseInternal(pcqr.RequestID, pcqr.RequestIdx, pcqr.Request.ChainId, status, results)
						md.queryResponseWriteC <- queryResponse
					}
					md.mutex.Unlock()
				}
			}
		}(chainId, md.chainQueryReqC[chainId])
	}

	return &md
}

// startResponseListener starts the response listener routine. It is called as part of the standard mock environment set up. Or, it can be used
// along with `createQueryHandlerForTestWithoutPublisher“ to test retries of response publication (by delaying the start of the response listener).
func (md *mockData) startResponseListener(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case qrp := <-md.queryResponsePublicationReadC:
				md.mutex.Lock()
				md.queryResponsePublication = qrp
				md.mutex.Unlock()
			}
		}
	}()
}

// waitForResponse is used by the tests to wait for a response publication. It will eventually timeout if the query fails.
func (md *mockData) waitForResponse() *QueryResponsePublication {
	for count := 0; count < 50; count++ {
		time.Sleep(pollIntervalForTest)
		ret := md.getQueryResponsePublication()
		if ret != nil {
			return ret
		}
	}
	return nil
}

// TestInvalidQueries tests all the obvious reasons why a query may fail (aside from watcher failures).
func TestInvalidQueries(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	var perChainQueries []*PerChainQueryRequest
	var signedQueryRequest *gossipv1.SignedQueryRequest

	// Query with a bad signature should fail.
	md.resetState()
	perChainQueries = []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, _ = createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	signedQueryRequest.Signature[0] += 1 // Corrupt the signature.
	md.signedQueryReqWriteC <- signedQueryRequest
	require.Nil(t, md.waitForResponse())

	// Query for an unsupported chain should fail. The supported chains are defined in supportedChains in query.go
	md.resetState()
	perChainQueries = []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDAlgorand, "0x28d9630", 2)}
	signedQueryRequest, _ = createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	md.signedQueryReqWriteC <- signedQueryRequest
	require.Nil(t, md.waitForResponse())

	// Query for a chain that supports queries but that is not in the watcher channel map should fail.
	md.resetState()
	perChainQueries = []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDSepolia, "0x28d9630", 2)}
	signedQueryRequest, _ = createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	md.signedQueryReqWriteC <- signedQueryRequest
	require.Nil(t, md.waitForResponse())
}

func TestSingleEthCallQueryShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Wait until we receive a response or timeout.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestSingleEthCallByTimestampQueryShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCallByTimestamp(t, vaa.ChainIDPolygon, "0x28d9630", "0x28d9631", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Wait until we receive a response or timeout.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestSingleEthCallWithFinalityQueryShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCallWithFinality(t, vaa.ChainIDPolygon, "0x28d9630", "safe", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Wait until we receive a response or timeout.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestBatchOfMultipleQueryTypesShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{
		createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2),
		createPerChainQueryForEthCallByTimestamp(t, vaa.ChainIDBSC, "0x28d9123", "0x28d9124", 3),
		createPerChainQueryForEthCallWithFinality(t, vaa.ChainIDArbitrum, "0x28d9123", "finalized", 3),
	}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Wait until we receive a response or timeout.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDBSC))
	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDArbitrum))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestQueryWithLimitedRetriesShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make it retry a couple of times, but not enough to make it fail.
	retries := 2
	md.setRetries(vaa.ChainIDPolygon, retries)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// The request should eventually succeed.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, retries+1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestQueryWithRetryDueToTimeoutShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make the first per chain query timeout, but the retry should succeed.
	md.setRetries(vaa.ChainIDPolygon, ignoreQuery)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// The request should eventually succeed.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 2, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestQueryWithTooManyRetriesShouldFail(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{
		createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2),
		createPerChainQueryForEthCall(t, vaa.ChainIDBSC, "0x28d9123", 3),
	}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make polygon retry a couple of times, but not enough to make it fail.
	retriesForPolygon := 2
	md.setRetries(vaa.ChainIDPolygon, retriesForPolygon)

	// Make BSC retry so many times that the request times out.
	md.setRetries(vaa.ChainIDBSC, 1000)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// The request should timeout.
	queryResponsePublication := md.waitForResponse()
	require.Nil(t, queryResponsePublication)

	assert.Equal(t, retriesForPolygon+1, md.getRequestsPerChain(vaa.ChainIDPolygon))
}

func TestQueryWithLimitedRetriesOnMultipleChainsShouldSucceed(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{
		createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2),
		createPerChainQueryForEthCall(t, vaa.ChainIDBSC, "0x28d9123", 3),
	}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make both chains retry a couple of times, but not enough to make it fail.
	retriesForPolygon := 2
	md.setRetries(vaa.ChainIDPolygon, retriesForPolygon)

	retriesForBSC := 3
	md.setRetries(vaa.ChainIDBSC, retriesForBSC)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// The request should eventually succeed.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, retriesForPolygon+1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.Equal(t, retriesForBSC+1, md.getRequestsPerChain(vaa.ChainIDBSC))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestFatalErrorOnPerChainQueryShouldCauseRequestToFail(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{
		createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2),
		createPerChainQueryForEthCall(t, vaa.ChainIDBSC, "0x28d9123", 3),
	}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make BSC return a fatal error.
	md.setRetries(vaa.ChainIDBSC, fatalError)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// The request should timeout.
	queryResponsePublication := md.waitForResponse()
	require.Nil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDBSC))
}

func TestPublishRetrySucceeds(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTestWithoutPublisher(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results. Give the expected results to the mock.
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Submit the query request to the handler.
	md.signedQueryReqWriteC <- signedQueryRequest

	// Sleep for a bit before we start listening for published results.
	// If you look in the log, you should see one of these: "failed to publish query response to p2p, will retry publishing next interval"
	// and at least one of these: "resend of query response to p2p failed again, will keep retrying".
	time.Sleep(retryIntervalForTest * 3)

	// Now start the publisher routine.
	// If you look in the log, you should see one of these: "resend of query response to p2p succeeded".
	md.startResponseListener(ctx)

	// The response should still get published.
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication)

	assert.Equal(t, 1, md.getRequestsPerChain(vaa.ChainIDPolygon))
	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

func TestPerChainConfigValid(t *testing.T) {
	for chainID, config := range perChainConfig {
		if config.NumWorkers <= 0 {
			assert.Equal(t, "", fmt.Sprintf(`perChainConfig for "%s" has an invalid NumWorkers: %d`, chainID.String(), config.NumWorkers))
		}
	}
}
