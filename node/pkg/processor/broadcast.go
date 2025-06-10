package processor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_queued_for_broadcast",
			Help: "Total number of signed observations queued for broadcast",
		})

	batchObservationsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_batch_observations_queued_for_broadcast",
			Help: "Total number of signed batched observations queued for broadcast",
		})

	signedVAAsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_signed_vaas_queued_for_broadcast",
			Help: "Total number of signed vaas queued for broadcast",
		})
)

// broadcastSignature broadcasts the observation for something we observed locally.
func (p *Processor) broadcastSignature(
	messageID string,
	k *common.MessagePublication,
	digest ethCommon.Hash,
	signature []byte,
	shouldPublishImmediately bool,
) (ourObs *gossipv1.Observation, msg []byte) {
	// Create the observation to either be submitted to the batch processor or published immediately.
	ourObs = &gossipv1.Observation{
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    k.TxID,
		MessageId: messageID,
	}

	if shouldPublishImmediately {
		msg = p.publishImmediately(ourObs)
		observationsBroadcast.Inc()
	} else {
		p.postObservationToBatch(ourObs)
		batchObservationsBroadcast.Inc()
	}

	if p.alternatePublisher != nil {
		p.alternatePublisher.PublishObservation(k.EmitterChain, ourObs)
	}

	return ourObs, msg
}

// broadcastSignedVAA broadcasts a VAA to the gossip network.
func (p *Processor) broadcastSignedVAA(v *vaa.VAA) {
	b, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedVaaWithQuorum{
		SignedVaaWithQuorum: &gossipv1.SignedVAAWithQuorum{Vaa: b},
	}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	// Broadcast the signed VAA. The channel is buffered. If it overflows, just drop it and rely on a reobservation if necessary.
	common.WriteToChannelWithoutBlocking(p.gossipVaaSendC, msg, "vaa_broadcast")
	select {
	case p.gossipVaaSendC <- msg:
		signedVAAsBroadcast.Inc()
	default:
		vaaPublishChannelOverflow.Inc()
	}

	if p.gatewayRelayer != nil {
		p.gatewayRelayer.SubmitVAA(v)
	}
}
