package tss

import (
	"context"
	"crypto/ecdsa"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/yossigi/tss-lib/v2/common"
)

// ReliableMessageHandler is the interface to give any component with the ability to receive over the network incoming TSS messages.
type ReliableMessageHandler interface {
	// HandleIncomingTssMessage receives a network message and process it using a reliable-broadcast protocol.
	HandleIncomingTssMessage(msg *gossipv1.GossipMessage_TssMessage)
	ProducedOutputMessages() <-chan *gossipv1.GossipMessage // just need to propagate this through the p2p network
}

// Signer is the interface to give any component with the ability to authorise a new threshold signature over a message.
type Signer interface {
	BeginAsyncThresholdSigningProtocol(vaaDigest []byte) error
	ProducedSignature() <-chan *common.SignatureData

	GetPublicKey() *ecdsa.PublicKey
	GetEthAddress() ethcommon.Address
}

// ReliableTSS represents a TSS engine that can fully support logic of reliable broadcast needed for the security of TSS over the network.
type ReliableTSS interface {
	ReliableMessageHandler
	Signer
	Start(ctx context.Context) error
}
