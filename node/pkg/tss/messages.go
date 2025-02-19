package tss

import (
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"google.golang.org/protobuf/proto"
)

// Echo is a broadcast message.
// can contain a `tsscommv1.Echo` and a list of recipients.
// and implements the `Sendable` interface.
type Echo struct {
	Echo       *tsscommv1.Echo
	Recipients []*tsscommv1.PartyId
}

type Unicast struct {
	Unicast     *tsscommv1.Unicast
	Receipients []*tsscommv1.PartyId
}

type IncomingMessage struct {
	Source  *tsscommv1.PartyId
	Content *tsscommv1.PropagatedMessage
}

func (i *IncomingMessage) hasContent() bool {
	return i != nil && i.Content != nil
}

func (i *IncomingMessage) IsUnicast() bool {
	if !i.hasContent() {
		return false
	}

	_, ok := i.Content.Message.(*tsscommv1.PropagatedMessage_Unicast)

	return ok
}

func (i *IncomingMessage) toBroadcastMsg() *tsscommv1.Echo {
	if !i.hasContent() {
		return nil
	}

	if echo, ok := i.Content.Message.(*tsscommv1.PropagatedMessage_Echo); ok {
		return echo.Echo
	}

	return nil
}

func (i *IncomingMessage) toUnicast() *tsscommv1.Unicast {
	if !i.hasContent() {
		return nil
	}

	if unicast, ok := i.Content.Message.(*tsscommv1.PropagatedMessage_Unicast); ok {
		return unicast.Unicast
	}

	return nil
}

func (i *IncomingMessage) IsBroadcast() bool {
	if !i.hasContent() {
		return false
	}

	_, ok := i.Content.Message.(*tsscommv1.PropagatedMessage_Echo)

	return ok
}

func (i *IncomingMessage) GetNetworkMessage() *tsscommv1.PropagatedMessage {
	if i.hasContent() {
		return i.Content
	}

	return nil
}

func (i *IncomingMessage) GetSource() *tsscommv1.PartyId {
	if i == nil {
		return nil
	}

	return i.Source
}

func newEcho(msg *tsscommv1.SignedMessage, recipients []*tsscommv1.PartyId) *Echo {
	return &Echo{Echo: &tsscommv1.Echo{Message: msg}, Recipients: recipients}
}

// GetDestinations implements Sendable.
func (e *Echo) GetDestinations() []*tsscommv1.PartyId {
	return e.Recipients
}

// GetNetworkMessage implements Sendable.
func (e *Echo) GetNetworkMessage() *tsscommv1.PropagatedMessage {
	return &tsscommv1.PropagatedMessage{
		Message: &tsscommv1.PropagatedMessage_Echo{Echo: e.Echo},
	}
}

// IsBroadcast implements Sendable.
func (e *Echo) IsBroadcast() bool {
	return true
}
func (e *Echo) cloneSelf() Sendable {
	return &Echo{Echo: proto.Clone(e.Echo).(*tsscommv1.Echo)}
}

func (e *Unicast) IsBroadcast() bool {
	return false
}

// GetDestination implements Sendable.
func (u *Unicast) GetDestinations() []*tsscommv1.PartyId {
	return u.Receipients
}

// GetNetworkMessage implements Sendable.
func (u *Unicast) GetNetworkMessage() *tsscommv1.PropagatedMessage {
	return &tsscommv1.PropagatedMessage{
		Message: &tsscommv1.PropagatedMessage_Unicast{
			Unicast: u.Unicast,
		},
	}
}

func (u *Unicast) cloneSelf() Sendable {
	clns := make([]*tsscommv1.PartyId, 0, len(u.Receipients))
	for _, pid := range u.Receipients {
		clns = append(clns, proto.Clone(pid).(*tsscommv1.PartyId))
	}

	return &Unicast{
		Unicast:     proto.Clone(u.Unicast).(*tsscommv1.Unicast),
		Receipients: clns,
	}
}
