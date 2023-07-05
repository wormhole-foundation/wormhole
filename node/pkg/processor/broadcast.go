package processor

import (
	"encoding/hex"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcastTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_broadcast_total",
			Help: "Total number of signed observations queued for broadcast",
		})
)

func (p *Processor) broadcastSignature(
	o Observation,
	signature []byte,
	txhash []byte,
) {
	digest := o.SigningDigest()
	obsv := gossipv1.SignedObservation{
		Addr:      crypto.PubkeyToAddress(p.gk.PublicKey).Bytes(),
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

	p.gossipSendC <- msg

	// Store our VAA in case we're going to submit it to Solana
	hash := hex.EncodeToString(digest.Bytes())

	obsState, created := p.state.getOrCreateState(hash)
	obsState.lock.Lock()
	defer obsState.lock.Unlock()

	if created {
		obsState.source = "loopback"
		obsState.nextRetry = time.Now().Add(nextRetryDuration(0))
	}

	obsState.ourObservation = o
	obsState.ourMsg = msg
	obsState.txHash = txhash
	obsState.source = o.GetEmitterChain().String()

	// Fast path for our own signature
	// send to obsvC directly if there is capacity, otherwise do it in a go routine.
	// We can't block here because the same process would be responsible for reading from obsvC.
	om := node_common.CreateMsgWithTimestamp[gossipv1.SignedObservation](&obsv)
	select {
	case p.obsvC <- om:
	default:
		go func() { p.obsvC <- om }()
	}

	observationsBroadcastTotal.Inc()
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

	p.gossipSendC <- msg

	if p.gatewayRelayer != nil {
		p.gatewayRelayer.SubmitVAA(v)
	}
}
