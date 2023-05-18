package guardiand

import (
	"context"

	"github.com/benbjohnson/clock"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// TODO: should this use a different standard of signing messages, like https://eips.ethereum.org/EIPS/eip-712
var queryRequestPrefix = []byte("query_request_00000000000000000000|")

func queryRequestDigest(b []byte) common.Hash {
	return ethcrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

var allowedRequestor = common.BytesToAddress(common.Hex2Bytes("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"))

// Multiplex observation requests to the appropriate chain
func handleQueryRequests(
	ctx context.Context,
	clock clock.Clock,
	logger *zap.Logger,
	signedQueryReqC <-chan *gossipv1.SignedQueryRequest,
	chainQueryReqC map[vaa.ChainID]chan *gossipv1.QueryRequest,
) {
	qLogger := logger.With(zap.String("component", "queryHandler"))
	for {
		select {
		case <-ctx.Done():
			return
		case signedQueryRequest := <-signedQueryReqC:
			// requestor validation happens here
			// request type validation is currently handled by the watcher
			// in the future, it may be worthwhile to catch certain types of 
			// invalid requests here for tracking purposes
			requestorAddr := common.BytesToAddress(signedQueryRequest.RequestorAddr)
			if requestorAddr != allowedRequestor {
				qLogger.Error("invalid requestor", zap.String("requestor", requestorAddr.Hex()))
				continue
			}

			digest := queryRequestDigest(signedQueryRequest.QueryRequest)

			signerBytes, err := ethcrypto.Ecrecover(digest.Bytes(), signedQueryRequest.Signature)
			if err != nil {
				qLogger.Error("failed to recover public key", zap.String("requestor", requestorAddr.Hex()))
				continue
			}

			signerAddress := common.BytesToAddress(ethcrypto.Keccak256(signerBytes[1:])[12:])
			if signerAddress != requestorAddr {
				qLogger.Error("requestor signer mismatch", zap.String("requestor", requestorAddr.Hex()), zap.String("signer", signerAddress.Hex()))
				continue
			}

			var queryRequest gossipv1.QueryRequest
			err = proto.Unmarshal(signedQueryRequest.QueryRequest, &queryRequest)
			if err != nil {
				qLogger.Error("received invalid message",
					zap.String("requestor", requestorAddr.Hex()))
				continue
			}

			if channel, ok := chainQueryReqC[vaa.ChainID(queryRequest.ChainId)]; ok {
				select {
				// TODO: is pointer fine here?
				case channel <- &queryRequest:
				default:
					qLogger.Warn("failed to send query request to watcher",
						zap.Uint16("chain_id", uint16(queryRequest.ChainId)))
				}
			} else {
				qLogger.Error("unknown chain ID for query request",
					zap.Uint16("chain_id", uint16(queryRequest.ChainId)))
			}
		}
	}
}
