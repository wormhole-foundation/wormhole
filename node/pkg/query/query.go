package query

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"
)

const (
	// RequestTimeout indicates how long before a request is considered to have timed out.
	RequestTimeout = 1 * time.Minute

	// RetryInterval specifies how long we will wait between retry intervals. This is the interval of our ticker.
	RetryInterval = 10 * time.Second

	// AuditInterval specifies how often to audit the list of pending queries.
	AuditInterval = time.Second

	// SignedQueryRequestChannelSize is the buffer size of the incoming query request channel.
	SignedQueryRequestChannelSize = 50

	// QueryRequestBufferSize is the buffer size of the per-network query request channel.
	QueryRequestBufferSize = 25
)

func NewQueryHandler(
	logger *zap.Logger,
	env common.Environment,
	allowedRequestorsStr string,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *PerChainQueryInternal,
	queryResponseReadC <-chan *PerChainQueryResponseInternal,
	queryResponseWriteC chan<- *QueryResponsePublication,
) *QueryHandler {
	return &QueryHandler{
		logger:               logger.With(zap.String("component", "ccq")),
		env:                  env,
		allowedRequestorsStr: allowedRequestorsStr,
		signedQueryReqC:      signedQueryReqC,
		chainQueryReqC:       chainQueryReqC,
		queryResponseReadC:   queryResponseReadC,
		queryResponseWriteC:  queryResponseWriteC,
	}
}

