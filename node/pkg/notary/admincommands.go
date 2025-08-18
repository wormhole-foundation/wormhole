package notary

// Admin commands for the Notary. This defines a public API for the Notary to be used by the Guardians.
// This structure allows the admin-level commands to be implemented separately from the core Notary code.

import (
	"github.com/certusone/wormhole/node/pkg/common"
)

// BlackholeDelayedMsg adds a message publication to the blackholed in-memory set and stores it in the database.
// It also removes the message from the delayed list and database.
func (n *Notary) BlackholeDelayedMsg(msgID string) error {

	var (
		msgPub *common.MessagePublication
		bz     = []byte(msgID)
	)
	msgPub = n.delayed.FetchMessagePublication(bz)

	if msgPub == nil {
		return ErrMsgNotFound
	}

	blackholeErr := n.blackhole(msgPub)
	if blackholeErr != nil {
		return blackholeErr
	}
	return nil
}

// ReleaseDelayedMsg removes a message from the delayed list and publishes it immediately.
func (n *Notary) ReleaseDelayedMsg(msgID string) error {
	err := n.release([]byte(msgID))
	if err != nil {
		return err
	}

	return nil
}

// RemoveBlackholedMsg removes a message from the blackholed list and deletes it from the database.
func (n *Notary) RemoveBlackholedMsg(msgID string) error {

	err := n.removeBlackholed([]byte(msgID))
	if err != nil {
		return err
	}

	return nil
}
