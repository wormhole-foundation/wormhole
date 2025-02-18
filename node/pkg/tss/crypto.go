package tss

import (
	"bytes"
	"encoding/binary"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/crypto/sha3"
)

type digest [digestSize]byte

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

	var b *bytes.Buffer

	// Since the msg is a protobug, we need to switch on the type of
	// the content (instead of adding an interface to the protogen file).
	switch m := msg.Content.(type) {
	case *tsscommv1.SignedMessage_TssContent:
		b = bytes.NewBuffer(nil)

		b.Write(m.TssContent.Payload)
		vaa.MustWrite(b, binary.BigEndian, m.TssContent.MsgSerialNumber)

		b.Write([]byte(msg.Sender.Id))
		b.Write(msg.Sender.Key)
	case *tsscommv1.SignedMessage_Problem:
		bts := (&parsedProblem{
			Problem: m.Problem,
			issuer:  msg.Sender,
		}).serialize()

		b = bytes.NewBuffer(bts)
	case *tsscommv1.SignedMessage_Announcement:
		bts := (&parsedAnnouncement{
			SawDigest: m.Announcement,
			issuer:    msg.Sender,
		}).serialize()

		b = bytes.NewBuffer(bts)
	case *tsscommv1.SignedMessage_HashEcho:
		d := digest{}
		copy(d[:], m.HashEcho.OriginalContetDigest)

		return d
	}

	return hash(b.Bytes())
}