type (
	// QueryHandler defines the cross chain query handler.
	QueryHandler struct {
		logger               *zap.Logger
		env                  common.Environment
		allowedRequestorsStr string
		signedQueryReqC      <-chan *gossipv1.SignedQueryRequest
		chainQueryReqC       map[vaa.ChainID]chan *PerChainQueryInternal
		queryResponseReadC   <-chan *PerChainQueryResponseInternal
		queryResponseWriteC  chan<- *QueryResponsePublication
		allowedRequestors    map[ethCommon.Address]struct{}
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
)

// Start initializes the query handler and starts the runnable.
func (qh *QueryHandler) Start(ctx context.Context) error {
	qh.logger.Debug("entering Start", zap.String("enforceFlag", qh.allowedRequestorsStr))

	var err error
	qh.allowedRequestors, err = parseAllowedRequesters(qh.allowedRequestorsStr)
	if err != nil {
		return fmt.Errorf("failed to parse allowed requesters: %w", err)
	}

	if err := supervisor.Run(ctx, "query_handler", common.WrapWithScissors(qh.handleQueryRequests, "query_handler")); err != nil {
		return fmt.Errorf("failed to start query handler routine: %w", err)
	}

	return nil
}

// handleQueryRequests multiplexes observation requests to the appropriate chain
func (qh *QueryHandler) handleQueryRequests(ctx context.Context) error {
	return handleQueryRequestsImpl(ctx, qh.logger, qh.signedQueryReqC, qh.chainQueryReqC, qh.allowedRequestors, qh.queryResponseReadC, qh.queryResponseWriteC, qh.env, RequestTimeout, RetryInterval, AuditInterval)
}

// handleQueryRequestsImpl allows instantiating the handler in the test environment with shorter timeout and retry parameters.
func handleQueryRequestsImpl(
	ctx context.Context,
	logger *zap.Logger,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *PerChainQueryInternal,
	allowedRequestors map[ethCommon.Address]struct{},
	queryResponseReadC <-chan *PerChainQueryResponseInternal,
	queryResponseWriteC chan<- *QueryResponsePublication,
	env common.Environment,
	requestTimeoutImpl time.Duration,
	retryIntervalImpl time.Duration,
	auditIntervalImpl time.Duration,
) error {
	qLogger := logger.With(zap.String("component", "ccqhandler"))
	qLogger.Info("cross chain queries are enabled", zap.Any("allowedRequestors", allowedRequestors), zap.String("env", string(env)))

	pendingQueries := make(map[string]*pendingQuery) // Key is requestID.

	// CCQ is currently only supported on EVM and Solana.
	supportedChains := map[vaa.ChainID]struct{}{
		vaa.ChainIDSolana:          {},
		vaa.ChainIDEthereum:        {},
		vaa.ChainIDBSC:             {},
		vaa.ChainIDPolygon:         {},
		vaa.ChainIDAvalanche:       {},
		vaa.ChainIDOasis:           {},
		vaa.ChainIDAurora:          {},
		vaa.ChainIDFantom:          {},
		vaa.ChainIDKarura:          {},
		vaa.ChainIDAcala:           {},
		vaa.ChainIDKlaytn:          {},
		vaa.ChainIDCelo:            {},
		vaa.ChainIDMoonbeam:        {},
		vaa.ChainIDArbitrum:        {},
		vaa.ChainIDOptimism:        {},
		vaa.ChainIDBase:            {},
		vaa.ChainIDScroll:          {},
		vaa.ChainIDMantle:          {},
		vaa.ChainIDSepolia:         {},
		vaa.ChainIDHolesky:         {},
		vaa.ChainIDArbitrumSepolia: {},
		vaa.ChainIDBaseSepolia:     {},
		vaa.ChainIDOptimismSepolia: {},
	}

	// But we don't want to allow CCQ if the chain is not enabled.
	for chainID := range supportedChains {
		if _, exists := chainQueryReqC[chainID]; !exists {
			delete(supportedChains, chainID)
		} else {
			logger.Info("queries supported on chain", zap.Stringer("chainID", chainID))

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
			// requestor validation happens here
			// request type validation is currently handled by the watcher
			// in the future, it may be worthwhile to catch certain types of
			// invalid requests here for tracking purposes
			// e.g.
			// - length check on "signature" 65 bytes
			// - length check on "to" address 20 bytes
			// - valid "block" strings

			allQueryRequestsReceived.Inc()
			digest := QueryRequestDigest(env, signedRequest.QueryRequest)

			// It's possible that the signature alone is not unique, and the digest alone is not unique, but the combination should be.
			requestID := hex.EncodeToString(signedRequest.Signature) + ":" + digest.String()

			qLogger.Info("received a query request", zap.String("requestID", requestID))

			signerBytes, err := ethCrypto.Ecrecover(digest.Bytes(), signedRequest.Signature)
			if err != nil {
				qLogger.Error("failed to recover public key", zap.String("requestID", requestID))
				invalidQueryRequestReceived.WithLabelValues("failed_to_recover_public_key").Inc()
				continue
			}

			signerAddress := ethCommon.BytesToAddress(ethCrypto.Keccak256(signerBytes[1:])[12:])

			if _, exists := allowedRequestors[signerAddress]; !exists {
				qLogger.Debug("invalid requestor", zap.String("requestor", signerAddress.Hex()), zap.String("requestID", requestID))
				invalidQueryRequestReceived.WithLabelValues("invalid_requestor").Inc()
				continue
			}

			// Make sure this is not a duplicate request. TODO: Should we do something smarter here than just dropping the duplicate?
			if oldReq, exists := pendingQueries[requestID]; exists {
				qLogger.Warn("dropping duplicate query request", zap.String("requestID", requestID), zap.Stringer("origRecvTime", oldReq.receiveTime))
				invalidQueryRequestReceived.WithLabelValues("duplicate_request").Inc()
				continue
			}

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
			errorFound := false
			queries := []*perChainQuery{}
			responses := make([]*PerChainQueryResponseInternal, len(queryRequest.PerChainQueries))
			receiveTime := time.Now()

			for requestIdx, pcq := range queryRequest.PerChainQueries {
				chainID := vaa.ChainID(pcq.ChainId)
				if _, exists := supportedChains[chainID]; !exists {
					qLogger.Debug("chain does not support cross chain queries", zap.String("requestID", requestID), zap.Stringer("chainID", chainID))
					invalidQueryRequestReceived.WithLabelValues("chain_does_not_support_ccq").Inc()
					errorFound = true
					break
				}

				channel, channelExists := chainQueryReqC[chainID]
				if !channelExists {
					qLogger.Debug("unknown chain ID for query request, dropping it", zap.String("requestID", requestID), zap.Stringer("chain_id", chainID))
					invalidQueryRequestReceived.WithLabelValues("failed_to_look_up_channel").Inc()
					errorFound = true
					break
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

			if errorFound {
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
				for _, resp := range pq.responses {
					if resp == nil {
						qLogger.Error("unexpected null response in pending query!", zap.String("requestID", resp.RequestID), zap.Int("requestIdx", resp.RequestIdx))
						continue
					}

					responses = append(responses, &PerChainQueryResponse{
						ChainId:  resp.ChainId,
						Response: resp.Response,
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
								pcq.ccqForwardToWatcher(qLogger, pq.receiveTime)
							}
						}
					}
				}
			}
		}
	}
}

// parseAllowedRequesters parses a comma separated list of allowed requesters into a map to be used for look ups.
func parseAllowedRequesters(ccqAllowedRequesters string) (map[ethCommon.Address]struct{}, error) {
	if ccqAllowedRequesters == "" {
		return nil, fmt.Errorf("if cross chain query is enabled `--ccqAllowedRequesters` must be specified")
	}

	var nullAddr ethCommon.Address
	result := make(map[ethCommon.Address]struct{})
	for _, str := range strings.Split(ccqAllowedRequesters, ",") {
		addr := ethCommon.BytesToAddress(ethCommon.Hex2Bytes(strings.TrimPrefix(str, "0x")))
		if addr == nullAddr {
			return nil, fmt.Errorf("invalid value in `--ccqAllowedRequesters`: `%s`", str)
		}
		result[addr] = struct{}{}
	}

	if len(result) <= 0 {
		return nil, fmt.Errorf("no allowed requestors specified, ccqAllowedRequesters: `%s`", ccqAllowedRequesters)
	}

	return result, nil
}

// ccqForwardToWatcher submits a query request to the appropriate watcher. It updates the request object if the write succeeds.
// If the write fails, it does not update the last update time, which will cause a retry next interval (until it times out)
func (pcq *perChainQuery) ccqForwardToWatcher(qLogger *zap.Logger, receiveTime time.Time) {
	select {
	// TODO: only send the query request itself and reassemble in this module
	case pcq.channel <- pcq.req:
		qLogger.Debug("forwarded query request to watcher", zap.String("requestID", pcq.req.RequestID), zap.Stringer("chainID", pcq.req.Request.ChainId))
		totalRequestsByChain.WithLabelValues(pcq.req.Request.ChainId.String()).Inc()
		pcq.lastUpdateTime = receiveTime
	default:
		// By leaving lastUpdateTime unset, we will retry next interval.
		qLogger.Warn("failed to send query request to watcher, will retry next interval", zap.String("requestID", pcq.req.RequestID), zap.Stringer("chain_id", pcq.req.Request.ChainId))
	}
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

func SupportsTimestampCaching(chainID vaa.ChainID) bool {
	/*
		- P1: Ethereum, Base, Optimism
		- P1.5: Arbitrum, Polygon, Avalanche
		- P2: BNB Chain, Moonbeam
		- P3: Acala, Celo, Fantom, Karura, Klaytn, Oasis
	*/

	return true
}
