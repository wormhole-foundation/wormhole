package processor

import (
	"encoding/hex"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
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
	digest := o.SigningMsg()
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

	p.sendC <- msg

	// Store our VAA in case we're going to submit it to Solana
	hash := hex.EncodeToString(digest.Bytes())

	if p.state.signatures[hash] == nil {
		p.state.signatures[hash] = &state{
			firstObserved: time.Now(),
			signatures:    map[ethcommon.Address][]byte{},
			source:        "loopback",
		}
	}

	p.state.signatures[hash].ourObservation = o
	p.state.signatures[hash].ourMsg = msg
	p.state.signatures[hash].source = o.GetEmitterChain().String()
	p.state.signatures[hash].gs = p.gs // guaranteed to match ourObservation - there's no concurrent access to p.gs

	// Fast path for our own signature
	go func() { p.obsvC <- &obsv }()

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

	p.sendC <- msg
}
