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
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

type (
	// Verdict is an enum that reports what action the Notary has taken after processing a message.
	Verdict uint8
)

const (
	// Approve means a message should be processed normally. All messages that are not Token Transfers
	// must always be Approve, as the Notary does not support other messasge types.
	Approve Verdict = iota
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
	default:
		return "Unknown"
	}
}

const (
	// How long a message should be held in the pending list before being processed.
	defaultDelay = time.Hour * 24
)

var (
	ErrCannotRelease      = errors.New("notary: could not release message")
	ErrAlreadyInitialized = errors.New("notary: message queues already initialized during database load")
)

type Notary struct {
	ctx    context.Context
	logger *zap.Logger
	// mutex guards database operations.
	mutex sync.Mutex
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
	// ready contains message publications that have been delayed but are now ready to release.
	ready []*common.MessagePublication
	// All of the messages that have been black-holed due to being rejected by the Transfer Verifier.
	blackholed []*common.MessagePublication

	// env reports whether the guardian is running in production or a test environment.
	env common.Environment
}

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
		ready:      nil,
		blackholed: nil,
		env:        env,
	}
}

func (n *Notary) Run(ctx context.Context) error {
	if n.env != common.GoTest {
		n.logger.Info("loading notary data from database")
		if err := n.loadFromDB(); err != nil {
			return err
		}
	}

	n.logger.Info("notary ready")

	return nil
}

func (n *Notary) ProcessMsg(msg *common.MessagePublication) (v Verdict, err error) {

	n.logger.Debug("notary: processing message", msg.ZapFields()...)

	// Only token transfers are currently supported.
	if !vaa.IsTransfer(msg.Payload) {
		return Approve, nil
	}

	switch msg.VerificationState() {
	case common.Anomalous:
		err = n.delay(msg, defaultDelay)
		v = Delay
	case common.Rejected:
		err = n.blackhole(msg)
		v = Blackhole
	default:
		// NOTE: All other statuses are simply approved for now. In the future, it may be
		// desirable to log a warning if a [common.NotVerified] message is handled here, with
		// the idea that messages handled by the Notary must already have a non-default
		// status.
		v = Approve
	}

	n.logger.Debug("notary result",
		msg.ZapFields(zap.String("verdict", v.String()))...,
	)
	return
}

// ProcessReadyMessages moves messages from the delayed queue to the ready queue if they are ready to
// be released.
func (n *Notary) ProcessReadyMessages() {
	if n == nil || n.delayed == nil {
		return // Avoid nil pointer dereference
	}

	n.logger.Debug(
		"notary: begin process ready message",
		zap.Int("readyCount", len(n.ready)),
		zap.Int("delayedCount", n.delayed.Len()),
	)
	now := time.Now()
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

		n.ready = append(n.ready, &pMsg.Msg)

		err := n.database.DeleteDelayed(pMsg)
		if err != nil {
			n.logger.Error("delete pending message from notary database", pMsg.Msg.ZapFields(zap.Error(err))...)
			continue
		}
	}

	n.logger.Debug(
		"notary: finish process ready message",
		zap.Int("readyCount", len(n.ready)),
		zap.Int("delayedCount", n.delayed.Len()),
	)

	return
}

// Releases a message publication held by the Notary and deletes it from the database.
func (n *Notary) Release(msg *common.MessagePublication) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if found := slices.Contains(n.ready, msg); !found {
		return errors.Join(
			ErrCannotRelease,
			errors.New("target message publication is not in the list of ready messages"),
		)
	}

	n.ready = slices.DeleteFunc(n.ready, func(element *common.MessagePublication) bool {
		return element == msg
	})

	return nil
}

