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
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/txverifier"
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

	// The ticker interval for the notary's periodic metrics update.
	metricsUpdateInterval = time.Second * 30
)

var (
	ErrAlreadyBlackholed = errors.New("notary: message is already blackholed")
	ErrInvalidMsg        = errors.New("notary: message is invalid")
	ErrMsgNotFound       = errors.New("notary: message not found")
	ErrAlreadyDelayed    = errors.New("notary: message is already delayed")
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

		// state holds the delayed and blackholed messages and persists them in the database.
		// The Notary calls only its methods without the Locked suffix. Each of those is atomic.
		state *notaryState

		// env reports whether the guardian is running in production or a test environment.
		env common.Environment
	}
)

func NewNotary(
	ctx context.Context,
	logger *zap.Logger,
	guardianDB *db.Database,
	env common.Environment,
) (*Notary, error) {

	// Ensure that the essential components are not nil before proceeding.
	if guardianDB == nil || guardianDB.Conn() == nil || logger == nil {
		return nil, errors.New("notary: guardianDB and logger must not be nil")
	}

	// Get the underlying database connection from the Guardian.
	state := newNotaryState(logger, db.NewNotaryDB(guardianDB.Conn()))

	// Load persisted state before returning, so that a Notary is never usable without it.
	logger.Info("loading notary data from database")
	if err := state.load(); err != nil {
		return nil, fmt.Errorf("notary: load from database: %w", err)
	}

	return &Notary{
		ctx:    ctx,
		logger: logger,
		state:  state,
		env:    env,
	}, nil
}

// Run starts the Notary's background work. The Notary loads its persisted state in [NewNotary], so it
// can process messages before Run is called.
func (n *Notary) Run() {
	// Initialize and register Prometheus metrics when notary starts.
	// This ensures metrics are only registered when the notary is actually enabled.
	// Safe to call multiple times - will only initialize once.
	initMetrics(n.logger)

	n.logger.Info("notary ready")

	// Spawn a goroutine to periodically update prometheus gauge metrics
	go n.updateMetrics()
}

// updateMetrics runs periodically to update gauge metrics for queue sizes
func (n *Notary) updateMetrics() {
	ticker := time.NewTicker(metricsUpdateInterval)
	defer ticker.Stop()

	// Update metrics immediately on start
	n.updateGauges()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.updateGauges()
		}
	}
}

// updateGauges updates the prometheus gauge metrics
func (n *Notary) updateGauges() {
	// Safety check: ensure metrics are initialized (should always be true after Run() is called)
	if notaryDelayedMessagesGauge == nil || notaryBlackholedMessagesGauge == nil {
		n.logger.Warn("Notary metrics not initialized, skipping gauges update")
		return
	}

	delayedCount, blackholedCount := n.state.lens()
	notaryDelayedMessagesGauge.Set(float64(delayedCount))
	notaryBlackholedMessagesGauge.Set(float64(blackholedCount))
}

