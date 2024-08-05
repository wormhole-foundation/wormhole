package processor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_queued_for_broadcast",
			Help: "Total number of signed observations queued for broadcast",
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
	txhash []byte,
	digest ethCommon.Hash,
	signature []byte,
) (*gossipv1.SignedObservation, []byte) {
	obsv := gossipv1.SignedObservation{
		Addr:      p.ourAddr.Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    txhash,
		MessageId: messageID,
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservation{SignedObservation: &obsv}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	// Broadcast the observation.
	p.gossipAttestationSendC <- msg
	observationsBroadcast.Inc()
	return &obsv, msg
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

	// Broadcast the signed VAA.
	p.gossipVaaSendC <- msg
	signedVAAsBroadcast.Inc()

	if p.gatewayRelayer != nil {
		p.gatewayRelayer.SubmitVAA(v)
	}
}
