package notary

import (
	"errors"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"
)

var errInjected = errors.New("injected database failure")

// faultyNotaryDB wraps a real Notary database. When fail is true, every write and delete returns
// errInjected and does not change the database.
type faultyNotaryDB struct {
	db.NotaryDBInterface
	fail bool
}

func (f *faultyNotaryDB) StoreBlackholed(m *common.MessagePublication) error {
	if f.fail {
		return errInjected
	}
	return f.NotaryDBInterface.StoreBlackholed(m)
}

func (f *faultyNotaryDB) StoreDelayed(p *common.PendingMessage) error {
	if f.fail {
		return errInjected
	}
	return f.NotaryDBInterface.StoreDelayed(p)
}

func (f *faultyNotaryDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	if f.fail {
		return nil, errInjected
	}
	return f.NotaryDBInterface.DeleteBlackholed(msgID)
}

func (f *faultyNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	if f.fail {
		return nil, errInjected
	}
	return f.NotaryDBInterface.DeleteDelayed(msgID)
}

const (
	fuzzMsgCount = 4

	// fuzzMaxOps limits the operations per input. Longer inputs add run time but rarely reach a new
	// state with only fuzzMsgCount messages.
	fuzzMaxOps = 64
)

// FuzzNotaryStateOperations runs a sequence of notaryState operations against a real in-memory
// database and checks the invariants after each one.
//
// Each operation is two bytes: an opcode and an argument. The argument selects one of
// fuzzMsgCount messages and a duration in hours.
func FuzzNotaryStateOperations(f *testing.F) {
	f.Add([]byte{0, 0, 2, 0, 3, 0, 7, 0})           // delay, blackholeDelayed, unblackhole, evaluate
	f.Add([]byte{0, 4, 4, 0, 6, 255, 7, 0})         // delay, reschedule, releaseReady, evaluate
	f.Add([]byte{0, 0, 8, 0, 2, 0, 8, 0, 7, 0})     // delay, fail, blackholeDelayed, recover, evaluate
	f.Add([]byte{1, 1, 8, 0, 3, 1, 8, 0, 7, 1})     // blackhole, fail, unblackhole, recover, evaluate
	f.Add([]byte{0, 2, 8, 0, 6, 255, 8, 0, 6, 255}) // delay, fail, releaseReady, recover, releaseReady

	logger := zap.NewNop()
	guardianDB := db.OpenDb(logger, nil)
	f.Cleanup(func() { _ = guardianDB.Close() })

	f.Fuzz(func(t *testing.T, ops []byte) {
		if len(ops) > 2*fuzzMaxOps {
			ops = ops[:2*fuzzMaxOps]
		}

		notaryDB := db.NewNotaryDB(guardianDB.Conn())
		database := &faultyNotaryDB{NotaryDBInterface: notaryDB}
		s := newNotaryState(logger, database)

		// Messages that differ only in sequence. makeUniqueMessagePublication picks a random
		// sequence, so each input uses its own keys in the shared database.
		base := makeUniqueMessagePublication(t)
		msgs := make([]*common.MessagePublication, fuzzMsgCount)
		for i := range msgs {
			m := *base
			m.Sequence = base.Sequence + uint64(i)
			msgs[i] = &m
		}

		// Delete this input's keys, so that the shared database does not grow across inputs and
		// slow down LoadAll. A missing key returns an error, which is expected here.
		t.Cleanup(func() {
			for _, m := range msgs {
				_, _ = notaryDB.DeleteDelayed(m.MessageID())
				_, _ = notaryDB.DeleteBlackholed(m.MessageID())
			}
		})

		now := time.Now()
		for i := 0; i+1 < len(ops); i += 2 {
			op, arg := ops[i], ops[i+1]
			msg := msgs[int(arg)%fuzzMsgCount]
			dur := time.Duration(arg/fuzzMsgCount) * time.Hour
			wasDelayed := s.isDelayed(msg.MessageID())
			wasBlackholed := s.isBlackholed(msg.MessageID())

			switch op % 9 {
			case 0:
				_ = s.delay(msg, dur)
			case 1:
				_ = s.blackhole(msg)
			case 2:
				_ = s.blackholeDelayed(msg.MessageID())
			case 3:
				_ = s.unblackhole(msg.MessageID())
			case 4:
				_ = s.reschedule(msg.MessageID(), dur)
			case 5:
				_ = s.forget(msg)
			case 6:
				cutoff := now.Add(dur)
				for _, released := range s.releaseReady(cutoff) {
					if s.isDelayed(released.MessageID()) || s.isBlackholed(released.MessageID()) {
						t.Fatalf("op %d: released message %s is still held in memory", i/2, released.MessageIDString())
					}
				}
			case 7:
				verdict, err := s.evaluate(msg, Approve)
				switch {
				case wasBlackholed && verdict != Blackhole:
					t.Fatalf("op %d: evaluate returned %s for a blackholed message (err %v)", i/2, verdict, err)
				case wasDelayed && verdict != Delay:
					t.Fatalf("op %d: evaluate returned %s for a delayed message (err %v)", i/2, verdict, err)
				}
			case 8:
				database.fail = !database.fail
			}

			checkStateInvariants(t, i/2, s, database, msgs)
		}
	})
}

// checkStateInvariants checks, for each message:
//   - It is never both delayed and blackholed in memory.
//   - If the database delays or blackholes it, memory also holds it back. A database error can make
//     memory hold more than the database, but never less.
func checkStateInvariants(t *testing.T, step int, s *notaryState, database *faultyNotaryDB, msgs []*common.MessagePublication) {
	t.Helper()

	result, err := database.NotaryDBInterface.LoadAll(zap.NewNop())
	if err != nil {
		t.Fatalf("step %d: LoadAll: %v", step, err)
	}
	inDB := make(map[string]string)
	for _, p := range result.Delayed {
		inDB[p.Msg.MessageIDString()] = "delayed"
	}
	for _, m := range result.Blackholed {
		inDB[m.MessageIDString()] = "blackholed"
	}

	for _, msg := range msgs {
		delayed := s.isDelayed(msg.MessageID())
		blackholed := s.isBlackholed(msg.MessageID())
		if delayed && blackholed {
			t.Fatalf("step %d: %s is both delayed and blackholed in memory", step, msg.MessageIDString())
		}
		if dbState, ok := inDB[msg.MessageIDString()]; ok && !delayed && !blackholed {
			t.Fatalf("step %d: database has %s as %s, but memory holds it in neither set", step, msg.MessageIDString(), dbState)
		}
	}
}
