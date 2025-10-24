package notary

// Admin commands for the Notary. This defines a public API for the Notary to be used by the Guardians.
// This structure allows the admin-level commands to be implemented separately from the core Notary code.
//
// TODO: This file uses LoadAll() for reads which is not efficient. However, the DB is exists mostly to facilitate
// loading data on Guardian restarts and write operations are much more common than reads, so a Get is not implemented
// yet. (See also the Governor's use of the DB.) The read operations defined here are only used by admins and tests,
// and the total number of delayed and blackholed messages should always be small, so the performance impact is
// acceptable.

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	ErrDelayExceedsMax = errors.New("notary: delay exceeds maximum")
	ErrInvalidMsgID    = errors.New("notary: the message ID must be specified as \"chainId/emitterAddress/seqNum\"")
	ErrNotDevMode      = errors.New("notary: inject commands only available in dev mode")
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

// InjectDelayedMessage creates a synthetic delayed message for testing (dev mode only).
func (n *Notary) InjectDelayedMessage(delayDays uint32) (string, error) {
	if n.env != common.UnsafeDevNet && n.env != common.GoTest {
		return "", ErrNotDevMode
	}

	if delayDays > MaxDelayDays {
		return "", ErrDelayExceedsMax
	}

	msgPub := createTestMessagePublication()
	msgID := msgPub.MessageIDString()

	const hoursInDay = time.Hour * 24
	delay := time.Duration(delayDays) * hoursInDay

	err := n.delay(msgPub, delay)
	if err != nil {
		return "", fmt.Errorf("failed to inject delayed message: %w", err)
	}

	return msgID, nil
}

// InjectBlackholedMessage creates a synthetic blackholed message for testing (dev mode only).
func (n *Notary) InjectBlackholedMessage() (string, error) {
	if n.env != common.UnsafeDevNet && n.env != common.GoTest {
		return "", ErrNotDevMode
	}

	msgPub := createTestMessagePublication()
	msgID := msgPub.MessageIDString()

	err := n.blackhole(msgPub)
	if err != nil {
		return "", fmt.Errorf("failed to inject blackholed message: %w", err)
	}

	return msgID, nil
}

// GetDelayedMessage retrieves details about a delayed message from the database.
func (n *Notary) GetDelayedMessage(msgID string) (*common.PendingMessage, error) {
	if len(msgID) < common.MinMsgIdLen {
		return nil, ErrInvalidMsgID
	}

	n.mutex.RLock()
	defer n.mutex.RUnlock()

	result, err := n.database.LoadAll(n.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load from database: %w", err)
	}

	for _, pendingMsg := range result.Delayed {
		if pendingMsg.Msg.MessageIDString() == msgID {
			return pendingMsg, nil
		}
	}

	return nil, ErrMsgNotFound
}

// GetBlackholedMessage retrieves details about a blackholed message from the database.
func (n *Notary) GetBlackholedMessage(msgID string) (*common.MessagePublication, error) {
	if len(msgID) < common.MinMsgIdLen {
		return nil, ErrInvalidMsgID
	}

	n.mutex.RLock()
	defer n.mutex.RUnlock()

	result, err := n.database.LoadAll(n.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load from database: %w", err)
	}

	for _, msgPub := range result.Blackholed {
		if msgPub.MessageIDString() == msgID {
			return msgPub, nil
		}
	}

	return nil, ErrMsgNotFound
}

// ListDelayedMessages returns all delayed message IDs from the database.
func (n *Notary) ListDelayedMessages() ([]string, error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	result, err := n.database.LoadAll(n.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load delayed messages from database: %w", err)
	}

	msgIDs := make([]string, 0, len(result.Delayed))
	for _, pendingMsg := range result.Delayed {
		msgIDs = append(msgIDs, pendingMsg.Msg.MessageIDString())
	}

	return msgIDs, nil
}

// ListBlackholedMessages returns all blackholed message IDs from the database.
func (n *Notary) ListBlackholedMessages() ([]string, error) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	result, err := n.database.LoadAll(n.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load blackholed messages from database: %w", err)
	}

	msgIDs := make([]string, 0, len(result.Blackholed))
	for _, msgPub := range result.Blackholed {
		msgIDs = append(msgIDs, msgPub.MessageIDString())
	}

	return msgIDs, nil
}

// createTestMessagePublication creates a unique test message for injection.
func createTestMessagePublication() *common.MessagePublication {
	originAddress, _ := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	targetAddress, _ := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")

	tokenBridge := sdk.KnownTokenbridgeEmitters[vaa.ChainIDEthereum]
	tokenBridgeAddress := vaa.Address(tokenBridge)

	payload := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}
	payloadBytes := encodePayloadBytes(payload)

	//#nosec G404: this value is only used in devnet mode and only for synthetic data. Cryptographic randomness is not required.
	sequence := rand.Uint64()
	msgPub := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Now(),
		Nonce:            123456,
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes,
		ConsistencyLevel: 32,
		Unreliable:       true,
		IsReobservation:  true,
	}

	return msgPub
}

// encodePayloadBytes encodes a TransferPayloadHdr into bytes. This is a utility function for testing.
func encodePayloadBytes(payload *vaa.TransferPayloadHdr) []byte {
	bz := make([]byte, 101)
	bz[0] = payload.Type

	amtBytes := payload.Amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}

	offset := 32 - len(amtBytes)
	copy(bz[1+offset:33], amtBytes)

	copy(bz[33:], payload.OriginAddress[:])
	bz[65] = byte(payload.OriginChain)

	copy(bz[66:], payload.TargetAddress[:])
	bz[98] = byte(payload.TargetChain)

	return bz
}
