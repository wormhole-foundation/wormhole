package common

import (
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

const SignedQueryRequestChannelSize = 50

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

func PostSignedQueryRequest(signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest, req *gossipv1.SignedQueryRequest) error {
	select {
	case signedQueryReqSendC <- req:
		return nil
	default:
		return ErrChanFull
	}
}
