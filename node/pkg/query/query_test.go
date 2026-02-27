package query

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
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

// parseAllowedRequesters parses a comma-separated list of allowed requesters for testing
func parseAllowedRequesters(allowedRequesters string) (map[ethCommon.Address]struct{}, error) {
	if allowedRequesters == "" {
		return nil, fmt.Errorf("allowedRequesters cannot be empty")
	}

	var nullAddr ethCommon.Address
	result := make(map[ethCommon.Address]struct{})
	for _, str := range strings.Split(allowedRequesters, ",") {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		addr := ethCommon.BytesToAddress(ethCommon.Hex2Bytes(strings.TrimPrefix(str, "0x")))
		if addr == nullAddr {
			return nil, fmt.Errorf("invalid address: %s", str)
		}
		result[addr] = struct{}{}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid addresses found")
	}

	return result, nil
}

// createPerChainQueryForEthCall creates a per chain query for an eth_call for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCall(
	t *testing.T,
	chainID vaa.ChainID,
	block string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := range numCalls {
		ethCallData = append(ethCallData, &EthCallData{
			To:   fmt.Appendf(nil, "%-20s", fmt.Sprintf("To for %d:%d", chainID, count)),
			Data: fmt.Appendf(nil, "CallData for %d:%d", chainID, count),
		})
	}

	callRequest := &EthCallQueryRequest{
		BlockId:  block,
		CallData: ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainID,
		Query:   callRequest,
	}
}

// createPerChainQueryForEthCallByTimestamp creates a per chain query for an eth_call_by_timestamp for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCallByTimestamp(
	t *testing.T,
	chainID vaa.ChainID,
	targetBlock string,
	followingBlock string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := range numCalls {
		ethCallData = append(ethCallData, &EthCallData{
			To:   fmt.Appendf(nil, "%-20s", fmt.Sprintf("To for %d:%d", chainID, count)),
			Data: fmt.Appendf(nil, "CallData for %d:%d", chainID, count),
		})
	}

	callRequest := &EthCallByTimestampQueryRequest{
		TargetTimestamp:      1697216322000000,
		TargetBlockIdHint:    targetBlock,
		FollowingBlockIdHint: followingBlock,
		CallData:             ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainID,
		Query:   callRequest,
	}
}

