package notary

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"
)

// notaryState holds the delayed queue, the blackholed set, and the database that persists them.
//
// Every method takes mu for its whole body, so each state transition is atomic. mu keeps these
// invariants true between calls:
//   - A message is never both delayed and blackholed.
//   - The in-memory structures and the database hold the same messages, unless a database
//     operation returned an error.
//
// If a database operation fails, the in-memory state errs toward holding the message back: a
// message stays delayed or blackholed in memory even if the database write or delete failed.
//
// [common.PendingMessageQueue] has its own lock, but that lock protects only the queue's internal
// structure. It does not protect the invariants above. Code outside this type must not access
// delayed, blackholed, or database directly.
type notaryState struct {
	mu     sync.RWMutex
	logger *zap.Logger

	// database persists information about delayed and black-holed messages.
	database db.NotaryDBInterface

	// Min-heap queue of delayed messages (MessagePublication + Timestamp for release)
	delayed *common.PendingMessageQueue

	// All of the messages that have been black-holed due to being rejected by the Transfer Verifier.
	blackholed *msgPubSet
}

func newNotaryState(logger *zap.Logger, database db.NotaryDBInterface) *notaryState {
	return &notaryState{
		logger:     logger,
		database:   database,
		delayed:    common.NewPendingMessageQueue(),
		blackholed: NewSet(),
	}
}

// load reads all the database entries into the in-memory structures.
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
	if result == nil {
		s.logger.Error(
			"notary: LoadAll call produced nil result",
		)
		return errors.New("nil result from database")
	}

	s.logger.Info(
		"loaded notary data from database",
		zap.Int("delayedMsgs", len(result.Delayed)),
		zap.Int("blackholedMsgs", len(result.Blackholed)),
	)

	// Avoid overwriting data by mistake.
	if s.delayed.Len() > 0 || s.blackholed.Len() > 0 {
		return ErrAlreadyInitialized
	}

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

func (s *notaryState) isDelayed(msgID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.delayed.FetchMessagePublication(msgID) != nil
}

func (s *notaryState) isBlackholed(msgID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.blackholed.Contains(msgID)
}

// delay stores a MessagePublication in the database and adds it to the delayed queue.
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

		// Update database. Do this before adding the message to the ready list so that we don't
		// accidentally add the same message twice if deleting the message from the database fails.
		deletedPendingMsg, err := s.database.DeleteDelayed(pMsg.Msg.MessageID())
		if err != nil {
			s.logger.Error("delete pending message from notary database", pMsg.Msg.ZapFields(zap.Error(err))...)
			failedMsgs = append(failedMsgs, pMsg)
			continue
		}

		if deletedPendingMsg == nil {
			s.logger.Warn("notary: delete pending message from notary database: deleted value was nil")
		}

		// If the message is in the delayed queue, it should not be in the blackholed queue.
		// This is a sanity check to ensure that the blackholed queue is not published,
		// but it should never happen.
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

// reschedule sets the release time of an existing delayed message to now plus duration.
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

	// Remove existing message from the delayed list and database.
	if _, err := s.removeDelayedLocked(msgID); err != nil {
		return err
	}

	// Add the message back to the delayed list and database with the new duration.
	if err := s.delayLocked(msgPub, duration); err != nil {
		return err
	}

	s.logger.Info("notary: duration set for message", msgPub.ZapFields()...)
	return nil
}

// blackhole adds a message publication to the blackholed in-memory set and stores it in the database.
// It also removes the message from the delayed list and database, if present.
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

// unblackhole removes a message from the blackholed set and database and delays it with a delay of
// zero, so that it is released on the next cycle. Returns [ErrMsgNotFound] if the database has no
// blackholed message with this ID.
func (s *notaryState) unblackhole(msgID []byte) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	removedMsgPub, err := s.removeBlackholedLocked(msgID)
	if err != nil {
		return err
	}

	if removedMsgPub == nil {
		return ErrMsgNotFound
	}

	return s.delayLocked(removedMsgPub, time.Duration(0))
}

