package notary

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotaryState_DelayRejectsDuplicate(t *testing.T) {
	s := newNotaryState(zap.NewNop(), MockNotaryDB{})
	msg := makeUniqueMessagePublication(t)

	require.NoError(t, s.delay(msg, time.Hour))

	// A second delay would overwrite the release time in the database but not in the queue.
	require.ErrorIs(t, s.delay(msg, 2*time.Hour), ErrAlreadyDelayed)
	require.Equal(t, 1, s.delayed.Len())
}

func TestNotaryState_UnblackholeMissingMessage(t *testing.T) {
	s := newNotaryState(zap.NewNop(), MockNotaryDB{})
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
		s := newNotaryState(zap.NewNop(), MockNotaryDB{})
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
	MockNotaryDB
	fail atomic.Bool
}

func (f *failingDeleteNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	if f.fail.Load() {
		return nil, errors.New("delete failed")
	}
	return nil, nil
}

// TestNotaryState_ReleaseReadyKeepsMessageWhenDeleteFails checks that a message whose database
// delete fails stays in the delayed queue and is released by a later call.
func TestNotaryState_ReleaseReadyKeepsMessageWhenDeleteFails(t *testing.T) {
	database := &failingDeleteNotaryDB{}
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
