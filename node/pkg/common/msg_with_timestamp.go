package common

import (
	"time"
)

// MsgWithTimeStamp allows us to track the time of receipt of an event.
type MsgWithTimeStamp[T any] struct {
	Msg       *T
	Timestamp time.Time
}

// CreateMsgWithTimestamp creates a new MsgWithTimeStamp with the current time.
func CreateMsgWithTimestamp[T any](msg *T) *MsgWithTimeStamp[T] {
	return &MsgWithTimeStamp[T]{
		Msg:       msg,
		Timestamp: time.Now(),
	}
}

// PostMsgWithTimestamp sends the message to the specified channel using the current timestamp. Returns ErrChanFull on error.
func PostMsgWithTimestamp[T any](msg *T, c chan<- *MsgWithTimeStamp[T]) error {
	select {
	case c <- CreateMsgWithTimestamp[T](msg):
		return nil
	default:
		return ErrChanFull
	}
}
