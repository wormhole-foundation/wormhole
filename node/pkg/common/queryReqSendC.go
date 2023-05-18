package common

import (
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

const QueryReqChannelSize = 50

func PostQueryRequest(obsvReqSendC chan<- *gossipv1.QueryRequest, req *gossipv1.QueryRequest) error {
	select {
	case obsvReqSendC <- req:
		return nil
	default:
		return ErrChanFull
	}
}