// forget removes a message from the database and from the delayed and blackholed lists.
func (s *notaryState) forget(msg *common.MessagePublication) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	removedPendingMsg, err := s.removeDelayedLocked(msg.MessageID())
	if err != nil {
		return err
	}

	removedMsgPub, err := s.removeBlackholedLocked(msg.MessageID())
	if err != nil {
		return err
	}

	if removedPendingMsg == nil && removedMsgPub == nil {
		s.logger.Info("notary: call to forget did not result in any changes", msg.ZapFields()...)
	}

	return nil
}

// delayLocked adds a message to the delayed queue and the database. The caller must hold mu.
func (s *notaryState) delayLocked(msg *common.MessagePublication, dur time.Duration) error {
	// Ensure that the message can't be added to the delayed list or database if it's already blackholed.
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

	// Store in in-memory slice. This should happen even if a database error occurs.
	s.delayed.Push(pMsg)

	// Store in database.
	if dbErr := s.database.StoreDelayed(pMsg); dbErr != nil {
		return dbErr
	}

	s.logger.Info("notary: delayed message", msg.ZapFields()...)

	return nil
}

// blackholeLocked removes a message from the delayed queue and database, if present, then adds it
// to the blackholed set and the database. The caller must hold mu.
func (s *notaryState) blackholeLocked(msg *common.MessagePublication) error {
	// The fetch call isn't strictly necessary, but it makes the code easier to reason
	// about given that removeDelayedLocked can return nil even if an error does not occur.
	if s.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		removedPendingMsg, err := s.removeDelayedLocked(msg.MessageID())
		if err != nil {
			return err
		}

		// Shouldn't happen, but checked for completeness.
		if removedPendingMsg == nil {
			return errors.New("notary: removeDelayed returned nil for removedPendingMsg")
		}
	}

	// Store in in-memory set. This should happen even if a database error occurs.
	s.blackholed.Add(msg.MessageID())

	// Store in database.
	if dbErr := s.database.StoreBlackholed(msg); dbErr != nil {
		return dbErr
	}

	s.logger.Info("notary: blackholed message", msg.ZapFields()...)

	return nil
}

// removeDelayedLocked removes a message from the delayed list and database. The caller must hold mu.
//
// Returns the removed message.
// Returns nil on error or if the message was not found.
func (s *notaryState) removeDelayedLocked(msgID []byte) (*common.PendingMessage, error) {
	removed, err := s.delayed.RemoveItem(msgID)
	if err != nil {
		return nil, err
	}

	deletedPendingMsg, err := s.database.DeleteDelayed(msgID)
	if err != nil {
		return nil, err
	}

	// no-op if the message is in neither the delayed list nor the database,
	// and no errors occurred.
	if removed == nil && deletedPendingMsg == nil {
		return nil, nil
	}

	return removed, nil
}

// removeBlackholedLocked removes a message from the blackholed list and database. The caller must
// hold mu.
//
// Returns the message that was removed from the database, or nil if an error occurred or the
// database did not contain the message.
func (s *notaryState) removeBlackholedLocked(msgID []byte) (*common.MessagePublication, error) {
	currLen := s.blackholed.Len()
	s.blackholed.Remove(msgID)
	removeOccurred := s.blackholed.Len() < currLen

	// Log if the message was not removed, then continue to try to delete it from the database
	// for consistency.
	if !removeOccurred {
		s.logger.Info("notary: call to removeBlackholed did not remove a message", zap.String("msgID", string(msgID)))
	} else {
		s.logger.Info("notary: removed blackholed message from in-memory set", zap.String("msgID", string(msgID)))
	}

	deletedMsgPub, err := s.database.DeleteBlackholed(msgID)
	if err != nil {
		return nil, err
	}

	// No-op if the message is not in the database.
	if deletedMsgPub == nil {
		return nil, nil
	}

	s.logger.Info("notary: removed blackholed message from database", deletedMsgPub.ZapFields()...)

	return deletedMsgPub, nil
}
