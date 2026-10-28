package common_test

import (
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const fuzzQueueMsgCount = 8

// FuzzPendingMessageQueue runs a sequence of queue operations and compares the queue with a map
// from message ID to release time.
//
// Each operation is two bytes: an opcode and an argument. The argument selects one of
// fuzzQueueMsgCount messages and a time offset in seconds.
func FuzzPendingMessageQueue(f *testing.F) {
	f.Add([]byte{0, 0, 0, 9, 0, 18, 1, 255, 1, 255, 1, 255})
	f.Add([]byte{0, 3, 0, 3, 2, 3, 3, 0, 1, 0})
	f.Add([]byte{0, 40, 0, 41, 1, 40, 1, 41, 3, 0})

	base := time.Unix(1_700_000_000, 0)

	f.Fuzz(func(t *testing.T, ops []byte) {
		q := common.NewPendingMessageQueue()
		model := make(map[string]time.Time)

		newMsg := func(arg byte) *common.PendingMessage {
			return &common.PendingMessage{
				ReleaseTime: base.Add(time.Duration(arg/fuzzQueueMsgCount) * time.Second),
				Msg: common.MessagePublication{
					TxID:           make([]byte, common.TxIDLenMin),
					EmitterChain:   vaa.ChainIDEthereum,
					EmitterAddress: vaa.Address{0x01},
					Sequence:       uint64(arg % fuzzQueueMsgCount),
				},
			}
		}

		earliest := func() (time.Time, bool) {
			var first time.Time
			found := false
			for _, rt := range model {
				if !found || rt.Before(first) {
					first, found = rt, true
				}
			}
			return first, found
		}

		for i := 0; i+1 < len(ops); i += 2 {
			op, arg := ops[i], ops[i+1]
			pMsg := newMsg(arg)
			id := pMsg.Msg.MessageIDString()

			switch op % 4 {
			case 0:
				q.Push(pMsg)
				// Push ignores a message ID that is already queued.
				if _, ok := model[id]; !ok {
					model[id] = pMsg.ReleaseTime
				}
			case 1:
				when := pMsg.ReleaseTime
				got := q.Pop(when)
				first, ok := earliest()
				if !ok || first.After(when) {
					if got != nil {
						t.Fatalf("op %d: Pop(%v) returned %s, but no message is ready", i/2, when, got.Msg.MessageIDString())
					}
					break
				}
				if got == nil {
					t.Fatalf("op %d: Pop(%v) returned nil, but a message with release time %v is ready", i/2, when, first)
				}
				if !got.ReleaseTime.Equal(first) {
					t.Fatalf("op %d: Pop returned release time %v, want the earliest %v", i/2, got.ReleaseTime, first)
				}
				delete(model, got.Msg.MessageIDString())
			case 2:
				removed, err := q.RemoveItem(pMsg.Msg.MessageID())
				if err != nil {
					t.Fatalf("op %d: RemoveItem: %v", i/2, err)
				}
				_, inModel := model[id]
				if (removed != nil) != inModel {
					t.Fatalf("op %d: RemoveItem returned %v, but model has message: %v", i/2, removed, inModel)
				}
				delete(model, id)
			case 3:
				peeked := q.Peek()
				first, ok := earliest()
				if (peeked != nil) != ok {
					t.Fatalf("op %d: Peek returned %v, but model is empty: %v", i/2, peeked, !ok)
				}
				if ok && !peeked.ReleaseTime.Equal(first) {
					t.Fatalf("op %d: Peek returned release time %v, want the earliest %v", i/2, peeked.ReleaseTime, first)
				}
			}

			if q.Len() != len(model) {
				t.Fatalf("op %d: Len is %d, model has %d", i/2, q.Len(), len(model))
			}
		}
	})
}
