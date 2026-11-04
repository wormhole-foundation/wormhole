package notary

import (
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotaryState_DelayRejectsDuplicate(t *testing.T) {
	s := newNotaryState(zap.NewNop(), newMockNotaryDB())
	msg := makeUniqueMessagePublication(t)

	require.NoError(t, s.delay(msg, time.Hour))

	// A second delay would overwrite the release time in the database but not in the queue.
	require.ErrorIs(t, s.delay(msg, 2*time.Hour), ErrAlreadyDelayed)
	require.Equal(t, 1, s.delayed.Len())
}

func TestNotaryState_UnblackholeMissingMessage(t *testing.T) {
	s := newNotaryState(zap.NewNop(), newMockNotaryDB())
	msg := makeUniqueMessagePublication(t)

	require.ErrorIs(t, s.unblackhole(msg.MessageID()), ErrMsgNotFound)

	delayedCount, blackholedCount := s.lens()
	require.Zero(t, delayedCount)
	require.Zero(t, blackholedCount)
}

// TestNotaryState_RescheduleCompetesWithReleaseReady checks that a message that is ready for release
// is either released or rescheduled, never both. If reschedule did not hold the lock across its
// lookup, removal, and insert, releaseReady could release the message between those steps, and
// reschedule would then queue it again.
func TestNotaryState_RescheduleCompetesWithReleaseReady(t *testing.T) {
	const iterations = 100

	for range iterations {
		s := newNotaryState(zap.NewNop(), newMockNotaryDB())
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, s.delay(msg, 0))

		// Every message delayed with a duration of zero is ready at this time.
		now := time.Now().Add(time.Second)

		var (
			released      []*common.MessagePublication
			rescheduleErr error
			wg            sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			released = s.releaseReady(now)
		}()
		go func() {
			defer wg.Done()
			rescheduleErr = s.reschedule(msg.MessageID(), time.Hour)
		}()
		wg.Wait()

		delayedCount, _ := s.lens()
		if len(released) == 1 {
			// releaseReady won the lock. The message is gone, so reschedule can't find it.
			require.ErrorIs(t, rescheduleErr, ErrMsgNotFound)
			require.Zero(t, delayedCount)
		} else {
			// reschedule won the lock. The new release time is in the future.
			require.NoError(t, rescheduleErr)
			require.Empty(t, released)
			require.Equal(t, 1, delayedCount)
		}
	}
}

// failingDeleteNotaryDB fails every DeleteDelayed call until fail is set to false.
type failingDeleteNotaryDB struct {
	*MockNotaryDB
	fail atomic.Bool
}

func (f *failingDeleteNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	if f.fail.Load() {
		return nil, errors.New("delete failed")
	}
	return f.MockNotaryDB.DeleteDelayed(msgID)
}

// TestNotaryState_ReleaseReadyKeepsMessageWhenDeleteFails checks that a message whose database
// delete fails stays in the delayed queue and is released by a later call.
func TestNotaryState_ReleaseReadyKeepsMessageWhenDeleteFails(t *testing.T) {
	database := &failingDeleteNotaryDB{MockNotaryDB: newMockNotaryDB()}
	database.fail.Store(true)

	s := newNotaryState(zap.NewNop(), database)
	msg := makeUniqueMessagePublication(t)
	require.NoError(t, s.delay(msg, 0))

	// Every message delayed with a duration of zero is ready at this time.
	now := time.Now().Add(time.Second)

	require.Empty(t, s.releaseReady(now))
	require.True(t, s.isDelayed(msg.MessageID()), "message must stay delayed while the database still holds it")

	database.fail.Store(false)

	released := s.releaseReady(now)
	require.Len(t, released, 1)
	require.Equal(t, msg.MessageIDString(), released[0].MessageIDString())
	require.False(t, s.isDelayed(msg.MessageID()))
}

// TestNotaryState_EvaluateCompetesWithBlackholeDelayed checks that a reobserved Valid message is never
// approved while an admin command moves it from the delayed queue to the blackholed set. If evaluate
// checked the set and the queue in separate lock sections, it could check the set before the move and
// the queue after it, see neither, and approve a message that the admin blackholed.
func TestNotaryState_EvaluateCompetesWithBlackholeDelayed(t *testing.T) {
	const iterations = 100

	for range iterations {
		s := newNotaryState(zap.NewNop(), newMockNotaryDB())
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, s.delay(msg, time.Hour))

		var (
			verdict      Verdict
			evaluateErr  error
			blackholeErr error
			wg           sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			verdict, evaluateErr = s.evaluate(msg, Approve)
		}()
		go func() {
			defer wg.Done()
			blackholeErr = s.blackholeDelayed(msg.MessageID())
		}()
		wg.Wait()

		require.NoError(t, evaluateErr)
		require.NoError(t, blackholeErr)
		require.Contains(t, []Verdict{Delay, Blackhole}, verdict)
		require.True(t, s.isBlackholed(msg.MessageID()))
	}
}

