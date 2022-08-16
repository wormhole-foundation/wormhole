package common

import (
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

const ObsvReqChannelSize = 50
const ObsvReqChannelFullError = "channel is full"

func PostObservationRequest(obsvReqSendC chan *gossipv1.ObservationRequest, req *gossipv1.ObservationRequest) error {
	select {
	case obsvReqSendC <- req:
		return nil
	default:
		return fmt.Errorf(ObsvReqChannelFullError)
	}
}
