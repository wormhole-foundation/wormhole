package notary

// Admin commands for the Notary. This defines a public API for the Notary to be used by the Guardians.
// This structure allows the admin-level commands to be implemented separately from the core Notary code.

import (
	"errors"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
)

var (
	ErrDelayExceedsMax = errors.New("notary: delay exceeds maximum")
	ErrInvalidMsgID    = errors.New("the message ID must be specified as \"chainId/emitterAddress/seqNum\"")
)

// BlackholeDelayedMsg adds a message publication to the blackholed in-memory set and stores it in the database.
// It also removes the message from the delayed list and database.
func (n *Notary) BlackholeDelayedMsg(msgID string) error {

	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	var (
		msgPub *common.MessagePublication
		bz     = []byte(msgID)
	)
	msgPub = n.delayed.FetchMessagePublication(bz)

	if msgPub == nil {
		return ErrMsgNotFound
	}

	// This method also takes care of removing the message from the delayed list and database.
	blackholeErr := n.blackhole(msgPub)
	if blackholeErr != nil {
		return blackholeErr
	}
	return nil
}

// ReleaseDelayedMsg removes a message from the delayed list and publishes it immediately.
func (n *Notary) ReleaseDelayedMsg(msgID string) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}
	err := n.release([]byte(msgID))
	if err != nil {
		return err
	}

	return nil
}

// RemoveBlackholedMsg removes a message from the blackholed list and adds it to the delayed list with a delay of zero,
// so that it will be published on the next cycle.
func (n *Notary) RemoveBlackholedMsg(msgID string) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	removedMsgPub, removeErr := n.removeBlackholed([]byte(msgID))
	if removeErr != nil {
		return removeErr
	}

	delayErr := n.delay(removedMsgPub, time.Duration(0))
	if delayErr != nil {
		return delayErr
	}

	return nil
}

// ResetReleaseTimer resets the release timer for a delayed message.
func (n *Notary) ResetReleaseTimer(msgID string, delayDays uint8) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	if delayDays > MaxDelayDays {
		return ErrDelayExceedsMax
	}

	const hoursInDay = time.Hour * 24
	delay := time.Duration(delayDays) * hoursInDay

	if delay > MaxDelay {
		return ErrDelayExceedsMax
	}

	err := n.setDuration([]byte(msgID), delay)
	if err != nil {
		return err
	}

	return nil
}
