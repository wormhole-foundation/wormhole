// Notary evaluates the status of [common.MessagePublication]s and makes decisions regarding
// how they should be processed.
//
// Currently, it returns one of three possible verdicts:
// 1. Approve
//   - Messages should pass through normally.
//   - This verdict is used for any message that has a non-error status.
//
// 2. Delay
//   - Messages should be delayed.
//   - This verdict is used for Anomalous messages.
//
// 3. Blackhole
//   - Messages should be blocked from publication permanently, including for reobservation pathways.
//   - This status is reserved for messages with a Rejected status.
//
// The Notary does not modify message publications nor does it stop them from
// being processed. It only informs other code what to do. When a message is
// Delayed or Rejected, the Notary will track it in a database.
//
// Delayed messages are stored with a timestamp indicating when they should be
// released. After the timestamp expires, they can be removed from the
// database.
//
// Because Blackholed messages are meant to be blocked permanently, they should
// be stored in the database forever. In practice, messages will be marked as
// Rejected only in very extreme circumstances, so the database should always
// be small.
package notary

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

type (
	// Verdict is an enum that reports what action the Notary has taken after processing a message.
	Verdict uint8
)

const (
	Unknown Verdict = iota
	// Approve means a message should be processed normally. All messages that are not Token Transfers
	// must always be Approve, as the Notary does not support other message types.
	Approve
	// Delay means a message should be temporarily delayed so that it can be manually inspected.
	Delay
	// Blackhole means a message should be permanently blocked from being processed.
	Blackhole
)