func (n *Notary) ProcessMsg(msg *common.MessagePublication) (v Verdict, err error) {
	if msg == nil {
		return Unknown, errors.New("notary: ProcessMsg called with nil message")
	}

	n.logger.Debug("notary: processing message", msg.ZapFields()...)

	// For the initial implementation, the Notary only rules on messages based
	// on the Transfer Verifier. However, there is no technical barrier to
	// supporting other message types.
	if !txverifier.IsSupported(msg.EmitterChain) {
		n.logger.Debug("notary: automatically approving message: sent from a chain without a transfer verifier implementation", msg.ZapFields()...)
		return Approve, nil
	}

	if !vaa.IsTransfer(msg.Payload) {
		n.logger.Debug("notary: automatically approving message: it is not a wrapped token transfer", msg.ZapFields()...)
		return Approve, nil
	}

	var tbEmitters = make(map[vaa.ChainID][]byte)
	switch n.env {
	case common.MainNet:
		tbEmitters = sdk.KnownTokenbridgeEmitters
	case common.TestNet:
		tbEmitters = sdk.KnownTestnetTokenbridgeEmitters
	case common.UnsafeDevNet:
		tbEmitters = sdk.KnownDevnetTokenbridgeEmitters
	case common.AccountantMock, common.GoTest:
	default:
		n.logger.Debug("skipping token bridge emitter check because environment is not mainnet or testnet")
	}

	// Perform emitter checks when outside of unit tests or mock environments
	if n.env == common.MainNet || n.env == common.TestNet || n.env == common.UnsafeDevNet {
		if tokenBridge, ok := tbEmitters[msg.EmitterChain]; !ok {
			// Return Unknown if the token bridge is not registered in the SDK.
			n.logger.Error("notary: unknown token bridge emitter", msg.ZapFields()...)
			if notaryErrors != nil {
				notaryErrors.WithLabelValues("unknown_token_bridge").Inc()
			}
			return Unknown, errors.New("unknown token bridge emitter")
		} else {
			// Approve if the token transfer is not from the token bridge.
			// For now, the notary only rules on token transfers from the token bridge.
			if !bytes.Equal(msg.EmitterAddress.Bytes(), tokenBridge) {
				n.logger.Debug("notary: automatically approving message publication because it is not from the token bridge", msg.ZapFields()...)
				return Approve, nil
			}
		}
	}

	// Choose the verdict for a message that is neither blackholed nor delayed.
	switch msg.VerificationState() {
	// Both Anomalous and Rejected messages are delayed. In the future, we could consider blackholing
	// rejected messages, but for now, we are choosing the cautious approach of delaying VAA production
	// rather than rejecting them permanently.
	// CouldNotVerify messages are also delayed. For the Transfer Verifier, this should only happen if the
	// message publication coming from the Core Bridge is malformed.
	case common.Anomalous, common.Rejected, common.CouldNotVerify:
		v = Delay
	case common.Valid:
		v = Approve
	case common.NotVerified, common.NotApplicable:
		// NOTE: All other statuses are simply approved for now. In the future, it may be
		// desirable to log a warning if a [common.NotVerified] message is handled here, with
		// the idea that messages handled by the Notary must already have a non-default
		// status.
		n.logger.Debug("notary: got unexpected verification status for token transfer", msg.ZapFields()...)
		v = Approve
	}

	// The state check and the delay happen under one lock. Separate IsBlackholed, IsDelayed, and
	// delay calls would let an admin command change the message's state between them.
	v, err = n.state.evaluate(msg, v)

	n.logger.Debug("notary result",
		msg.ZapFields(zap.String("verdict", v.String()))...,
	)

	// Track messages that receive non-Approve verdicts
	if v != Approve && notaryTokenTransferNonApprove != nil {
		notaryTokenTransferNonApprove.WithLabelValues(v.String()).Inc()
	}

	return
}

// ReleaseReadyMessages removes messages from the database and the delayed queue if they are ready to
// be released. Returns the messages that are ready to be published.
func (n *Notary) ReleaseReadyMessages() []*common.MessagePublication {
	n.logger.Debug("notary: begin process ready message")

	readyMsgs := n.state.releaseReady(time.Now())

	n.logger.Debug(
		"notary: finish process ready message",
		zap.Int("readyCount", len(readyMsgs)),
	)

	if notaryReleasedMessagesCounter != nil {
		notaryReleasedMessagesCounter.Add(float64(len(readyMsgs)))
	}
	return readyMsgs
}

// IsBlackholed reports whether the message is blackholed. The result is a snapshot; see
// notaryState.isDelayed.
func (n *Notary) IsBlackholed(msgID []byte) bool {
	return n.state.isBlackholed(msgID)
}

// IsDelayed reports whether the message is delayed. The result is a snapshot; see
// notaryState.isDelayed.
func (n *Notary) IsDelayed(msg *common.MessagePublication) bool {
	if msg == nil {
		return false
	}

	return n.state.isDelayed(msg.MessageID())
}

// NewSet creates and initializes a new Set
func NewSet() *msgPubSet {
	// Keys are the message IDs, which are strings as []byte is not a valid type for a map key.
	return &msgPubSet{
		elements: make(map[string]struct{}),
	}
}

// Len returns the number of elements in the set.
func (s *msgPubSet) Len() int {
	return len(s.elements)
}

// Add adds an element to the set
func (s *msgPubSet) Add(element []byte) {
	s.elements[string(element)] = struct{}{}
}

// Contains checks if an element is in the set
func (s *msgPubSet) Contains(element []byte) bool {
	_, exists := s.elements[string(element)]
	return exists
}

// Remove removes an element from the set
func (s *msgPubSet) Remove(element []byte) {
	delete(s.elements, string(element))
}
