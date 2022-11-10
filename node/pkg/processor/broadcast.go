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
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	observationsBroadcastTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_observations_broadcast_total",
			Help: "Total number of signed observations queued for broadcast",
		})
	batchObservationsBroadcastTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_batch_observations_broadcast_total",
			Help: "Total number of signed batch observations queued for broadcast",
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
	p.state.signatures[hash].txHash = txhash
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

func (p *Processor) broadcastBatchSignature(
	b Batch,
	signature []byte,
	txhash []byte,
) {
	digest := b.SigningMsg()
	obsv := gossipv1.SignedBatchObservation{
		Addr:      crypto.PubkeyToAddress(p.gk.PublicKey).Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
		TxId:      txhash,
		ChainId:   uint32(b.GetEmitterChain()),
		BatchId:   b.BatchID(),
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedBatchObservation{SignedBatchObservation: &obsv}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	p.sendC <- msg

	// Store our batch VAA
	hash := hex.EncodeToString(digest.Bytes())

	if p.state.batchSignatures[hash] == nil {
		p.state.batchSignatures[hash] = &batchState{
			state: state{
				firstObserved: time.Now(),
				signatures:    map[ethcommon.Address][]byte{},
				source:        "loopback",
			},
		}
	}

	p.state.batchSignatures[hash].ourObservation = b
	p.state.batchSignatures[hash].ourMsg = msg
	p.state.batchSignatures[hash].source = b.GetEmitterChain().String()
	p.state.batchSignatures[hash].gs = p.gs // guaranteed to match ourObservation - there's no concurrent access to p.gs

	// Fast path for our own signature
	go func() { p.batchObsvC <- &obsv }()

	batchObservationsBroadcastTotal.Inc()
}

func (p *Processor) broadcastSignedBatchVAA(v *vaa.BatchVAA) {
	b, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedBatchVaaWithQuorum{
		SignedBatchVaaWithQuorum: &gossipv1.SignedBatchVAAWithQuorum{BatchVaa: b},
	}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	p.sendC <- msg
}
