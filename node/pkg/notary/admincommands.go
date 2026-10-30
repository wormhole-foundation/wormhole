package notary

// Admin commands for the Notary. This defines a public API for the Notary to be used by the Guardians.
// This structure allows the admin-level commands to be implemented separately from the core Notary code.
//
// TODO: This file uses LoadAll() for reads which is not efficient. However, the DB exists mostly to facilitate
// loading data on Guardian restarts and write operations are much more common than reads, so a Get is not implemented
// yet. (See also the Governor's use of the DB.) The read operations defined here are only used by admins and tests,
// and the total number of delayed and blackholed messages should always be small, so the performance impact is
// acceptable.

import (
	"encoding/binary"
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

// hoursInDay is the length of one day, for delays given in days.
const hoursInDay = time.Hour * 24

var (
	ErrDelayExceedsMax = errors.New("notary: delay exceeds maximum")
	ErrInvalidMsgID    = errors.New("notary: the message ID must be specified as \"chainId/emitterAddress/seqNum\"")
	ErrNotDevMode      = errors.New("notary: inject commands only available in dev mode")
)

// BlackholeDelayedMsg adds a message publication to the blackholed in-memory set and stores it in the database.
// It also removes the message from the delayed queue and database. Returns [ErrMsgNotFound] if the
// message is not delayed.
func (n *Notary) BlackholeDelayedMsg(msgID string) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	// This method also removes the message from the delayed queue and database.
	return n.state.blackholeDelayed([]byte(msgID))
}

// ReleaseDelayedMsg sets the release time of a delayed message to now, so that the next poll of
// [Notary.ReleaseReadyMessages] releases it.
func (n *Notary) ReleaseDelayedMsg(msgID string) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}
	// Set the delay to zero so that the next poll of ReleaseReadyMessages releases the message.
	return n.state.reschedule([]byte(msgID), time.Duration(0))
}

// RemoveBlackholedMsg removes a message from the blackholed set and adds it to the delayed queue with a
// delay of zero, so that the next poll of [Notary.ReleaseReadyMessages] releases it. Returns
// [ErrMsgNotFound] if the message is not blackholed.
func (n *Notary) RemoveBlackholedMsg(msgID string) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	return n.state.unblackhole([]byte(msgID))
}

// ResetReleaseTimer resets the release timer for a delayed message.
func (n *Notary) ResetReleaseTimer(msgID string, delayDays uint8) error {
	if len(msgID) < common.MinMsgIdLen {
		return ErrInvalidMsgID
	}

	if delayDays > MaxDelayDays {
		return ErrDelayExceedsMax
	}

	return n.state.reschedule([]byte(msgID), time.Duration(delayDays)*hoursInDay)
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

	delay := time.Duration(delayDays) * hoursInDay

	err := n.state.delay(msgPub, delay)
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

	err := n.state.blackhole(msgPub)
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

	result, err := n.state.loadAll()
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

	result, err := n.state.loadAll()
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
	result, err := n.state.loadAll()
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
	result, err := n.state.loadAll()
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
	binary.BigEndian.PutUint16(bz[65:67], uint16(payload.OriginChain))

	copy(bz[67:], payload.TargetAddress[:])
	binary.BigEndian.PutUint16(bz[99:101], uint16(payload.TargetChain))

	return bz
}
