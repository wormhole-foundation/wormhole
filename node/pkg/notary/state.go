package notary

import (
	"slices"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"
)

// notaryState holds the delayed queue, the blackholed set, and the database that persists them.
//
// [notaryState.mu] keeps these invariants true between calls:
//   - A message is never both delayed and blackholed.
//   - The in-memory structures and the database hold the same messages, unless a database
//     operation returned an error.
//
// If a database operation fails, the in-memory state errs toward holding the message back: a
// message stays delayed or blackholed in memory even if the database write or delete failed.
//
// Locking convention:
//   - A method without the Locked suffix takes [notaryState.mu] for its whole body, so it is one
//     atomic state transition. Code outside this type calls only these methods.
//   - A method with the Locked suffix requires the caller to hold [notaryState.mu] and does not take
//     it. A transition that has several steps, such as reschedule, calls Locked methods so that all
//     the steps happen under one lock.
//   - A Locked method must not call a method without the suffix. [sync.RWMutex] is not reentrant, so
//     that call would deadlock.
//
// [common.PendingMessageQueue] has its own lock, but that lock protects only the queue's internal
// structure. It does not protect the invariants above. Code outside this type must not access
// delayed, blackholed, or database directly.
type notaryState struct {
	mu     sync.RWMutex
	logger *zap.Logger

	// database persists the delayed and blackholed messages, so that they survive a restart.
	database db.NotaryDBInterface

	// delayed holds the delayed messages with their release times, earliest release first.
	delayed *common.PendingMessageQueue

	// blackholed holds the IDs of blackholed messages. It stores IDs only; the database stores the
	// messages.
	blackholed *msgPubSet
}

// newNotaryState returns an empty notaryState that persists to database. Call load to read the
// stored messages.
func newNotaryState(logger *zap.Logger, database db.NotaryDBInterface) *notaryState {
	return &notaryState{
		logger:     logger,
		database:   database,
		delayed:    common.NewPendingMessageQueue(),
		blackholed: NewSet(),
	}
}

// load reads all the database entries into the in-memory structures. Only [NewNotary] calls it, on
// a new, empty notaryState.
func (s *notaryState) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.database.LoadAll(s.logger)
	if err != nil {
		s.logger.Error(
			"notary: LoadAll call returned error",
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(
		"loaded notary data from database",
		zap.Int("delayedMsgs", len(result.Delayed)),
		zap.Int("blackholedMsgs", len(result.Blackholed)),
	)

	for entry := range slices.Values(result.Delayed) {
		s.delayed.Push(entry)
	}

	for msgPub := range slices.Values(result.Blackholed) {
		s.blackholed.Add(msgPub.MessageID())
	}

	s.logger.Info(
		"initialized notary",
		zap.Int("delayedMsgs", s.delayed.Len()),
		zap.Int("blackholedMsgs", s.blackholed.Len()),
	)

	return nil
}

// loadAll returns the database contents. It holds the read lock so that the result does not include
// a partially applied state transition.
//
// The admin read commands use the database rather than memory, because the blackholed set stores
// only message IDs, not the messages.
func (s *notaryState) loadAll() (*db.NotaryLoadResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.database.LoadAll(s.logger)
}

// lens returns the number of delayed and blackholed messages.
func (s *notaryState) lens() (delayed int, blackholed int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.delayed.Len(), s.blackholed.Len()
}

// isDelayed reports whether the delayed queue holds the message.
//
// The result is a snapshot. Another goroutine can change the state after isDelayed returns, so do
// not use the result to decide a state change. Put the check and the change in one method instead,
// as evaluate does.
func (s *notaryState) isDelayed(msgID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.delayed.FetchMessagePublication(msgID) != nil
}

// isBlackholed reports whether the blackholed set holds the message.
//
// The result is a snapshot, with the same limits as isDelayed.
func (s *notaryState) isBlackholed(msgID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.blackholed.Contains(msgID)
}

// evaluate returns the verdict for a token transfer that the Notary rules on. If the message is
// already blackholed or delayed, it returns Blackhole or Delay. Otherwise it returns verdict, and if
// verdict is Delay, it first delays the message for [DefaultDelay]. The error is the error from that
// delay, if any.
func (s *notaryState) evaluate(msg *common.MessagePublication, verdict Verdict) (Verdict, error) {
	if msg == nil {
		return Unknown, ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Return early if the message has already been blackholed. This is important in case a message
	// is reobserved or otherwise processed here more than once. An Anomalous message that becomes
	// delayed and later blackholed should not be able to be re-added to the delayed queue.
	if s.blackholed.Contains(msg.MessageID()) {
		s.logger.Warn("notary: got message publication that is already blackholed",
			msg.ZapFields(zap.String("verdict", Blackhole.String()))...,
		)
		return Blackhole, nil
	}

	// Return early if the message has already been delayed. A message may be processed here more
	// than once: reobservation re-runs the transfer verifier from scratch, so the same message can
	// re-enter with a different verification state. Without this check, a reobservation that
	// verifies as Valid would return Approve and bypass the remainder of the delay period.
	if s.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		s.logger.Warn("notary: got message publication that is already delayed",
			msg.ZapFields(zap.String("verdict", Delay.String()))...,
		)
		return Delay, nil
	}

	if verdict == Delay {
		return Delay, s.delayLocked(msg, DefaultDelay)
	}

	return verdict, nil
}

// delay adds a message to the delayed queue and the database, with a release time of now plus dur.
// See delayLocked for the errors.
func (s *notaryState) delay(msg *common.MessagePublication, dur time.Duration) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.delayLocked(msg, dur)
}

