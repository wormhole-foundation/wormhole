package common

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

const SignedQueryRequestChannelSize = 50

// QueryRequest is an internal representation of a query request.
type QueryRequest struct {
	SignedRequest *gossipv1.SignedQueryRequest
	Request       *gossipv1.QueryRequest
	RequestID     string
	ChainID       vaa.ChainID
}

// CreateQueryRequest creates a QueryRequest object from the signed query request.
func CreateQueryRequest(signedRequest *gossipv1.SignedQueryRequest, request *gossipv1.QueryRequest) *QueryRequest {
	return &QueryRequest{
		SignedRequest: signedRequest,
		Request:       request,
		RequestID:     hex.EncodeToString(signedRequest.Signature),
		ChainID:       vaa.ChainID(request.ChainId),
	}
}

// QueryRequestDigest returns the query signing prefix based on the environment.
func QueryRequestDigest(env Environment, b []byte) ethCommon.Hash {
	// TODO: should this use a different standard of signing messages, like https://eips.ethereum.org/EIPS/eip-712
	var queryRequestPrefix []byte
	if env == MainNet {
		queryRequestPrefix = []byte("mainnet_query_request_000000000000|")
	} else if env == TestNet {
		queryRequestPrefix = []byte("testnet_query_request_000000000000|")
	} else {
		queryRequestPrefix = []byte("devnet_query_request_0000000000000|")
	}

	return ethCrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

// PostSignedQueryRequest posts a signed query request to the specified channel.
func PostSignedQueryRequest(signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest, req *gossipv1.SignedQueryRequest) error {
	select {
	case signedQueryReqSendC <- req:
		return nil
	default:
		return ErrChanFull
	}
}

// ValidateQueryRequest does basic validation on a received query request.
func ValidateQueryRequest(queryRequest *gossipv1.QueryRequest) error {
	if queryRequest.ChainId > math.MaxUint16 {
		return fmt.Errorf("invalid chain id: %d is out of bounds", queryRequest.ChainId)
	}
	switch req := queryRequest.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		if len(req.EthCallQueryRequest.To) != 20 {
			return fmt.Errorf("invalid length for To contract")
		}
		if len(req.EthCallQueryRequest.Data) > math.MaxUint32 {
			return fmt.Errorf("request data too long")
		}
		if len(req.EthCallQueryRequest.Block) > math.MaxUint32 {
			return fmt.Errorf("request block too long")
		}
		if !strings.HasPrefix(req.EthCallQueryRequest.Block, "0x") {
			return fmt.Errorf("request block must be a hex number or hash starting with 0x")
		}
	default:
		return fmt.Errorf("received invalid message from query module")
	}

	return nil
}

func SignedQueryRequestEqual(left *gossipv1.SignedQueryRequest, right *gossipv1.SignedQueryRequest) bool {
	if !bytes.Equal(left.QueryRequest, right.QueryRequest) {
		return false
	}
	if !bytes.Equal(left.Signature, right.Signature) {
		return false
	}
	return true
}

func QueryRequestEqual(left *gossipv1.QueryRequest, right *gossipv1.QueryRequest) bool {
	if left.ChainId != right.ChainId {
		return false
	}
	if left.Nonce != right.Nonce {
		return false
	}

	switch reqLeft := left.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		switch reqRight := right.Message.(type) {
		case *gossipv1.QueryRequest_EthCallQueryRequest:
			if reqLeft.EthCallQueryRequest.Block != reqRight.EthCallQueryRequest.Block {
				return false
			}
			if !bytes.Equal(reqLeft.EthCallQueryRequest.To, reqRight.EthCallQueryRequest.To) {
				return false
			}
			if !bytes.Equal(reqLeft.EthCallQueryRequest.Data, reqRight.EthCallQueryRequest.Data) {
				return false
			}
		default:
			return false
		}
	default:
		return false
	}

	return true
}
