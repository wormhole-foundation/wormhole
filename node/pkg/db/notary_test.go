package db

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestStoreAndReloadData(t *testing.T) {
	// Set-up.
	dbPath := t.TempDir()
	database := OpenDb(zap.NewNop(), &dbPath)
	defer database.Close()
	defer os.Remove(dbPath)
	nDB := NotaryDB{db: database.db}

	// Build messages.
	msg1 := makeNewMsgPub(t)
	msg2 := *msg1
	pendingMsg := makeNewPendingMsg(t, msg1)

	// Store messages.
	delayErr := nDB.StoreDelayed(pendingMsg)
	require.NoError(t, delayErr, fmt.Sprintf("failed to store delayed message: %v", delayErr))
	blackholeErr := nDB.StoreBlackhole(&msg2)
	require.NoError(t, blackholeErr, fmt.Sprintf("failed to store blackholed message: %v", blackholeErr))

	// Retrieve both messages and ensure they're equal to what was stored.
	res, loadErr := nDB.LoadAll(zap.NewNop())
	require.NoError(t, loadErr)
	require.Equal(t, 1, len(res.Delayed))
	require.Equal(t, 1, len(res.Blackholed))
	require.Equal(t, pendingMsg, res.Delayed[0])
	require.Equal(t, &msg2, res.Blackholed[0])
}

func TestKeysForStoredMessagesV1(t *testing.T) {
	msg1 := makeNewMsgPub(t)
	pMsg := makeNewPendingMsg(t, msg1)

	require.Equal(
		t,
		[]byte("NOTARY:DELAY:V1:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
		delayKey(pMsg),
	)

	require.Equal(
		t,
		[]byte("NOTARY:BLACKHOLE:V1:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
		blackholeKey(msg1),
	)
}

// nowSeconds is a helper function that returns time.Now() with the nanoseconds truncated.
// The nanoseconds are not important to us and are not serialized.
func nowSeconds() time.Time {
	return time.Unix(time.Now().Unix(), 0)
}

// makeNewMsgPub returns a MessagePublication that has a token transfer payload
// but otherwise has default values.
func makeNewMsgPub(t *testing.T) *common.MessagePublication {
	t.Helper()

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	validTxID, err := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")
	require.NoError(t, err)

	msg := &common.MessagePublication{
		TxID:            validTxID,
		Timestamp:       nowSeconds(),
		Nonce:           1,
		Sequence:        789101112131415,
		EmitterChain:    vaa.ChainIDEthereum,
		EmitterAddress:  ethereumTokenBridgeAddr,
		Unreliable:      false,
		IsReobservation: false,
		Payload:         []byte{0x01},
	}

	err = msg.SetVerificationState(common.Anomalous)
	require.NoError(t, err)

	return msg
}

// makeNewPendingMsg wraps a message publication and adds a release time to create a PendingMessage
func makeNewPendingMsg(t *testing.T, msg *common.MessagePublication) *common.PendingMessage {
	t.Helper()

	return &common.PendingMessage{
		// The nanoseconds are not important to us and are not serialized.
		ReleaseTime: nowSeconds().Add(24 * time.Hour),
		Msg:         *msg,
	}
}