func (v Verdict) String() string {
	switch v {
	case Approve:
		return "Approve"
	case Delay:
		return "Delay"
	case Blackhole:
		return "Blackhole"
	case Unknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

const (
	// How long a message should be held in the pending list before being processed.
	// The value should be long enough to allow for manual review and classification
	// by the Guardians.
	DefaultDelay = time.Hour * 24 * 4
	MaxDelayDays = 30
	MaxDelay     = time.Hour * 24 * MaxDelayDays
)

var (
	ErrAlreadyInitialized = errors.New("notary: message queues already initialized during database load")
	ErrAlreadyBlackholed  = errors.New("notary: message is already blackholed")
	ErrCannotRelease      = errors.New("notary: could not release message")
	ErrInvalidMsg         = errors.New("notary: message is invalid")
	ErrMsgNotFound        = errors.New("notary: message not found")
)

type (
	// A set corresponding to message publications. The elements of the set must be the results of
	// the function [common.MessagePublication.MessageIDString()].
	msgPubSet struct {
		elements map[string]struct{}
	}

	Notary struct {
		ctx    context.Context
		logger *zap.Logger
		mutex  sync.RWMutex
		// database persists information about delayed and black-holed messages.
		// Must be guarded by a read-write mutex.
		database db.NotaryDBInterface

		// Min-heap queue of delayed messages (MessagePublication + Timestamp for release)
		delayed *common.PendingMessageQueue

		// All of the messages that have been black-holed due to being rejected by the Transfer Verifier.
		// [msgPubSet] is not thread-safe so this field must be guarded by a read-write mutex.
		blackholed *msgPubSet

		// env reports whether the guardian is running in production or a test environment.
		env common.Environment
	}
)

func NewNotary(
	ctx context.Context,
	logger *zap.Logger,
	guardianDB *db.Database,
	env common.Environment,
) *Notary {
	return &Notary{
		ctx:    ctx,
		logger: logger,
		mutex:  sync.RWMutex{},
		// Get the underlying database connection from the Guardian.
		database:   db.NewNotaryDB(guardianDB.Conn()),
		delayed:    common.NewPendingMessageQueue(),
		blackholed: nil,
		env:        env,
	}
}

func (n *Notary) Run() error {
	if n.env != common.GoTest {
		n.logger.Info("loading notary data from database")
		if err := n.loadFromDB(n.logger); err != nil {
			return err
		}
	}

	n.logger.Info("notary ready")

	return nil
}

func (n *Notary) ProcessMsg(msg *common.MessagePublication) (v Verdict, err error) {

	n.logger.Debug("notary: processing message", msg.ZapFields()...)

	// NOTE: Only token transfers originated on Ethereum are currently considered.
	// For the initial implementation, the Notary only rules on messages based
	// on the Transfer Verifier. However, there is no technical barrier to
	// supporting other message types.
	if msg.EmitterChain != vaa.ChainIDEthereum {
		n.logger.Debug("notary: automatically approving message publication because it is not from Ethereum", msg.ZapFields()...)
		return Approve, nil
	}

	if !vaa.IsTransfer(msg.Payload) {
		n.logger.Debug("notary: automatically approving message publication because it is not a token transfer", msg.ZapFields()...)
		return Approve, nil
	}

	if tokenBridge, ok := sdk.KnownTokenbridgeEmitters[msg.EmitterChain]; !ok {
		// Return Unknown if the token bridge is not registered in the SDK.
		n.logger.Error("notary: unknown token bridge emitter", msg.ZapFields()...)
		return Unknown, errors.New("unknown token bridge emitter")
	} else {
		// Approve if the token transfer is not from the token bridge.
		// For now, the notary only rules on token transfers from the token bridge.
		if !bytes.Equal(msg.EmitterAddress.Bytes(), tokenBridge) {
			n.logger.Debug("notary: automatically approving message publication because it is not from the token bridge", msg.ZapFields()...)
			return Approve, nil
		}
	}

	// Return early if the message has already been blackholed. This is important in case a message
	// is reobserved or otherwise processed here more than once. An Anomalous message that becomes
	// delayed and later blackholed should not be able to be re-added to the Delayed queue.
	if n.IsBlackholed(msg.MessageID()) {
		n.logger.Warn("notary: got message publication that is already blackholed",
			msg.ZapFields(zap.String("verdict", Blackhole.String()))...,
		)
		return Blackhole, nil
	}

	switch msg.VerificationState() {
	// Both Anomalous and Rejected messages are delayed. In the future, we could consider blackholing
	// rejected messages, but for now, we are choosing the cautious approach of delaying VAA production
	// rather than rejecting them permanently.
	case common.Anomalous, common.Rejected:
		err = n.delay(msg, DefaultDelay)
		v = Delay
	case common.Valid:
		v = Approve
	case common.CouldNotVerify, common.NotVerified, common.NotApplicable:
		// NOTE: All other statuses are simply approved for now. In the future, it may be
		// desirable to log a warning if a [common.NotVerified] message is handled here, with
		// the idea that messages handled by the Notary must already have a non-default
		// status.
		n.logger.Debug("notary: got unexpected verification status for token transfer", msg.ZapFields()...)
		v = Approve
	}

	n.logger.Debug("notary result",
		msg.ZapFields(zap.String("verdict", v.String()))...,
	)
	return
}

// ReleaseReadyMessages removes messages from the database and the delayed queue if they are ready to
// be released. Returns the messages that are ready to be published.
func (n *Notary) ReleaseReadyMessages() []*common.MessagePublication {
	if n == nil || n.delayed == nil {
		return nil
	}

	n.logger.Debug(
		"notary: begin process ready message",
		zap.Int("delayedCount", n.delayed.Len()),
	)
	var (
		readyMsgs = make([]*common.MessagePublication, 0, n.delayed.Len())
		now       = time.Now()
	)

	// Pop elements from the queue until the release time is after the current time.
	// If errors occur, continue instead of returning early so that other messages
	// can still be processed.
	for n.delayed.Len() != 0 {
		next := n.delayed.Peek()
		if next == nil || next.ReleaseTime.After(now) {
			break // No more messages to process or next message not ready
		}

		// Pop reduces the length of n.delayed
		pMsg := n.delayed.Pop()
		if pMsg == nil {
			n.logger.Error("nil message after pop")
			continue // Skip if Pop returns nil (shouldn't happen if Peek worked)
		}

		// Update database. Do this before adding the message to the ready list so that we don't
		// accidentally add the same message twice if deleting the message from the database fails.
		deletedPendingMsg, err := n.database.DeleteDelayed(pMsg.Msg.MessageID())
		if err != nil {
			n.logger.Error("delete pending message from notary database", pMsg.Msg.ZapFields(zap.Error(err))...)
			continue
		}

		if deletedPendingMsg == nil {
			n.logger.Warn("notary: delete pending message from notary database: deleted value was nil")
		}

		// If the message is in the delayed queue, it should not be in the blackholed queue.
		// This is a sanity check to ensure that the blackholed queue is not published,
		// but it should never happen.
		if n.IsBlackholed(pMsg.Msg.MessageID()) {
			n.logger.Error("notary: got blackholed message in delayed queue", pMsg.Msg.ZapFields()...)
			continue
		}

		// Append return value.
		readyMsgs = append(readyMsgs, &pMsg.Msg)

	}

	n.logger.Debug(
		"notary: finish process ready message",
		zap.Int("readyCount", len(readyMsgs)),
		zap.Int("delayedCount", n.delayed.Len()),
	)

	return readyMsgs
}

// delay stores a MessagePublication in the database and populates its in-memory
// representation in the Notary.
// Acquires the mutex lock and unlocks when complete.
func (n *Notary) delay(msg *common.MessagePublication, dur time.Duration) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Ensure that the message can't be added to the delayed list or database if it's already blackholed.
	if n.blackholed.Contains(msg.MessageID()) {
		return ErrAlreadyBlackholed
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
	n.delayed.Push(pMsg)

	// Store in database.
	dbErr := n.database.StoreDelayed(pMsg)
	if dbErr != nil {
		return dbErr
	}

	n.logger.Info("notary: delayed message", msg.ZapFields()...)

	return dbErr
}

// blackhole adds a message publication to the blackholed in-memory set and stores it in the database.
// It also removes the message from the delayed list and database, if present.
// Acquires the mutex and unlocks when complete.
func (n *Notary) blackhole(msg *common.MessagePublication) error {

	if msg == nil {
		return ErrInvalidMsg
	}

	// Check if the message is already in the delayed list. If so, remove it, before
	// adding it to the blackholed list.

	// The fetch call isn't strictly necessary, but it makes the code easier to reason
	// about given that removeDelayed can return nil even if an error does not occur.
	if n.delayed.FetchMessagePublication(msg.MessageID()) != nil {
		removedPendingMsg, err := n.removeDelayed(msg.MessageID())
		if err != nil {
			return err
		}

		// Shouldn't happen, but checked for completeness.
		if removedPendingMsg == nil {
			return errors.New("notary: removeDelayed returned nil for removedPendingMsg")
		}
	}

	// Now blackhole the message
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Store in in-memory slice. This should happen even if a database error occurs.
	n.blackholed.Add(msg.MessageID())

	// Store in database.
	dbErr := n.database.StoreBlackholed(msg)
	if dbErr != nil {
		// Ensure the mutex is unlocked before returning.
		// Not using defer for unlocking here because removeDelayed acquires the mutex.
		n.mutex.Unlock()
		return dbErr
	}

	n.logger.Info("notary: blackholed message", msg.ZapFields()...)

	return nil
}

// forget removes a message from the database and from the delayed and blackholed lists.
func (n *Notary) forget(msg *common.MessagePublication) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	removedPendingMsg, err := n.removeDelayed(msg.MessageID())
	if err != nil {
		return err
	}

	removedMsgPub, err := n.removeBlackholed(msg.MessageID())
	if err != nil {
		return err
	}

	if removedPendingMsg == nil && removedMsgPub == nil {
		n.logger.Info("notary: call to forget did not result in any changes", msg.ZapFields()...)
	}

	return nil
}

// IsBlackholed returns true if the message is in the blackholed list.
func (n *Notary) IsBlackholed(msgID []byte) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.blackholed.Contains(msgID)
}