// Shutdown stores pending messages to the database.
func (n *Notary) Shutdown() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Save ready messages back to the pending database. Store them with release time
	// equal to the current time so that they are marked ready on restart.
	var (
		now = time.Now()
		// if multiple errors occur, collect them with errors.Join and return all
		// of them at the end.
		errs error
	)

	for msg := range slices.Values(n.ready) {
		pMsg := &common.PendingMessage{
			Msg:         *msg,
			ReleaseTime: now,
		}
		err := n.database.StoreDelayed(pMsg)
		if err != nil {
			err = errors.Join(errs, err)
			continue
		}
	}

	for _, pMsg := range n.delayed.Iter() {
		err := n.database.StoreDelayed(pMsg)
		if err != nil {
			err = errors.Join(errs, err)
			continue
		}
	}

	return errs
}

// delay stores a MessagePublication in the database and populated its in-memory
// representation in the Notary.
// Acquires the mutex lock and unlocks when complete.
func (n *Notary) delay(msg *common.MessagePublication, dur time.Duration) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	pMsg := &common.PendingMessage{
		Msg:         *msg,
		ReleaseTime: time.Now().Add(dur),
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

// Acquires the mutex lock and unlocks when complete.
func (n *Notary) blackhole(msg *common.MessagePublication) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Store in in-memory slice. This should happen even if a database error occurs.
	n.blackholed = append(n.blackholed, msg)

	// Store in database.
	dbErr := n.database.StoreBlackhole(msg)
	if dbErr != nil {
		return dbErr
	}

	n.logger.Info("notary: blackholed message", msg.ZapFields()...)

	return nil
}

// Delayed returns a slice containing a copy of all delayed pending messages.
func (n *Notary) Delayed() (result []*common.PendingMessage) {
	if n.delayed == nil {
		return
	}
	if n.delayed.Len() == 0 {
		return
	}

	// Create a deep copy of the delayed messages.
	result = make([]*common.PendingMessage, n.delayed.Len())
	for i, pendingMsg := range n.delayed.Iter() {
		// Create a deep copy of each pending message.
		copied := *pendingMsg // Copy the struct
		result[i] = &copied
	}

	return
}

// Ready returns a copy of all ready pending messages.
func (n *Notary) Ready() []*common.MessagePublication {
	return deepCopy(n.ready)
}

// Blackholed returns a copy of all black-holed message publications.
func (n *Notary) Blackholed() []*common.MessagePublication {
	return deepCopy(n.blackholed)
}

func deepCopy(slice []*common.MessagePublication) []*common.MessagePublication {
	if len(slice) == 0 {
		return nil
	}
	result := make([]*common.MessagePublication, len(slice))
	for i, msg := range slice {
		// Create a deep copy of each publication
		copied := *msg // Copy the struct
		result[i] = &copied
	}
	return result
}

// loadFromDB reads all the database entries.
func (n *Notary) loadFromDB() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	result, err := n.database.LoadAll()
	if err != nil {
		return err
	}
	if result == nil {
		// TODO replace with better error
		return errors.New("nil result from database")
	}

	n.logger.Info(
		"loaded notary data from database",
		zap.Int("delayedMsgs", len(result.Delayed)),
		zap.Int("blackholedMsgs", len(result.Blackholed)),
	)

	// Avoid overwriting data by mistake.
	if (len(n.ready) > 0) || (n.delayed != nil && n.delayed.Len() > 0) {
		return ErrAlreadyInitialized
	}

	var (
		ready      []*common.MessagePublication
		delayed    = common.NewPendingMessageQueue()
		blackholed = make([]*common.MessagePublication, len(result.Blackholed))
		now        = time.Now()
	)

	if len(result.Delayed) > 0 {
		for entry := range slices.Values(result.Delayed) {
			if entry.ReleaseTime.Before(now) {
				ready = append(ready, &entry.Msg)
				continue
			}

			// If a message isn't ready, it's delayed.
			delayed.Push(entry)
		}
	}

	if len(result.Blackholed) > 0 {
		blackholed = result.Blackholed
	}

	n.blackholed = blackholed
	n.delayed = delayed
	n.ready = ready
	n.logger.Info(
		"initialized notary",
		zap.Int("delayedMsgs", n.delayed.Len()),
		zap.Int("blackholedMsgs", len(n.blackholed)),
		zap.Int("readyMsgs", len(n.ready)),
	)

	return nil
}
