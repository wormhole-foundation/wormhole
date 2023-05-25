package guardiand

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Multiplex observation requests to the appropriate chain
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

			if channel, ok := chainQueryReqC[vaa.ChainID(queryRequest.ChainId)]; ok {
				select {
				// TODO: only send the query request itself and reassemble in this module
				case channel <- signedQueryRequest:
					qLogger.Debug("forwarded query request to watcher", zap.Uint32("chainID", queryRequest.ChainId), zap.String("requestID", hex.EncodeToString(signedQueryRequest.Signature)))
				default:
					qLogger.Warn("failed to send query request to watcher",
						zap.Uint16("chain_id", uint16(queryRequest.ChainId)))
				}
			} else {
				qLogger.Error("unknown chain ID for query request",
					zap.Uint16("chain_id", uint16(queryRequest.ChainId)))
			}

		case resp := <-queryResponseReadC:
			if resp.Success {
				select {
				case queryResponseWriteC <- resp.Msg:
					qLogger.Debug("forwarded query response to p2p", zap.String("requestID", resp.RequestID()))
					// TODO: Remove from cache.
				default:
					qLogger.Warn("failed to send query response to p2p, dropping it", zap.String("requestID", resp.RequestID()))
				}
			} else {
				// TODO: Retry logic
			}
		}
	}
}

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
