package processor3

import (
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcastTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_broadcast_total3",
			Help: "Total number of signed observations queued for broadcast",
		})
)

func (p *ConcurrentProcessor) broadcastSignature(
	hash []byte,
	signature []byte,
	txhash []byte,
	msgId string,
) {
	obsv := gossipv1.SignedObservation{
		Addr:      p.ourAddr[:],
		Hash:      hash,
		Signature: signature,
		TxHash:    txhash,
		MessageId: msgId,
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservation{SignedObservation: &obsv}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	// send on p2p
	p.gossipSendC <- msg

	// Fast path for our own signature
	p.obsvC <- &obsv

	observationsBroadcastTotal.Inc()
}

func (p *ConcurrentProcessor) broadcastSignedVAA(v *vaa.VAA) {
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

	p.gossipSendC <- msg
}
