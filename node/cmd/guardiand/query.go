package guardiand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	// requestTimeout indicates how long before a request is considered to have timed out.
	requestTimeout = 1 * time.Minute

	// retryInterval specifies how long we will wait between retry intervals. This is the interval of our ticker.
	retryInterval = 10 * time.Second
)

type (
	// pendingQuery is the cache entry for a given query.
	pendingQuery struct {
		req            *common.QueryRequest
		channel        chan *common.QueryRequest
		receiveTime    time.Time
		lastUpdateTime time.Time
		inProgress     bool

		// respPub is only populated when we need to retry sending the response to p2p.
		respPub *common.QueryResponsePublication
	}
)

// handleQueryRequests multiplexes observation requests to the appropriate chain
func handleQueryRequests(
	ctx context.Context,
	logger *zap.Logger,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *common.QueryRequest,
	allowedRequestors map[ethCommon.Address]struct{},
	queryResponseReadC <-chan *common.QueryResponse,
	queryResponseWriteC chan<- *common.QueryResponsePublication,
	env common.Environment,
) {
	qLogger := logger.With(zap.String("component", "ccqhandler"))
	qLogger.Info("cross chain queries are enabled", zap.Any("allowedRequestors", allowedRequestors), zap.String("env", string(env)))

	pendingQueries := make(map[string]*pendingQuery) // Key is requestID.

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case signedQueryRequest := <-signedQueryReqC:
			// requestor validation happens here
			// request type validation is currently handled by the watcher
			// in the future, it may be worthwhile to catch certain types of
			// invalid requests here for tracking purposes
			// e.g.
			// - length check on "signature" 65 bytes
			// - length check on "to" address 20 bytes
			// - valid "block" strings

			digest := common.QueryRequestDigest(env, signedQueryRequest.QueryRequest)

			signerBytes, err := ethCrypto.Ecrecover(digest.Bytes(), signedQueryRequest.Signature)
			if err != nil {
				qLogger.Error("failed to recover public key")
				continue
			}

			signerAddress := ethCommon.BytesToAddress(ethCrypto.Keccak256(signerBytes[1:])[12:])

			if _, exists := allowedRequestors[signerAddress]; !exists {
				qLogger.Error("invalid requestor", zap.String("requestor", signerAddress.Hex()))
				continue
			}

			var qr gossipv1.QueryRequest
			err = proto.Unmarshal(signedQueryRequest.QueryRequest, &qr)
			if err != nil {
				qLogger.Error("failed to unmarshal query request", zap.String("requestor", signerAddress.Hex()), zap.Error(err))
				continue
			}

			if err := common.ValidateQueryRequest(&qr); err != nil {
				qLogger.Error("received invalid message", zap.String("requestor", signerAddress.Hex()), zap.Error(err))
				continue
			}

			queryRequest := common.CreateQueryRequest(signedQueryRequest, &qr)

			// Look up the channel for this chain.
			channel, channelExists := chainQueryReqC[queryRequest.ChainID]
			if !channelExists {
				qLogger.Error("unknown chain ID for query request, dropping it", zap.String("requestID", queryRequest.RequestID), zap.Stringer("chain_id", queryRequest.ChainID))
				continue
			}

			// Make sure this is not a duplicate request. TODO: Should we do something smarter here than just dropping the duplicate?
			if oldReq, exists := pendingQueries[queryRequest.RequestID]; exists {
				qLogger.Warn("dropping duplicate query request", zap.String("requestID", queryRequest.RequestID), zap.Stringer("origRecvTime", oldReq.receiveTime))
				continue
			}

			// Add the query to our cache.
			pq := &pendingQuery{
				req:         queryRequest,
				channel:     channel,
				receiveTime: time.Now(),
				inProgress:  true,
			}
			pendingQueries[queryRequest.RequestID] = pq

			// Forward the request to the watcher.
			ccqForwardToWatcher(qLogger, pq)

		case resp := <-queryResponseReadC:
			if resp.Status == common.QuerySuccess {
				if len(resp.Results) == 0 {
					qLogger.Error("received a successful query response with no results, dropping it!", zap.String("requestID", resp.RequestID))
					continue
				}

				respPub := &common.QueryResponsePublication{
					Request:   resp.SignedRequest,
					Responses: resp.Results,
				}

				// Send the response to be published.
				select {
				case queryResponseWriteC <- respPub:
					qLogger.Debug("forwarded query response to p2p", zap.String("requestID", resp.RequestID))
					delete(pendingQueries, resp.RequestID)
				default:
					if pq, exists := pendingQueries[resp.RequestID]; exists {
						qLogger.Warn("failed to publish query response to p2p, will retry publishing next interval", zap.String("requestID", resp.RequestID))
						pq.respPub = respPub
						pq.inProgress = false
					} else {
						qLogger.Warn("failed to publish query response to p2p, request is no longer in cache, dropping it", zap.String("requestID", resp.RequestID))
						delete(pendingQueries, resp.RequestID)
					}
				}
			} else if resp.Status == common.QueryRetryNeeded {
				if pq, exists := pendingQueries[resp.RequestID]; exists {
					qLogger.Warn("query failed, will retry next interval", zap.String("requestID", resp.RequestID))
					pq.inProgress = false
				} else {
					qLogger.Warn("query failed, request is no longer in cache, dropping it", zap.String("requestID", resp.RequestID))
				}
			} else if resp.Status == common.QueryFatalError {
				qLogger.Error("query encountered a fatal error, dropping it", zap.String("requestID", resp.RequestID))
				delete(pendingQueries, resp.RequestID)
			} else {
				qLogger.Error("received an unexpected query status, dropping it", zap.String("requestID", resp.RequestID), zap.Int("status", int(resp.Status)))
				delete(pendingQueries, resp.RequestID)
			}

		case <-ticker.C:
			now := time.Now()
			for reqId, pq := range pendingQueries {
				timeout := pq.receiveTime.Add(requestTimeout)
				qLogger.Debug("audit", zap.String("requestId", reqId), zap.Stringer("receiveTime", pq.receiveTime), zap.Stringer("retryTime", pq.lastUpdateTime.Add(retryInterval)), zap.Stringer("timeout", timeout))
				if timeout.Before(now) {
					qLogger.Warn("query request timed out, dropping it", zap.String("requestId", reqId), zap.Stringer("receiveTime", pq.receiveTime))
					delete(pendingQueries, reqId)
				} else {
					if pq.respPub != nil {
						// Resend the response to be published.
						select {
						case queryResponseWriteC <- pq.respPub:
							qLogger.Debug("resend of query response to p2p succeeded", zap.String("requestID", reqId))
							delete(pendingQueries, reqId)
						default:
							qLogger.Warn("resend of query response to p2p failed again, will keep retrying", zap.String("requestID", reqId))
						}
					} else if !pq.inProgress && pq.lastUpdateTime.Add(retryInterval).Before(now) {
						qLogger.Info("retrying query request", zap.String("requestId", reqId), zap.Stringer("receiveTime", pq.receiveTime))
						pq.inProgress = true
						ccqForwardToWatcher(qLogger, pq)
					}
				}
			}
		}
	}
}