// releaseReady removes the messages whose release time is before or equal to now from the delayed
// queue and the database. Returns the messages that are ready to be published.
//
// If the database delete fails for a message, the message stays in the delayed queue and is not
// returned, so that the next call retries it.
func (s *notaryState) releaseReady(now time.Time) []*common.MessagePublication {
	// The lock is held across the database deletes so that memory and the database change together.
	// Admin commands wait for this call to finish.
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		readyMsgs = make([]*common.MessagePublication, 0, s.delayed.Len())
		// Messages that could not be deleted from the database. They are pushed back after the
		// loop, because pushing them back inside the loop would make Pop return them again.
		failedMsgs []*common.PendingMessage
	)

	// Pop elements from the queue until the release time is after now.
	// If errors occur, continue instead of returning early so that other messages
	// can still be processed.
	for {
		pMsg := s.delayed.Pop(now)

		if pMsg == nil {
			break // No more messages to process or next message not ready
		}

		// Delete from the database before returning the message. If the delete fails, the message
		// goes back in the queue after the loop and is not returned, so that it stays held.
		deletedPendingMsg, err := s.database.DeleteDelayed(pMsg.Msg.MessageID())
		if err != nil {
			s.logger.Error("delete pending message from notary database", pMsg.Msg.ZapFields(zap.Error(err))...)
			failedMsgs = append(failedMsgs, pMsg)
			continue
		}

		if deletedPendingMsg == nil {
			s.logger.Warn("notary: delete pending message from notary database: deleted value was nil")
		}

		// A delayed message is never blackholed, by the invariant on notaryState. This sanity
		// check makes sure that a blackholed message is never returned for publication.
		if s.blackholed.Contains(pMsg.Msg.MessageID()) {
			s.logger.Error("notary: got blackholed message in delayed queue", pMsg.Msg.ZapFields()...)
			continue
		}

		readyMsgs = append(readyMsgs, &pMsg.Msg)
	}

	for _, pMsg := range failedMsgs {
		s.delayed.Push(pMsg)
	}

	return readyMsgs
}

// reschedule sets the release time of a delayed message to now plus duration. Returns
// [ErrMsgNotFound] if the message is not delayed.
func (s *notaryState) reschedule(msgID []byte, duration time.Duration) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	msgPub := s.delayed.FetchMessagePublication(msgID)
	if msgPub == nil {
		return ErrMsgNotFound
	}

	// Remove the current entry from the database and the delayed queue.
	if err := s.removeDelayedLocked(msgID); err != nil {
		return err
	}

	// Add the message back with the new release time.
	if err := s.delayLocked(msgPub, duration); err != nil {
		return err
	}

	s.logger.Info("notary: duration set for message", msgPub.ZapFields()...)
	return nil
}

// blackhole adds a message to the blackholed set and the database. If the message is delayed, it
// first removes the message from the delayed queue and the database.
func (s *notaryState) blackhole(msg *common.MessagePublication) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.blackholeLocked(msg)
}

// blackholeDelayed moves a delayed message to the blackholed set. Returns [ErrMsgNotFound] if the
// message is not delayed.
func (s *notaryState) blackholeDelayed(msgID []byte) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	msgPub := s.delayed.FetchMessagePublication(msgID)
	if msgPub == nil {
		return ErrMsgNotFound
	}

	return s.blackholeLocked(msgPub)
}

// unblackhole removes a message from the blackholed set and the database, then delays it with a
// delay of zero, so that the next call to releaseReady returns it. Returns [ErrMsgNotFound] if the
// blackholed set does not hold the message.
func (s *notaryState) unblackhole(msgID []byte) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check memory first. The database returns an error, not ErrMsgNotFound, for a missing key.
	if !s.blackholed.Contains(msgID) {
		return ErrMsgNotFound
	}

	removedMsgPub, err := s.removeBlackholedLocked(msgID)
	if err != nil {
		return err
	}

	return s.delayLocked(removedMsgPub, time.Duration(0))
}

