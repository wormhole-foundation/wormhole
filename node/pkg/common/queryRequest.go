package common

import (
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

const SignedQueryRequestChannelSize = 50

func PostSignedQueryRequest(signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest, req *gossipv1.SignedQueryRequest) error {
	select {
	case signedQueryReqSendC <- req:
		return nil
	default:
		return ErrChanFull
	}
}