// TestNotaryState_ConcurrentOperations runs every notaryState method from several goroutines at the
// same time. Run it with -race: a method that does not take [notaryState.mu] races on the
// blackholed set or the database. After the goroutines finish, it checks the state invariants.
func TestNotaryState_ConcurrentOperations(t *testing.T) {
	const (
		goroutines = 8
		opsEach    = 300
		msgCount   = 4
	)

	database := newMockNotaryDB()
	s := newNotaryState(zap.NewNop(), database)

	base := makeUniqueMessagePublication(t)
	msgs := make([]*common.MessagePublication, msgCount)
	for i := range msgs {
		m := *base
		m.Sequence = base.Sequence + uint64(i)
		msgs[i] = &m
	}

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// #nosec G404 G115 -- test randomness only, and g is a small non-negative loop index.
			r := rand.New(rand.NewPCG(uint64(g), 0))
			for range opsEach {
				msg := msgs[r.IntN(msgCount)]
				id := msg.MessageID()
				dur := time.Duration(r.IntN(3)) * time.Hour
				switch r.IntN(13) {
				case 0:
					_ = s.delay(msg, dur)
				case 1:
					_ = s.blackhole(msg)
				case 2:
					_ = s.blackholeDelayed(id)
				case 3:
					_ = s.unblackhole(id)
				case 4:
					_ = s.reschedule(id, dur)
				case 5:
					_ = s.forget(msg)
				case 6:
					_ = s.releaseReady(time.Now().Add(dur))
				case 7:
					_, _ = s.evaluate(msg, Approve)
				case 8:
					_, _ = s.evaluate(msg, Delay)
				case 9:
					_ = s.isDelayed(id)
				case 10:
					_ = s.isBlackholed(id)
				case 11:
					_, _ = s.lens()
				case 12:
					_, _ = s.loadAll()
				}
			}
		}()
	}
	wg.Wait()

	result, err := database.LoadAll(zap.NewNop())
	require.NoError(t, err)
	inDB := make(map[string]bool)
	for _, p := range result.Delayed {
		inDB[p.Msg.MessageIDString()] = true
	}
	for _, m := range result.Blackholed {
		inDB[m.MessageIDString()] = true
	}

	for _, msg := range msgs {
		delayed := s.isDelayed(msg.MessageID())
		blackholed := s.isBlackholed(msg.MessageID())
		require.False(t, delayed && blackholed, "%s is both delayed and blackholed", msg.MessageIDString())
		require.Equal(t, inDB[msg.MessageIDString()], delayed || blackholed,
			"memory and database disagree about %s", msg.MessageIDString())
	}
}

// TestNotaryState_UnblackholeDelaysMessage checks that unblackhole moves the message to the delayed
// queue with a delay of zero, so that the next release returns it.
func TestNotaryState_UnblackholeDelaysMessage(t *testing.T) {
	s := newNotaryState(zap.NewNop(), newMockNotaryDB())
	msg := makeUniqueMessagePublication(t)
	require.NoError(t, s.blackhole(msg))

	require.NoError(t, s.unblackhole(msg.MessageID()))
	require.False(t, s.isBlackholed(msg.MessageID()))
	require.True(t, s.isDelayed(msg.MessageID()))

	released := s.releaseReady(time.Now().Add(time.Second))
	require.Len(t, released, 1)
	require.Equal(t, msg.MessageIDString(), released[0].MessageIDString())
}

// failingStoreNotaryDB fails every StoreDelayed and StoreBlackholed call.
type failingStoreNotaryDB struct {
	*MockNotaryDB
}

var errStoreFailed = errors.New("store failed")

func (f *failingStoreNotaryDB) StoreDelayed(*common.PendingMessage) error { return errStoreFailed }
func (f *failingStoreNotaryDB) StoreBlackholed(*common.MessagePublication) error {
	return errStoreFailed
}

// TestNotaryState_StoreErrorsReturned checks that a database store error reaches the caller, and that
// memory still holds the message back.
func TestNotaryState_StoreErrorsReturned(t *testing.T) {
	t.Run("delay", func(t *testing.T) {
		s := newNotaryState(zap.NewNop(), &failingStoreNotaryDB{newMockNotaryDB()})
		msg := makeUniqueMessagePublication(t)

		require.ErrorIs(t, s.delay(msg, time.Hour), errStoreFailed)
		require.True(t, s.isDelayed(msg.MessageID()))
	})

	t.Run("blackhole", func(t *testing.T) {
		s := newNotaryState(zap.NewNop(), &failingStoreNotaryDB{newMockNotaryDB()})
		msg := makeUniqueMessagePublication(t)

		require.ErrorIs(t, s.blackhole(msg), errStoreFailed)
		require.True(t, s.isBlackholed(msg.MessageID()))
	})

	t.Run("reschedule", func(t *testing.T) {
		mock := newMockNotaryDB()
		s := newNotaryState(zap.NewNop(), mock)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, s.delay(msg, time.Hour))

		// Fail the store that reschedule makes after it deletes the old entry.
		s.database = &failingStoreNotaryDB{mock}
		require.ErrorIs(t, s.reschedule(msg.MessageID(), 2*time.Hour), errStoreFailed)
		require.True(t, s.isDelayed(msg.MessageID()))
	})
}