// createPerChainQueryForEthCallWithFinality creates a per chain query for an eth_call_with_finality for use in tests. The To and Data fields are meaningless gibberish, not ABI.
func createPerChainQueryForEthCallWithFinality(
	t *testing.T,
	chainID vaa.ChainID,
	blockId string,
	finality string,
	numCalls int,
) *PerChainQueryRequest {
	t.Helper()
	ethCallData := []*EthCallData{}
	for count := range numCalls {
		ethCallData = append(ethCallData, &EthCallData{
			To:   fmt.Appendf(nil, "%-20s", fmt.Sprintf("To for %d:%d", chainID, count)),
			Data: fmt.Appendf(nil, "CallData for %d:%d", chainID, count),
		})
	}

	callRequest := &EthCallWithFinalityQueryRequest{
		BlockId:  blockId,
		Finality: finality,
		CallData: ethCallData,
	}

	return &PerChainQueryRequest{
		ChainId: chainID,
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
		Timestamp:       uint64(time.Now().Unix()), // #nosec G115 -- time.Now() always returns positive Unix timestamps
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
func (md *mockData) setRetries(chainID vaa.ChainID, count int) {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	md.retriesPerChain[chainID] = count
}

// incrementRequestsPerChainAlreadyLocked is used by the watchers to keep track of how many times they were invoked in a given test.
func (md *mockData) incrementRequestsPerChainAlreadyLocked(chainID vaa.ChainID) {
	if val, exists := md.requestsPerChain[chainID]; exists {
		md.requestsPerChain[chainID] = val + 1
	} else {
		md.requestsPerChain[chainID] = 1
	}
}

// getQueryResponsePublication returns the latest query response publication received by the mock.
func (md *mockData) getQueryResponsePublication() *QueryResponsePublication {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	return md.queryResponsePublication
}

// getRequestsPerChain returns the count of the number of times the given watcher was invoked in a given test.
func (md *mockData) getRequestsPerChain(chainID vaa.ChainID) int {
	md.mutex.Lock()
	defer md.mutex.Unlock()
	if ret, exists := md.requestsPerChain[chainID]; exists {
		return ret
	}
	return 0
}

// shouldIgnoreAlreadyLocked is used by the watchers to see if they should ignore a query (causing a retry).
func (md *mockData) shouldIgnoreAlreadyLocked(chainID vaa.ChainID) bool {
	if val, exists := md.retriesPerChain[chainID]; exists {
		if val == ignoreQuery {
			delete(md.retriesPerChain, chainID)
			return true
		}
	}
	return false
}

// getStatusAlreadyLocked is used by the watchers to determine what query status they should return, based on the `retriesPerChain`.
func (md *mockData) getStatusAlreadyLocked(chainID vaa.ChainID) QueryStatus {
	if val, exists := md.retriesPerChain[chainID]; exists {
		if val == fatalError {
			return QueryFatalError
		}
		val -= 1
		if val > 0 {
			md.retriesPerChain[chainID] = val
		} else {
			delete(md.retriesPerChain, chainID)
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

	// Inbound observation requests from the p2p service (for all chains)
	md.signedQueryReqReadC, md.signedQueryReqWriteC = makeChannelPair[*gossipv1.SignedQueryRequest](SignedQueryRequestChannelSize)

	// Per-chain query requests
	md.chainQueryReqC = make(map[vaa.ChainID]chan *PerChainQueryInternal)
	for _, chainID := range chains {
		md.chainQueryReqC[chainID] = make(chan *PerChainQueryInternal)
	}

	// Query responses from watchers to query handler aggregated across all chains
	md.queryResponseReadC, md.queryResponseWriteC = makeChannelPair[*PerChainQueryResponseInternal](0)

	// Query responses from query handler to p2p
	md.queryResponsePublicationReadC, md.queryResponsePublicationWriteC = makeChannelPair[*QueryResponsePublication](0)

	md.resetState()

	go func() {
		err := handleQueryRequestsImpl(ctx, logger, md.signedQueryReqReadC, md.chainQueryReqC,
			md.queryResponseReadC, md.queryResponsePublicationWriteC, common.GoTest, requestTimeoutForTest, retryIntervalForTest, auditIntervalForTest)
		assert.NoError(t, err)
	}()

	// Create a routine for each configured watcher. It will take a per chain query and return the corresponding expected result.
	// It also pegs a counter of the number of requests the watcher received, for verification purposes.
	for chainID := range md.chainQueryReqC {
		go func(chainID vaa.ChainID, chainQueryReqC <-chan *PerChainQueryInternal) {
			for {
				select {
				case <-ctx.Done():
					return
				case pcqr := <-chainQueryReqC:
					require.Equal(t, chainID, pcqr.Request.ChainId)
					md.mutex.Lock()
					md.incrementRequestsPerChainAlreadyLocked(chainID)
					if md.shouldIgnoreAlreadyLocked(chainID) {
						logger.Info("watcher ignoring query", zap.String("chainID", chainID.String()), zap.Int("requestIdx", pcqr.RequestIdx))
					} else if pcqr.RequestIdx >= len(md.expectedResults) {
						logger.Error("unexpected query reached watcher", zap.String("chainID", chainID.String()), zap.Int("requestIdx", pcqr.RequestIdx), zap.Int("expectedResultsLen", len(md.expectedResults)))
					} else {
						results := md.expectedResults[pcqr.RequestIdx].Response
						status := md.getStatusAlreadyLocked(chainID)
						logger.Info("watcher returning", zap.String("chainID", chainID.String()), zap.Int("requestIdx", pcqr.RequestIdx), zap.Int("status", int(status)))
						queryResponse := CreatePerChainQueryResponseInternal(pcqr.RequestID, pcqr.RequestIdx, pcqr.Request.ChainId, status, results)
						md.queryResponseWriteC <- queryResponse
					}
					md.mutex.Unlock()
				}
			}
		}(chainID, md.chainQueryReqC[chainID])
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
	for range 50 {
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

// ============================================================================

// ============================================================================
// Delegation Tests
// ============================================================================

// TestQueryRequestMarshal
func TestQueryRequestMarshal(t *testing.T) {
	queryRequest := &QueryRequest{
		Nonce:     12345,
		Timestamp: uint64(time.Now().Unix()), // #nosec G115 -- time.Now() always returns positive Unix timestamps
		PerChainQueries: []*PerChainQueryRequest{
			createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2),
		},
	}

	// Marshal
	bytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	// Should use v2 format
	require.Equal(t, MSG_VERSION_V2, bytes[0], "Should use v2 format")

	// Unmarshal
	var unmarshaled QueryRequest
	err = unmarshaled.Unmarshal(bytes)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, queryRequest.Nonce, unmarshaled.Nonce)
	assert.Equal(t, queryRequest.Timestamp, unmarshaled.Timestamp)
}

// ============================================================================
// Signature Format Tests (EIP-191 prefixed signatures)
// ============================================================================

// TestRecoverPrefixedSigner tests that EIP-191 prefixed signatures are correctly recovered
func TestRecoverPrefixedSigner(t *testing.T) {
	sk, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	expectedAddr := ethCrypto.PubkeyToAddress(sk.PublicKey)

	// Create a test message and compute digest
	message := []byte("test query request bytes")
	digest := QueryRequestDigest(common.UnsafeDevNet, message)

	t.Run("prefixed signature recovery succeeds", func(t *testing.T) {
		// Sign with EIP-191 prefix (what personal_sign does)
		prefixedHash := ethCrypto.Keccak256(
			fmt.Appendf(nil, "\x19Ethereum Signed Message:\n%d", len(digest.Bytes())),
			digest.Bytes(),
		)
		sig, err := ethCrypto.Sign(prefixedHash, sk)
		require.NoError(t, err)

		// Recover using RecoverPrefixedSigner
		recovered, err := RecoverPrefixedSigner(digest.Bytes(), sig)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered)
	})

	t.Run("raw signature recovery still works", func(t *testing.T) {
		// Sign with raw ECDSA (no prefix)
		sig, err := ethCrypto.Sign(digest.Bytes(), sk)
		require.NoError(t, err)

		// Recover using RecoverQueryRequestSigner
		recovered, err := RecoverQueryRequestSigner(digest.Bytes(), sig)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered)
	})

	t.Run("wrong recovery method returns different address", func(t *testing.T) {
		// Sign with raw ECDSA
		sig, err := ethCrypto.Sign(digest.Bytes(), sk)
		require.NoError(t, err)

		// Try to recover as prefixed - should get wrong address
		recovered, err := RecoverPrefixedSigner(digest.Bytes(), sig)
		require.NoError(t, err)
		assert.NotEqual(t, expectedAddr, recovered, "Wrong recovery method should produce different address")
	})
}

// TestRecoverQueryRequestSigner_RecoveryIDValidation tests strict validation of signature recovery IDs
func TestRecoverQueryRequestSigner_RecoveryIDValidation(t *testing.T) {
	sk, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	expectedAddr := ethCrypto.PubkeyToAddress(sk.PublicKey)

	// Create a test message and compute digest
	message := []byte("test query request")
	digest := QueryRequestDigest(common.UnsafeDevNet, message)

	t.Run("valid recovery IDs are accepted", func(t *testing.T) {
		// Generate a valid signature (go-ethereum produces v=0 or v=1)
		sig, err := ethCrypto.Sign(digest.Bytes(), sk)
		require.NoError(t, err)
		require.Len(t, sig, 65)

		// Test with recovery ID = 0 or 1 (as produced by go-ethereum)
		originalV := sig[64]
		require.True(t, originalV == 0 || originalV == 1, "go-ethereum should produce v=0 or v=1")

		recovered, err := RecoverQueryRequestSigner(digest.Bytes(), sig)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered)

		// Test that v and v+27 are equivalent (normalization works correctly)
		// v=0 should be equivalent to v=27, and v=1 should be equivalent to v=28
		sigNormalized := make([]byte, 65)
		copy(sigNormalized, sig)
		sigNormalized[64] = originalV + 27

		recovered, err = RecoverQueryRequestSigner(digest.Bytes(), sigNormalized)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered, "v=%d and v=%d should recover the same address", originalV, originalV+27)
	})

	t.Run("invalid recovery IDs are rejected", func(t *testing.T) {
		// Generate a valid signature
		sig, err := ethCrypto.Sign(digest.Bytes(), sk)
		require.NoError(t, err)

		// Test invalid recovery IDs
		invalidIDs := []byte{2, 3, 26, 29, 30, 100, 255}

		for _, invalidV := range invalidIDs {
			invalidSig := make([]byte, 65)
			copy(invalidSig, sig)
			invalidSig[64] = invalidV

			_, err := RecoverQueryRequestSigner(digest.Bytes(), invalidSig)
			assert.Error(t, err, "recovery ID %d should be rejected", invalidV)
			assert.Contains(t, err.Error(), "invalid signature recovery ID", "error message should mention invalid recovery ID")
			assert.Contains(t, err.Error(), fmt.Sprintf("got %d", invalidV), "error message should include the invalid value")
		}
	})

	t.Run("signature length validation", func(t *testing.T) {
		// Test too short
		shortSig := make([]byte, 64)
		_, err := RecoverQueryRequestSigner(digest.Bytes(), shortSig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature must be 65 bytes")

		// Test too long
		longSig := make([]byte, 66)
		_, err = RecoverQueryRequestSigner(digest.Bytes(), longSig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature must be 65 bytes")
	})
}

// TestDuplicateRequestIsDropped tests that duplicate query requests (same signature + payload)
// are dropped while the original request is still pending
func TestDuplicateRequestIsDropped(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	md := createQueryHandlerForTest(t, ctx, logger, watcherChainsForTest)

	// Create the request and the expected results
	perChainQueries := []*PerChainQueryRequest{createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)}
	signedQueryRequest, queryRequest := createSignedQueryRequestForTesting(t, md.sk, perChainQueries)
	expectedResults := createExpectedResultsForTest(t, queryRequest.PerChainQueries)
	md.setExpectedResults(expectedResults)

	// Make the watcher delay before responding so the request stays pending
	md.setRetries(vaa.ChainIDPolygon, ignoreQuery)

	// Submit the first query request
	md.signedQueryReqWriteC <- signedQueryRequest

	// Give the handler time to process the first request and add it to pendingQueries
	time.Sleep(pollIntervalForTest * 2)

	// Submit the exact same query request again (duplicate) while the first is still pending
	md.signedQueryReqWriteC <- signedQueryRequest

	// Give the handler time to process the duplicate
	time.Sleep(pollIntervalForTest * 2)

	// Now let the first request complete by clearing the retries
	md.mutex.Lock()
	delete(md.retriesPerChain, vaa.ChainIDPolygon)
	md.mutex.Unlock()

	// Wait for the first response - should succeed
	queryResponsePublication := md.waitForResponse()
	require.NotNil(t, queryResponsePublication, "First request should succeed")

	// Verify the watcher was called twice (once for original, once for retry after ignoreQuery)
	// but NOT called a third time for the duplicate
	requestCount := md.getRequestsPerChain(vaa.ChainIDPolygon)
	assert.Equal(t, 2, requestCount, "Watcher should be called twice (original + retry), not for duplicate")

	assert.True(t, validateResponseForTest(t, queryResponsePublication, signedQueryRequest, queryRequest, expectedResults))
}

// TestNonCanonicalQueryLengthRejected tests that query requests with incorrect length fields are rejected
func TestNonCanonicalQueryLengthRejected(t *testing.T) {
	// Create a valid query request and marshal it
	perChainQuery := createPerChainQueryForEthCall(t, vaa.ChainIDPolygon, "0x28d9630", 2)
	validBytes, err := perChainQuery.Marshal()
	require.NoError(t, err)

	// Corrupt the length field (bytes 0-1 contain chainId, byte 2 contains queryType, bytes 3-6 contain length)
	corruptedBytes := make([]byte, len(validBytes))
	copy(corruptedBytes, validBytes)

	// Change the length field to an incorrect value (add 10 to the length)
	// The length is at position 3 (after 2 bytes chainId + 1 byte queryType)
	originalLength := uint32(corruptedBytes[3])<<24 | uint32(corruptedBytes[4])<<16 |
		uint32(corruptedBytes[5])<<8 | uint32(corruptedBytes[6])
	incorrectLength := originalLength + 10
	corruptedBytes[3] = byte(incorrectLength >> 24)
	corruptedBytes[4] = byte(incorrectLength >> 16)
	corruptedBytes[5] = byte(incorrectLength >> 8)
	corruptedBytes[6] = byte(incorrectLength)

	// Try to unmarshal - should fail with length mismatch error
	var perChainQuery2 PerChainQueryRequest
	err = perChainQuery2.Unmarshal(corruptedBytes)
	require.Error(t, err, "Non-canonical query with incorrect length should be rejected")
	assert.Contains(t, err.Error(), "query length mismatch",
		"Error should indicate length mismatch")
}

// TestExcessiveLengthFieldsRejected tests that queries with excessive length fields are rejected before allocation
func TestExcessiveLengthFieldsRejected(t *testing.T) {
	t.Run("excessive block id length", func(t *testing.T) {
		// Create a buffer with a valid structure but excessive blockIdLen
		buf := new(bytes.Buffer)
		// Write chainId (2 bytes)
		vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDPolygon)
		// Write queryType (1 byte)
		vaa.MustWrite(buf, binary.BigEndian, uint8(EthCallQueryRequestType))
		// Write queryLength (4 bytes) - we'll validate this separately
		vaa.MustWrite(buf, binary.BigEndian, uint32(1000))
		// Write excessive blockIdLen (4 bytes) - should trigger validation
		vaa.MustWrite(buf, binary.BigEndian, uint32(MAX_BLOCK_ID_LEN+1))

		var pcq PerChainQueryRequest
		err := pcq.Unmarshal(buf.Bytes())
		require.Error(t, err, "Excessive block id length should be rejected")
		assert.Contains(t, err.Error(), "exceeds maximum", "Error should mention exceeds maximum")
	})

	t.Run("excessive call data length", func(t *testing.T) {
		// Create a minimal EthCallQueryRequest with excessive dataLen
		buf := new(bytes.Buffer)
		// Write chainId
		vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDPolygon)
		// Write queryType
		vaa.MustWrite(buf, binary.BigEndian, uint8(EthCallQueryRequestType))
		// Write queryLength placeholder (will cause length mismatch but that's ok for this test)
		vaa.MustWrite(buf, binary.BigEndian, uint32(1000))
		// Write blockIdLen
		vaa.MustWrite(buf, binary.BigEndian, uint32(2))
		// Write blockId
		buf.Write([]byte("0x"))
		// Write numCallData
		vaa.MustWrite(buf, binary.BigEndian, uint8(1))
		// Write To address (20 bytes)
		buf.Write(make([]byte, 20))
		// Write excessive dataLen - should trigger validation
		vaa.MustWrite(buf, binary.BigEndian, uint32(MAX_CALL_DATA_LEN+1))

		var pcq PerChainQueryRequest
		err := pcq.Unmarshal(buf.Bytes())
		require.Error(t, err, "Excessive call data length should be rejected")
		assert.Contains(t, err.Error(), "exceeds maximum", "Error should mention exceeds maximum")
	})

	t.Run("excessive finality length", func(t *testing.T) {
		// Create a minimal EthCallWithFinalityQueryRequest with excessive finalityLen
		buf := new(bytes.Buffer)
		// Write chainId
		vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDPolygon)
		// Write queryType
		vaa.MustWrite(buf, binary.BigEndian, uint8(EthCallWithFinalityQueryRequestType))
		// Write queryLength placeholder
		vaa.MustWrite(buf, binary.BigEndian, uint32(1000))
		// Write blockIdLen
		vaa.MustWrite(buf, binary.BigEndian, uint32(2))
		// Write blockId
		buf.Write([]byte("0x"))
		// Write excessive finalityLen - should trigger validation
		vaa.MustWrite(buf, binary.BigEndian, uint32(MAX_FINALITY_LEN+1))

		var pcq PerChainQueryRequest
		err := pcq.Unmarshal(buf.Bytes())
		require.Error(t, err, "Excessive finality length should be rejected")
		assert.Contains(t, err.Error(), "exceeds maximum", "Error should mention exceeds maximum")
	})
}