// ccqParseAllowedRequesters parses a comma separated list of allowed requesters into a map to be used for look ups.
func ccqParseAllowedRequesters(ccqAllowedRequesters string) (map[ethCommon.Address]struct{}, error) {
	if ccqAllowedRequesters == "" {
		return nil, fmt.Errorf("if cross chain query is enabled `--ccqAllowedRequesters` must be specified")
	}

	var nullAddr ethCommon.Address
	result := make(map[ethCommon.Address]struct{})
	for _, str := range strings.Split(ccqAllowedRequesters, ",") {
		addr := ethCommon.BytesToAddress(ethCommon.Hex2Bytes(str))
		if addr == nullAddr {
			return nil, fmt.Errorf("invalid value in `--ccqAllowedRequesters`: `%s`", str)
		}
		result[addr] = struct{}{}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no allowed requestors specified, ccqAllowedRequesters: `%s`", ccqAllowedRequesters)
	}

	return result, nil
}

// ccqForwardToWatcher submits a query request to the appropriate watcher. It updates the request object if the write succeeds.
// If the write fails, it does not update the last update time, which will cause a retry next interval (until it times out)
func ccqForwardToWatcher(qLogger *zap.Logger, pq *pendingQuery) {
	select {
	// TODO: only send the query request itself and reassemble in this module
	case pq.channel <- pq.req:
		qLogger.Debug("forwarded query request to watcher", zap.String("requestID", pq.req.RequestID), zap.Stringer("chainID", pq.req.ChainID))
		pq.lastUpdateTime = pq.receiveTime
	default:
		// By leaving lastUpdateTime unset and setting inProgress to false, we will retry next interval.
		qLogger.Warn("failed to send query request to watcher, will retry next interval", zap.String("requestID", pq.req.RequestID), zap.Stringer("chain_id", pq.req.ChainID))
		pq.inProgress = false
	}
}
