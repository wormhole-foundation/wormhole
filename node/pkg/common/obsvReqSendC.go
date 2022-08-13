package common

import (
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

const ObsvReqChannelSize = 50
const ObsvReqChannelFullError = "channel is full"

func PostObservationRequest(obsvReqSendC chan *gossipv1.ObservationRequest, req *gossipv1.ObservationRequest) error {
	if len(obsvReqSendC) >= cap(obsvReqSendC) {
		return fmt.Errorf(ObsvReqChannelFullError)
	}

	obsvReqSendC <- req
	return nil
}
