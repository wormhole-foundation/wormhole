package tss

import (
	"bytes"
	"encoding/binary"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/crypto/sha3"
)

type digest [32]byte // TODO: Consider using the common.Hash they use in other places.

func hash(msg []byte) digest {
	d := sha3.Sum256(msg)

	return d
}

// using this function since proto.Marshal is either non-deterministic,
// or it isn't canonical - as stated in proto.MarshalOptions docs.

func hashSignedMessage(msg *tsscommv1.SignedMessage) digest {
	if msg == nil {
		return digest{}
	}

	b := bytes.NewBuffer(nil)
	b.Write(msg.Content.Payload)
	vaa.MustWrite(b, binary.BigEndian, msg.Content.MsgSerialNumber)
	pid := msg.Sender
	b.Write([]byte(pid.Id))
	b.Write(pid.Key)

	return hash(b.Bytes())
}