// forget removes a message from the delayed queue, the blackholed set, and the database. It is not
// an error if the message is in neither. Only tests call forget.
func (s *notaryState) forget(msg *common.MessagePublication) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove the message only from the queue or set that holds it. The database returns an error for
	// a missing key, so a removal from one that does not hold the message would fail.
	removedAny := false

	if s.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		if err := s.removeDelayedLocked(msg.MessageID()); err != nil {
			return err
		}
		removedAny = true
	}

	if s.blackholed.Contains(msg.MessageID()) {
		if _, err := s.removeBlackholedLocked(msg.MessageID()); err != nil {
			return err
		}
		removedAny = true
	}

	if !removedAny {
		s.logger.Info("notary: call to forget did not result in any changes", msg.ZapFields()...)
	}

	return nil
}

// delayLocked adds a message to the delayed queue and the database, with a release time of now plus
// dur. The caller must hold [notaryState.mu].
//
// Returns [ErrAlreadyBlackholed] or [ErrAlreadyDelayed] if memory already holds the message. If the
// database write fails, the message stays in the delayed queue and delayLocked returns the error.
func (s *notaryState) delayLocked(msg *common.MessagePublication, dur time.Duration) error {
	// A blackholed message must not also be delayed.
	if s.blackholed.Contains(msg.MessageID()) {
		return ErrAlreadyBlackholed
	}

	// The queue ignores a duplicate message, but the database would overwrite the stored release
	// time. Reject the message so that the queue and the database do not disagree.
	if s.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		return ErrAlreadyDelayed
	}

	// Remove nanoseconds from time.Now(). They are not serialized in the binary
	// representation. If we don't truncate nanoseconds here, then testing
	// message equality before and after loading to the database will fail.
	release := time.Unix(time.Now().Unix(), 0)

	pMsg := &common.PendingMessage{
		Msg:         *msg,
		ReleaseTime: release.Add(dur),
	}

	// Add to the queue before the database write, so that memory holds the message back even if
	// the write fails.
	s.delayed.Push(pMsg)

	// Store in database.
	if dbErr := s.database.StoreDelayed(pMsg); dbErr != nil {
		return dbErr
	}

	s.logger.Info("notary: delayed message", msg.ZapFields()...)

	return nil
}

// blackholeLocked removes a message from the delayed queue and the database, if it is delayed, then
// adds it to the blackholed set and the database. The caller must hold [notaryState.mu].
//
// If the database write fails, the message stays in the blackholed set and blackholeLocked returns
// the error.
func (s *notaryState) blackholeLocked(msg *common.MessagePublication) error {
	// Remove the message from the delayed queue only if it is there. The database returns an error
	// for a missing key.
	if s.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		if err := s.removeDelayedLocked(msg.MessageID()); err != nil {
			return err
		}
	}

	// Add to the set before the database write, so that memory holds the message back even if the
	// write fails.
	s.blackholed.Add(msg.MessageID())

	// Store in database.
	if dbErr := s.database.StoreBlackholed(msg); dbErr != nil {
		return dbErr
	}

	s.logger.Info("notary: blackholed message", msg.ZapFields()...)

	return nil
}

// removeDelayedLocked removes a message from the database and then from the delayed queue. The
// caller must hold [notaryState.mu], and the message must be in the database. The database returns
// an error for a missing key.
//
// The database delete happens first. If it fails, the message stays in the delayed queue, so that
// memory still holds back a message that the database holds.
func (s *notaryState) removeDelayedLocked(msgID []byte) error {
	if _, err := s.database.DeleteDelayed(msgID); err != nil {
		return err
	}

	_, err := s.delayed.RemoveItem(msgID)
	return err
}

// removeBlackholedLocked removes a message from the database and then from the blackholed set. The
// caller must hold [notaryState.mu], and the message must be in the database. The database returns
// an error for a missing key.
//
// The database delete happens first. If it fails, the message stays in the blackholed set, so that
// memory still holds back a message that the database holds.
//
// Returns the message that was removed from the database.
func (s *notaryState) removeBlackholedLocked(msgID []byte) (*common.MessagePublication, error) {
	deletedMsgPub, err := s.database.DeleteBlackholed(msgID)
	if err != nil {
		return nil, err
	}

	s.blackholed.Remove(msgID)
	s.logger.Info("notary: removed blackholed message", deletedMsgPub.ZapFields()...)

	return deletedMsgPub, nil
}
