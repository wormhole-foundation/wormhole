package common

import (
	"errors"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

var ErrChanFull = errors.New("channel is full")

func PostObservationRequest(obsvReqSendC chan<- *gossipv1.ObservationRequest, req *gossipv1.ObservationRequest) error {
	select {
	case obsvReqSendC <- req:
		return nil
	default:
		return ErrChanFull
	}
}