// removeBlackholed removes a message from the blackholed list and database.
// Returns the message that was removed or nil if an error occurred.
// Acquires the mutex and unlocks when complete.
func (n *Notary) removeBlackholed(msgID []byte) (*common.MessagePublication, error) {
	if len(msgID) == 0 {
		return nil, ErrInvalidMsg
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()

	currLen := n.blackholed.Len()
	n.blackholed.Remove(msgID)
	removeOccurred := n.blackholed.Len() < currLen

	// Log if the message was not removed, then continue to try to delete it from the database
	// for consistency.
	if !removeOccurred {
		n.logger.Info("notary: call to removeBlackholed did not remove a message", zap.String("msgID", string(msgID)))
	} else {
		n.logger.Info("notary: removed blackholed message from in-memory set", zap.String("msgID", string(msgID)))
	}

	deletedMsgPub, err := n.database.DeleteBlackholed(msgID)
	if err != nil {
		return nil, err
	}

	// No-op if the message is not in the database.
	if deletedMsgPub == nil {
		return nil, nil
	} else {
		n.logger.Info("notary: removed blackholed message from database", deletedMsgPub.ZapFields()...)
	}

	return deletedMsgPub, nil
}

func (n *Notary) IsDelayed(msg *common.MessagePublication) bool {
	// The notary's mutex is not used here because the pending message queue
	// uses its own read mutex for this method.
	return n.delayed.FetchMessagePublication(msg.MessageID()) != nil
}

// release sets the duration of an existing delayed message to zero so that it will be published on the next cycle.
func (n *Notary) release(msgID []byte) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}
	return n.setDuration(msgID, time.Duration(0))
}

