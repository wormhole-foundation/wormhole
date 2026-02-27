package query

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

const (
	// RequestTimeout indicates how long before a request is considered to have timed out.
	RequestTimeout = 1 * time.Minute

	// RetryInterval specifies how long we will wait between retry intervals.
	RetryInterval = 10 * time.Second

	// AuditInterval specifies how often to audit the list of pending queries.
	AuditInterval = time.Second

	// SignedQueryRequestChannelSize is the buffer size of the incoming query request channel.
	SignedQueryRequestChannelSize = 500

	// QueryRequestBufferSize is the buffer size of the per-network query request channel.
	QueryRequestBufferSize = 250

	// QueryResponseBufferSize is the buffer size of the single query response channel from the watchers.
	QueryResponseBufferSize = 500

	// QueryResponsePublicationChannelSize is the buffer size of the single query response channel back to the P2P publisher.
	QueryResponsePublicationChannelSize = 500
)

var (
	// secp256k1N is the order of the secp256k1 curve
	secp256k1N = ethCrypto.S256().Params().N
	// secp256k1HalfN is half the order of the secp256k1 curve.
	// Used to prevent signature malleability by enforcing s ≤ n/2.
	secp256k1HalfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

func NewQueryHandler(
	logger *zap.Logger,
	env common.Environment,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *PerChainQueryInternal,
	queryResponseReadC <-chan *PerChainQueryResponseInternal,
	queryResponseWriteC chan<- *QueryResponsePublication,
) *QueryHandler {
	return &QueryHandler{
		logger:              logger.With(zap.String("component", "ccq")),
		env:                 env,
		signedQueryReqC:     signedQueryReqC,
		chainQueryReqC:      chainQueryReqC,
		queryResponseReadC:  queryResponseReadC,
		queryResponseWriteC: queryResponseWriteC,
	}
}

// RecoverQueryRequestSigner recovers the Ethereum address from a Wormhole Query signature.
// Wormhole queries use raw ECDSA signatures
func RecoverQueryRequestSigner(digest, signature []byte) (ethCommon.Address, error) {
	if len(signature) != 65 {
		return ethCommon.Address{}, fmt.Errorf("signature must be 65 bytes, got %d", len(signature))
	}

	// Copy the signature because some libraries modify it in-place
	sig := make([]byte, len(signature))
	copy(sig, signature)

	// Validate recovery ID (v) is in the valid range: 0, 1, 27, or 28
	v := sig[64]
	if v != 0 && v != 1 && v != 27 && v != 28 {
		return ethCommon.Address{}, fmt.Errorf("invalid signature recovery ID: must be 0, 1, 27, or 28, got %d", v)
	}

	// Validate s value to prevent signature malleability.
	// ECDSA signatures have malleability: for a valid (r,s), the signature (r, -s mod n) is also valid.
	// Since signature is used as part of the requestID, we must enforce canonical form by requiring s ≤ n/2.
	s := new(big.Int).SetBytes(sig[32:64])
	if s.Cmp(secp256k1HalfN) > 0 {
		return ethCommon.Address{}, fmt.Errorf("invalid signature: s value must be in lower half of curve order to prevent malleability")
	}

	// Normalize 27/28 to 0/1 for go-ethereum's Ecrecover
	if sig[64] == 27 || sig[64] == 28 {
		sig[64] -= 27
	}

	// Recover the public key from the raw signature
	pubkey, err := ethCrypto.Ecrecover(digest, sig)
	if err != nil {
		return ethCommon.Address{}, fmt.Errorf("failed to recover public key from signature: %w", err)
	}

	address := ethCommon.BytesToAddress(ethCrypto.Keccak256(pubkey[1:])[12:])
	return address, nil
}

// RecoverPrefixedSigner recovers the signer from an EIP-191 personal_sign signature.
// Browser wallets add "\x19Ethereum Signed Message:\n{len}" before signing.
func RecoverPrefixedSigner(digest, signature []byte) (ethCommon.Address, error) {
	// Recreate what the wallet signed: keccak256("\x19Ethereum Signed Message:\n" + len + message)
	prefixed := ethCrypto.Keccak256(
		fmt.Appendf(nil, "\x19Ethereum Signed Message:\n%d", len(digest)),
		digest,
	)
	// Now recover using the same hash the wallet used
	return RecoverQueryRequestSigner(prefixed, signature)
}

type (
	// Watcher is the interface that any watcher that supports cross chain queries must implement.
	Watcher interface {
		QueryHandler(ctx context.Context, queryRequest *PerChainQueryInternal)
	}

	// QueryHandler defines the cross chain query handler.
	QueryHandler struct {
		logger              *zap.Logger
		env                 common.Environment
		signedQueryReqC     <-chan *gossipv1.SignedQueryRequest
		chainQueryReqC      map[vaa.ChainID]chan *PerChainQueryInternal
		queryResponseReadC  <-chan *PerChainQueryResponseInternal
		queryResponseWriteC chan<- *QueryResponsePublication
	}

	// pendingQuery is the cache entry for a given query.
	pendingQuery struct {
		signedRequest *gossipv1.SignedQueryRequest
		request       *QueryRequest
		requestID     string
		receiveTime   time.Time
		queries       []*perChainQuery
		responses     []*PerChainQueryResponseInternal

		// respPub is only populated when we need to retry sending the response to p2p.
		respPub *QueryResponsePublication
	}

	// perChainQuery is the data associated with a single per chain query in a query request.
	perChainQuery struct {
		req            *PerChainQueryInternal
		channel        chan *PerChainQueryInternal
		lastUpdateTime time.Time
	}

	PerChainConfig struct {
		TimestampCacheSupported bool
		NumWorkers              int
	}
)

// perChainConfig provides static config info for each chain. If a chain is not listed here, then it does not support queries.
// Every chain listed here must have at least one worker specified.
var perChainConfig = map[vaa.ChainID]PerChainConfig{
	vaa.ChainIDSolana:          {NumWorkers: 10, TimestampCacheSupported: false},
	vaa.ChainIDEthereum:        {NumWorkers: 5, TimestampCacheSupported: true},
	vaa.ChainIDBSC:             {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDPolygon:         {NumWorkers: 5, TimestampCacheSupported: true},
	vaa.ChainIDAvalanche:       {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDFantom:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDKlaytn:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDCelo:            {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMoonbeam:        {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDArbitrum:        {NumWorkers: 5, TimestampCacheSupported: true},
	vaa.ChainIDOptimism:        {NumWorkers: 5, TimestampCacheSupported: true},
	vaa.ChainIDBase:            {NumWorkers: 5, TimestampCacheSupported: true},
	vaa.ChainIDScroll:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMantle:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDXLayer:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDLinea:           {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDBerachain:       {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDUnichain:        {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDWorldchain:      {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDInk:             {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDSepolia:         {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDHolesky:         {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDArbitrumSepolia: {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDBaseSepolia:     {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDOptimismSepolia: {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDPolygonSepolia:  {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDHyperEVM:        {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMonad:           {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDSeiEVM:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMezo:            {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDFogo:            {NumWorkers: 10, TimestampCacheSupported: true},
	vaa.ChainIDConverge:        {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDPlume:           {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDXRPLEVM:         {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDPlasma:          {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDCreditCoin:      {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMoca:            {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMegaETH:         {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDMonadTestnet:    {NumWorkers: 1, TimestampCacheSupported: true},
	vaa.ChainIDZeroGravity:     {NumWorkers: 1, TimestampCacheSupported: true},
}

// GetPerChainConfig returns the config for the specified chain. If the chain is not configured it returns an empty struct,
// which is not an error. It just means that queries are not supported for that chain.
func GetPerChainConfig(chainID vaa.ChainID) PerChainConfig {
	if pcc, exists := perChainConfig[chainID]; exists {
		return pcc
	}
	return PerChainConfig{}
}

// QueriesSupported can be used by the watcher to determine if queries are supported for the chain.
func (config PerChainConfig) QueriesSupported() bool {
	return config.NumWorkers > 0
}

// Start initializes the query handler and starts the runnable.
func (qh *QueryHandler) Start(ctx context.Context) error {
	qh.logger.Debug("entering Start")
	if err := supervisor.Run(ctx, "query_handler", common.WrapWithScissors(qh.handleQueryRequests, "query_handler")); err != nil {
		return fmt.Errorf("failed to start query handler routine: %w", err)
	}
	return nil
}

// handleQueryRequests multiplexes observation requests to the appropriate chain
func (qh *QueryHandler) handleQueryRequests(ctx context.Context) error {
	return handleQueryRequestsImpl(
		ctx,
		qh.logger,
		qh.signedQueryReqC,
		qh.chainQueryReqC,
		qh.queryResponseReadC,
		qh.queryResponseWriteC,
		qh.env,
		RequestTimeout,
		RetryInterval,
		AuditInterval)
}

// handleQueryRequestsImpl allows instantiating the handler in the test environment with shorter timeout and retry parameters.
func handleQueryRequestsImpl(
	ctx context.Context,
	logger *zap.Logger,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *PerChainQueryInternal,
	queryResponseReadC <-chan *PerChainQueryResponseInternal,
	queryResponseWriteC chan<- *QueryResponsePublication,
	env common.Environment,
	requestTimeoutImpl time.Duration,
	retryIntervalImpl time.Duration,
	auditIntervalImpl time.Duration,
) error {
	qLogger := logger.With(zap.String("component", "ccqhandler"))
	qLogger.Info("cross chain queries are enabled", zap.String("env", string(env)))

	pendingQueries := make(map[string]*pendingQuery) // Key is requestID.

	// Create the set of chains for which CCQ is actually enabled. Those are the ones in the config for which we actually have a watcher enabled.
	supportedChains := make(map[vaa.ChainID]struct{})
	for chainID, config := range perChainConfig {
		if _, exists := chainQueryReqC[chainID]; exists {
			if config.NumWorkers <= 0 {
				panic(fmt.Sprintf(`invalid per chain config entry for "%s", no workers specified`, chainID.String()))
			}
			logger.Info("queries supported on chain", zap.Stringer("chainID", chainID), zap.Int("numWorkers", config.NumWorkers))
			supportedChains[chainID] = struct{}{}

			// Make sure we have a metric for every enabled chain, so we can see which ones are actually enabled.
			totalRequestsByChain.WithLabelValues(chainID.String()).Add(0)
		}
	}

	ticker := time.NewTicker(auditIntervalImpl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case signedRequest := <-signedQueryReqC: // Inbound query request.
			allQueryRequestsReceived.Inc()

			digest := QueryRequestDigest(env, signedRequest.QueryRequest)
			requestID := hex.EncodeToString(signedRequest.Signature) + ":" + digest.String()

			qLogger.Info("received a query request", zap.String("requestID", requestID))

			signerAddress, err := RecoverQueryRequestSigner(digest.Bytes(), signedRequest.Signature)
			if err != nil {
				qLogger.Error("failed to recover signer",
					zap.String("requestID", requestID),
					zap.Error(err),
				)
				invalidQueryRequestReceived.WithLabelValues("failed_to_recover_signer").Inc()
				continue
			}

			qLogger.Info("signer recovered",
				zap.String("requestID", requestID),
				zap.String("signer", signerAddress.Hex()),
			)

			// Make sure this is not a duplicate request.
			if oldReq, exists := pendingQueries[requestID]; exists {
				qLogger.Warn("dropping duplicate query request", zap.String("requestID", requestID), zap.Stringer("origRecvTime", oldReq.receiveTime))
				invalidQueryRequestReceived.WithLabelValues("duplicate_request").Inc()
				continue
			}

			// Unmarshal and validate the query request
			var queryRequest QueryRequest
			err = queryRequest.Unmarshal(signedRequest.QueryRequest)
			if err != nil {
				qLogger.Error("failed to unmarshal query request", zap.String("requestor", signerAddress.Hex()), zap.String("requestID", requestID), zap.Error(err))
				invalidQueryRequestReceived.WithLabelValues("failed_to_unmarshal_request").Inc()
				continue
			}

			if err := queryRequest.Validate(); err != nil {
				qLogger.Error("received invalid message", zap.String("requestor", signerAddress.Hex()), zap.String("requestID", requestID), zap.Error(err))
				invalidQueryRequestReceived.WithLabelValues("invalid_request").Inc()
				continue
			}

			// Build the set of per chain queries and placeholders for the per chain responses.
			queries := []*perChainQuery{}
			responses := make([]*PerChainQueryResponseInternal, len(queryRequest.PerChainQueries))
			receiveTime := time.Now()

			if errorFound := func() bool {
				for requestIdx, pcq := range queryRequest.PerChainQueries {
					chainID := vaa.ChainID(pcq.ChainId)
					if _, exists := supportedChains[chainID]; !exists {
						qLogger.Debug("chain does not support cross chain queries", zap.String("requestID", requestID), zap.Stringer("chainID", chainID))
						invalidQueryRequestReceived.WithLabelValues("chain_does_not_support_ccq").Inc()
						return true
					}

					channel, channelExists := chainQueryReqC[chainID]
					if !channelExists {
						qLogger.Debug("unknown chain ID for query request, dropping it", zap.String("requestID", requestID), zap.Stringer("chain_id", chainID))
						invalidQueryRequestReceived.WithLabelValues("failed_to_look_up_channel").Inc()
						return true
					}

					queries = append(queries, &perChainQuery{
						req: &PerChainQueryInternal{
							RequestID:  requestID,
							RequestIdx: requestIdx,
							Request:    pcq,
						},
						channel: channel,
					})
				}
				return false
			}(); errorFound {
				continue
			}

			validQueryRequestsReceived.Inc()

			// Create the pending query and add it to the cache.
			pq := &pendingQuery{
				signedRequest: signedRequest,
				request:       &queryRequest,
				requestID:     requestID,
				receiveTime:   receiveTime,
				queries:       queries,
				responses:     responses,
			}
			pendingQueries[requestID] = pq

			// Forward the requests to the watchers.
			for _, pcq := range pq.queries {
				pcq.ccqForwardToWatcher(qLogger, pq.receiveTime)
			}

		case resp := <-queryResponseReadC: // Response from a watcher.
			if resp.Status == QuerySuccess {
				successfulQueryResponsesReceivedByChain.WithLabelValues(resp.ChainId.String()).Inc()
				if resp.Response == nil {
					qLogger.Error("received a successful query response with no results, dropping it!", zap.String("requestID", resp.RequestID))
					continue
				}

				pq, exists := pendingQueries[resp.RequestID]
				if !exists {
					qLogger.Warn("received a success response with no outstanding query, dropping it", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
					continue
				}

				if resp.RequestIdx >= len(pq.responses) {
					qLogger.Error("received a response with an invalid index", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
					continue
				}

				// Store the result, which will mark this per-chain query as completed.
				pq.responses[resp.RequestIdx] = resp

				// If we still have other outstanding per chain queries for this request, keep waiting.
				numStillPending := pq.numPendingRequests()
				if numStillPending > 0 {
					qLogger.Info("received a per chain query response, still waiting for more", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx), zap.Int("numStillPending", numStillPending))
					continue
				} else {
					qLogger.Info("received final per chain query response, ready to publish", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
				}

				// Build the list of per chain response publications and the overall query response publication.
				responses := []*PerChainQueryResponse{}
				for _, pqResp := range pq.responses {
					if pqResp == nil {
						qLogger.Error("unexpected nil response in pending query!", zap.String("requestID", resp.RequestID))
						continue
					}

					responses = append(responses, &PerChainQueryResponse{
						ChainId:  pqResp.ChainId,
						Response: pqResp.Response,
					})
				}

				respPub := &QueryResponsePublication{
					Request:           pq.signedRequest,
					PerChainResponses: responses,
				}

				// Send the response to be published.
				select {
				case queryResponseWriteC <- respPub:
					qLogger.Info("forwarded query response to p2p", zap.String("requestID", resp.RequestID))
					queryResponsesPublished.Inc()
					delete(pendingQueries, resp.RequestID)
				default:
					qLogger.Warn("failed to publish query response to p2p, will retry publishing next interval", zap.String("requestID", resp.RequestID))
					pq.respPub = respPub
				}
			} else if resp.Status == QueryRetryNeeded {
				retryNeededQueryResponsesReceivedByChain.WithLabelValues(resp.ChainId.String()).Inc()
				if _, exists := pendingQueries[resp.RequestID]; exists {
					qLogger.Warn("query failed, will retry next interval", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
				} else {
					qLogger.Warn("received a retry needed response with no outstanding query, dropping it", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
				}
			} else if resp.Status == QueryFatalError {
				fatalQueryResponsesReceivedByChain.WithLabelValues(resp.ChainId.String()).Inc()
				qLogger.Error("received a fatal error response, dropping the whole request", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
				delete(pendingQueries, resp.RequestID)
			} else {
				qLogger.Error("received an unexpected query status, dropping the whole request", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx), zap.Int("status", int(resp.Status)))
				delete(pendingQueries, resp.RequestID)
			}

		case <-ticker.C: // Retry audit timer.
			now := time.Now()
			for reqId, pq := range pendingQueries {
				timeout := pq.receiveTime.Add(requestTimeoutImpl)
				qLogger.Debug("audit", zap.String("requestId", reqId), zap.Stringer("receiveTime", pq.receiveTime), zap.Stringer("timeout", timeout))
				if timeout.Before(now) {
					qLogger.Debug("query request timed out, dropping it", zap.String("requestId", reqId), zap.Stringer("receiveTime", pq.receiveTime))
					queryRequestsTimedOut.Inc()
					delete(pendingQueries, reqId)
				} else {
					if pq.respPub != nil {
						// Resend the response to be published.
						select {
						case queryResponseWriteC <- pq.respPub:
							qLogger.Info("resend of query response to p2p succeeded", zap.String("requestID", reqId))
							queryResponsesPublished.Inc()
							delete(pendingQueries, reqId)
						default:
							qLogger.Warn("resend of query response to p2p failed again, will keep retrying", zap.String("requestID", reqId))
						}
					} else {
						for requestIdx, pcq := range pq.queries {
							if pq.responses[requestIdx] == nil && pcq.lastUpdateTime.Add(retryIntervalImpl).Before(now) {
								qLogger.Info("retrying query request",
									zap.String("requestId", reqId),
									zap.Int("requestIdx", requestIdx),
									zap.Stringer("receiveTime", pq.receiveTime),
									zap.Stringer("lastUpdateTime", pcq.lastUpdateTime),
									zap.String("chainID", pq.queries[requestIdx].req.Request.ChainId.String()),
								)
								pcq.ccqForwardToWatcher(qLogger, now)
							}
						}
					}
				}
			}
		}
	}
}

// ccqForwardToWatcher submits a query request to the appropriate watcher. It updates the request object if the write succeeds.
// If the write fails, it does not update the last update time, which will cause a retry next interval (until it times out)
func (pcq *perChainQuery) ccqForwardToWatcher(qLogger *zap.Logger, receiveTime time.Time) {
	select {
	// TODO: only send the query request itself and reassemble in this module
	case pcq.channel <- pcq.req:
		qLogger.Debug("forwarded query request to watcher", zap.String("requestID", pcq.req.RequestID), zap.Stringer("chainID", pcq.req.Request.ChainId))
		totalRequestsByChain.WithLabelValues(pcq.req.Request.ChainId.String()).Inc()
	default:
		qLogger.Warn("failed to send query request to watcher, will retry next interval", zap.String("requestID", pcq.req.RequestID), zap.Stringer("chain_id", pcq.req.Request.ChainId))
	}
	pcq.lastUpdateTime = receiveTime
}

// numPendingRequests returns the number of per chain queries in a request that are still awaiting responses. Zero means the request can now be published.
func (pq *pendingQuery) numPendingRequests() int {
	numPending := 0
	for _, resp := range pq.responses {
		if resp == nil {
			numPending += 1
		}
	}

	return numPending
}

// StartWorkers is used by the watchers to start the query handler worker routines.
func StartWorkers(
	ctx context.Context,
	logger *zap.Logger,
	errC chan error,
	w Watcher,
	queryReqC <-chan *PerChainQueryInternal,
	config PerChainConfig,
	tag string,
) {
	for count := 0; count < config.NumWorkers; count++ {
		workerId := count
		common.RunWithScissors(ctx, errC, fmt.Sprintf("%s_fetch_query_req", tag), func(ctx context.Context) error {
			logger.Debug("CONCURRENT: starting worker", zap.Int("worker", workerId))
			for {
				select {
				case <-ctx.Done():
					return nil
				case queryRequest := <-queryReqC:
					logger.Debug("CONCURRENT: processing query request", zap.Int("worker", workerId))
					w.QueryHandler(ctx, queryRequest)
					logger.Debug("CONCURRENT: finished processing query request", zap.Int("worker", workerId))
				}
			}
		})
	}
}
