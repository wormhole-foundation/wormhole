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
	// must always be Approve, as the Notary does not support other messasge types.
	Approve
	// Approved means a message should be temporarily delayed so that it can be manually inspected.
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
	DelayFor = time.Hour * 24 * 4
)

var (
	ErrAlreadyInitialized = errors.New("notary: message queues already initialized during database load")
	ErrAlreadyBlackholed  = errors.New("notary: message is already blackholed")
	ErrCannotRelease      = errors.New("notary: could not release message")
	ErrInvalidMsg         = errors.New("notary: message is invalid")
)

type (
	// A set corresponding to message publications. The elements of the set must be the results of
	// the function [common.MessagePublication.VAAHashUnchecked].
	msgPubSet struct {
		elements map[string]struct{}
	}

	Notary struct {
		ctx    context.Context
		logger *zap.Logger
		mutex  sync.Mutex
		// database persists information about delayed and black-holed messages.
		database db.NotaryDBInterface

		// Define slices to manage delayed and black-holed message publications.
		//
		// These fields are private so that this package is responsible for managing its own
		// state.
		//
		// In particular, the following invariants must hold:
		// - When a message is released, it must be deleted from the database.

		delayed *common.PendingMessageQueue
		// All of the messages that have been black-holed due to being rejected by the Transfer Verifier.
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
		mutex:  sync.Mutex{},
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

	if tokenBridge, ok := sdk.KnownTokenbridgeEmitters[msg.EmitterChain]; !ok {
		n.logger.Error("notary: unknown token bridge emitter", msg.ZapFields()...)
		return Unknown, errors.New("unknown token bridge emitter")
	} else {
		// Only token transfers are currently supported and the transfer must be from the token bridge.
		if !vaa.IsTransfer(msg.Payload) || !bytes.Equal(msg.EmitterAddress.Bytes(), tokenBridge) {
			return Approve, nil
		}
	}

	// Return early if the message has already been blackholed. This is important in case a message
	// is reobserved or otherwise processed here more than once. An Anomalous message that becomes
	// delayed and later blackholed should not be able to be re-added to the Delayed queue.
	if n.blackholed.Contains(msg.VAAHash()) {
		n.logger.Warn("notary: got message publication that is already blackholed",
			msg.ZapFields(zap.String("verdict", v.String()))...,
		)
		return Blackhole, nil
	}

	switch msg.VerificationState() {
	case common.Anomalous:
		err = n.delay(msg, DelayFor)
		v = Delay
	case common.Rejected:
		err = n.blackhole(msg)
		v = Blackhole
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

		// Append return value.
		readyMsgs = append(readyMsgs, &pMsg.Msg)

		// Update database.
		err := n.database.DeleteDelayed(pMsg)
		if err != nil {
			n.logger.Error("delete pending message from notary database", pMsg.Msg.ZapFields(zap.Error(err))...)
			continue
		}
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
	if n.blackholed.Contains(msg.VAAHash()) {
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
	n.mutex.Lock()

	// Store in in-memory slice. This should happen even if a database error occurs.
	n.blackholed.Add(msg.VAAHash())

	// Store in database.
	dbErr := n.database.StoreBlackholed(msg)
	if dbErr != nil {
		return dbErr
	}
	// Unlock mutex before calling removeDelayed, which also acquires the mutex.
	n.mutex.Unlock()

	// When a message is blackholed, it should be removed from the delayed list and database.
	err := n.removeDelayed(msg)
	if err != nil {
		return err
	}

	n.logger.Info("notary: blackholed message", msg.ZapFields()...)

	return nil
}

// forget removes a message from the database and from the delayed and blackholed lists.
func (n *Notary) forget(msg *common.MessagePublication) error {
	if msg == nil {
		return ErrInvalidMsg
	}

	err := n.removeDelayed(msg)
	if err != nil {
		return err
	}

	err = n.removeBlackholed(msg)
	if err != nil {
		return err
	}

	return nil
}

// Acquires the mutex and unlocks when complete.
func (n *Notary) IsBlackholed(msg *common.MessagePublication) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return n.blackholed.Contains(msg.VAAHash())
}

// removeBlackholed removes a message from the blackholed list and database.
func (n *Notary) removeBlackholed(msg *common.MessagePublication) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if msg == nil {
		return ErrInvalidMsg
	}

	n.blackholed.Remove(msg.VAAHash())

	err := n.database.DeleteBlackholed(msg)
	if err != nil {
		return err
	}

	n.logger.Info("notary: removed blackholed message", msg.ZapFields()...)

	return nil
}

// Acquires the mutex and unlocks when complete.
func (n *Notary) IsDelayed(msg *common.MessagePublication) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return n.delayed.ContainsMessagePublication(msg)
}

// removeDelayed removes a message from the delayed list and database.
func (n *Notary) removeDelayed(msg *common.MessagePublication) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if msg == nil {
		return ErrInvalidMsg
	}
	removed, err := n.delayed.RemoveItem(msg)
	if err != nil {
		return err
	}

	if removed != nil {
		err := n.database.DeleteDelayed(removed)
		if err != nil {
			return err
		}
	}
	return nil
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
			blackholed.Add(result.VAAHash())
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
	return &msgPubSet{
		elements: make(map[string]struct{}),
	}
}

func (s *msgPubSet) Len() int {
	return len(s.elements)
}

// Add adds an element to the set
func (s *msgPubSet) Add(element string) {
	if s == nil {
		return // Protect against nil receiver
	}
	s.elements[element] = struct{}{}
}

// Contains checks if an element is in the set
func (s *msgPubSet) Contains(element string) bool {
	if s == nil {
		return false // Protect against nil receiver
	}
	_, exists := s.elements[element]
	return exists
}

// Remove removes an element from the set
func (s *msgPubSet) Remove(element string) {
	if s == nil {
		return // Protect against nil receiver
	}
	delete(s.elements, element)
}