// setDuration sets the duration of an existing delayed message to a new value.
func (n *Notary) setDuration(msgID []byte, duration time.Duration) error {
	if len(msgID) == 0 {
		return ErrInvalidMsg
	}

	// The notary's mutex is not used here because the pending message queue
	// uses its own read mutex for this method.
	msgPub := n.delayed.FetchMessagePublication(msgID)

	if msgPub == nil {
		return ErrMsgNotFound
	}

	// Remove existing message from the delayed list and database.
	deletedPendingMsg, removeErr := n.removeDelayed(msgID)
	if removeErr != nil {
		return removeErr
	}

	// Shouldn't happen, but log it for completeness.
	if deletedPendingMsg == nil {
		n.logger.Warn("notary: no pending message was removed during call to setDuration")
	}

	// Add the message back to the delayed list and database with the new duration.
	delayErr := n.delay(msgPub, duration)
	if delayErr != nil {
		return delayErr
	}

	n.logger.Info("notary: duration set for message", msgPub.ZapFields()...)
	return nil
}

// removeDelayed removes a message from the delayed list and database.
// This is a convenience function and should be equivalent to calling
// RemoveItem and DeleteDelayed separately for the same message ID.
//
// Returns the removed message.
// Returns nil on error or if the message was not found.
func (n *Notary) removeDelayed(msgID []byte) (*common.PendingMessage, error) {
	if len(msgID) == 0 {
		return nil, ErrInvalidMsg
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	removed, err := n.delayed.RemoveItem(msgID)
	if err != nil {
		return nil, err
	}

	deletedPendingMsg, err := n.database.DeleteDelayed(msgID)
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

// loadFromDB reads all the database entries.
func (n *Notary) loadFromDB(logger *zap.Logger) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	result, err := n.database.LoadAll(logger)
	if err != nil {
		n.logger.Error(
			"notary: LoadAll call returned error",
			zap.Error(err),
		)
		return err
	}
	if result == nil {
		n.logger.Error(
			"notary: LoadAll call produced nil result",
		)
		return errors.New("nil result from database")
	}

	n.logger.Info(
		"loaded notary data from database",
		zap.Int("delayedMsgs", len(result.Delayed)),
		zap.Int("blackholedMsgs", len(result.Blackholed)),
	)

	// Avoid overwriting data by mistake.
	if n.delayed != nil && n.delayed.Len() > 0 {
		return ErrAlreadyInitialized
	}

	var (
		delayed    = common.NewPendingMessageQueue()
		blackholed = NewSet()
	)

	if len(result.Delayed) > 0 {
		for entry := range slices.Values(result.Delayed) {
			delayed.Push(entry)
		}
	}

	if len(result.Blackholed) > 0 {
		for result := range slices.Values(result.Blackholed) {
			blackholed.Add(result.MessageID())
		}
	}

	n.blackholed = blackholed
	n.delayed = delayed
	n.logger.Info(
		"initialized notary",
		zap.Int("delayedMsgs", n.delayed.Len()),
		zap.Int("blackholedMsgs", n.blackholed.Len()),
	)

	return nil
}

// NewSet creates and initializes a new Set
func NewSet() *msgPubSet {
	// Keys are the message IDs, which are strings as []byte is not a valid type for a map key.
	return &msgPubSet{
		elements: make(map[string]struct{}),
	}
}

func (s *msgPubSet) Len() int {
	return len(s.elements)
}

// Add adds an element to the set
func (s *msgPubSet) Add(element []byte) {
	if s == nil {
		return // Protect against nil receiver
	}
	s.elements[string(element)] = struct{}{}
}

// Contains checks if an element is in the set
func (s *msgPubSet) Contains(element []byte) bool {
	if s == nil {
		return false // Protect against nil receiver
	}
	_, exists := s.elements[string(element)]
	return exists
}

// Remove removes an element from the set
func (s *msgPubSet) Remove(element []byte) {
	if s == nil {
		return // Protect against nil receiver
	}
	delete(s.elements, string(element))
}
