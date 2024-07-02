package processor

import (
	"encoding/hex"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_queued_for_broadcast",
			Help: "Total number of signed observations queued for broadcast",
		})

	observationsPostedInternally = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_posted_internally",
			Help: "Total number of our observations posted internally",
		})

	signedVAAsBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_signed_vaas_queued_for_broadcast",
			Help: "Total number of signed vaas queued for broadcast",
		})
)

func (p *Processor) broadcastSignature(
	o Observation,
	signature []byte,
	txhash []byte,
) {
	digest := o.SigningDigest()
	obsv := gossipv1.SignedObservation{
		Addr:      p.ourAddr.Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    txhash,
		MessageId: o.MessageID(),
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservation{SignedObservation: &obsv}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	// Broadcast the observation.
	p.gossipSendC <- msg
	observationsBroadcast.Inc()

	hash := hex.EncodeToString(digest.Bytes())

	if p.state.signatures[hash] == nil {
		p.state.signatures[hash] = &state{
			firstObserved: time.Now(),
			nextRetry:     time.Now().Add(nextRetryDuration(0)),
			signatures:    map[ethcommon.Address][]byte{},
			source:        "loopback",
		}
	}

	p.state.signatures[hash].ourObservation = o
	p.state.signatures[hash].ourMsg = msg
	p.state.signatures[hash].txHash = txhash
	p.state.signatures[hash].source = o.GetEmitterChain().String()
	p.state.signatures[hash].gs = p.gs // guaranteed to match ourObservation - there's no concurrent access to p.gs

	// Fast path for our own signature
	// send to obsvC directly if there is capacity, otherwise do it in a go routine.
	// We can't block here because the same process would be responsible for reading from obsvC.
	om := node_common.CreateMsgWithTimestamp[gossipv1.SignedObservation](&obsv)
	select {
	case p.obsvC <- om:
	default:
		go func() { p.obsvC <- om }()
	}

	observationsPostedInternally.Inc()
}

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
	p.gossipSendC <- msg
	signedVAAsBroadcast.Inc()

	if p.gatewayRelayer != nil {
		p.gatewayRelayer.SubmitVAA(v)
	}
}
