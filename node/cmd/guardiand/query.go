package guardiand

import (
	"context"
	"encoding/hex"
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
	pendingQuery struct {
		req            *gossipv1.SignedQueryRequest
		reqId          string
		chainId        vaa.ChainID
		channel        chan *gossipv1.SignedQueryRequest
		receiveTime    time.Time
		lastUpdateTime time.Time
		inProgress     bool
		resp           *common.QueryResponse
	}
)

// TODO: should this use a different standard of signing messages, like https://eips.ethereum.org/EIPS/eip-712
var queryRequestPrefix = []byte("mainnet_query_request_000000000000|")

func queryRequestDigest(b []byte) ethCommon.Hash {
	return ethCrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

// handleQueryRequests multiplexes observation requests to the appropriate chain
func handleQueryRequests(
	ctx context.Context,
	logger *zap.Logger,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *gossipv1.SignedQueryRequest,
	allowedRequestors map[ethCommon.Address]struct{},
	queryResponseReadC <-chan *common.QueryResponse,
	queryResponseWriteC chan<- *common.QueryResponsePublication,
	env common.Environment,
) {
	qLogger := logger.With(zap.String("component", "ccqhandler"))
	qLogger.Info("cross chain queries are enabled", zap.Any("allowedRequestors", allowedRequestors))

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

			var queryRequest gossipv1.QueryRequest
			err = proto.Unmarshal(signedQueryRequest.QueryRequest, &queryRequest)
			if err != nil {
				qLogger.Error("received invalid message",
					zap.String("requestor", signerAddress.Hex()))
				continue
			}

			reqId := requestID(signedQueryRequest)
			chainId := vaa.ChainID(queryRequest.ChainId)

			// Look up the channel for this chain.
			channel, channelExists := chainQueryReqC[chainId]
			if !channelExists {
				qLogger.Error("unknown chain ID for query request, dropping it", zap.String("requestID", reqId), zap.Uint32("chain_id", queryRequest.ChainId))
				continue
			}

			// Make sure this is not a duplicate request. TODO: Should we do something smarter here than just dropping the duplicate?
			if oldReq, exists := pendingQueries[reqId]; exists {
				qLogger.Warn("dropping duplicate query request", zap.String("requestID", reqId), zap.Stringer("origRecvTime", oldReq.receiveTime))
				continue
			}

			// Add the query to our cache.
			pq := &pendingQuery{
				req:         signedQueryRequest,
				reqId:       reqId,
				chainId:     chainId,
				channel:     channel,
				receiveTime: time.Now(),
			}
			pendingQueries[reqId] = pq

			// Forward the request to the watcher.
			ccqForwardToWatcher(qLogger, pq)

		case resp := <-queryResponseReadC:
			reqId := resp.RequestID()
			if resp.Success {
				// Send the response to be published.
				select {
				case queryResponseWriteC <- resp.Msg:
					qLogger.Debug("forwarded query response to p2p", zap.String("requestID", reqId))
					delete(pendingQueries, reqId)
				default:
					if pq, exists := pendingQueries[reqId]; exists {
						qLogger.Warn("failed to publish query response to p2p, will retry publishing next interval", zap.String("requestID", reqId))
						pq.inProgress = false
						pq.resp = resp
					} else {
						qLogger.Warn("failed to publish query response to p2p, request is no longer in cache, dropping it", zap.String("requestID", reqId))
						delete(pendingQueries, reqId)
					}
				}
			} else {
				if pq, exists := pendingQueries[reqId]; exists {
					qLogger.Warn("query failed, will retry next interval", zap.String("requestID", reqId))
					pq.inProgress = false
				}
			}

		case <-ticker.C:
			now := time.Now()
			for reqId, pq := range pendingQueries {
				if now.Before(pq.receiveTime.Add(requestTimeout)) {
					qLogger.Warn("query request timed out, dropping it", zap.String("requestId", reqId))
					delete(pendingQueries, reqId)
				} else if !pq.inProgress {
					if pq.resp != nil {
						// Resend the response to be published.
						select {
						case queryResponseWriteC <- pq.resp.Msg:
							qLogger.Debug("resend of query response to p2p succeeded", zap.String("requestID", reqId))
							delete(pendingQueries, reqId)
						default:
							qLogger.Warn("resend of query response to p2p failed again, will keep retrying", zap.String("requestID", reqId))
						}
					} else if now.Before(pq.lastUpdateTime.Add(retryInterval)) {
						qLogger.Info("retrying query request", zap.String("requestId", pq.reqId), zap.Stringer("receiveTime", pq.receiveTime))
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
		qLogger.Debug("forwarded query request to watcher", zap.String("requestID", pq.reqId), zap.Stringer("chainID", pq.chainId))
		pq.lastUpdateTime = pq.receiveTime
		pq.inProgress = true
	default:
		// By setting inProgress to false and leaving lastUpdateTime unset, we will retry next interval.
		qLogger.Warn("failed to send query request to watcher, will retry next interval", zap.String("requestID", pq.reqId), zap.Uint16("chain_id", uint16(pq.chainId)))
		pq.inProgress = false
	}
}

// requestID returns the request signature as a hex string.
func requestID(req *gossipv1.SignedQueryRequest) string {
	if req == nil {
		return "nil"
	}
	return hex.EncodeToString(req.Signature)
}
