package processor

import (
	"encoding/hex"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

func (p *Processor) broadcastSignature(v *vaa.VAA, signature []byte) {
	digest, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}

	obsv := gossipv1.SignedObservation{
		Addr:      crypto.PubkeyToAddress(p.gk.PublicKey).Bytes(),
		Hash:      digest.Bytes(),
		Signature: signature,
	}

	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservation{SignedObservation: &obsv}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		panic(err)
	}

	p.sendC <- msg

	// Store our VAA in case we're going to submit it to Solana
	hash := hex.EncodeToString(digest.Bytes())

	if p.state.vaaSignatures[hash] == nil {
		p.state.vaaSignatures[hash] = &vaaState{
			firstObserved: time.Now(),
			signatures:    map[ethcommon.Address][]byte{},
		}
	}

	p.state.vaaSignatures[hash].ourVAA = v
	p.state.vaaSignatures[hash].ourMsg = msg

	// Fast path for our own signature
	go func() { p.obsvC <- &obsv }()
}
