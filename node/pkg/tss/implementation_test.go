package tss

import "testing"

func TestIncomingMessageHandling(t *testing.T) {
	// todo: test a macced message is verified.
	// todo: test faulty message.
	// todo: test a signed message is verified.
}

func TestOutgoingMessageCreation(t *testing.T) {
	// todo: ensure first outgoing messages are 5 unicast messages
	// ensure each is MACe with the correct guardian symkey.
	// todo: ensure the next outgoing message is a broadcast message
	// ensure it is signed.
}
