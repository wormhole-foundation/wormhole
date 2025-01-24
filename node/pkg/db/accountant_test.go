package db

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/dgraph-io/badger/v3"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestAcctPendingTransferMsgID(t *testing.T) {
	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	assert.Equal(t, []byte("ACCT:PXFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), acctOldPendingTransferMsgID(msg1.MessageIDString()))
	assert.Equal(t, []byte("ACCT:PXFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), acctPendingTransferMsgID(msg1.MessageIDString()))
}

func TestAcctIsPendingTransfer(t *testing.T) {
	assert.Equal(t, true, acctIsPendingTransfer([]byte("ACCT:PXFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER3:")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER3:1")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER3:1/1/1")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER3:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, acctIsPendingTransfer([]byte("ACCT:PXFER3:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, acctIsPendingTransfer([]byte{}))
	assert.Equal(t, true, acctIsOldPendingTransfer([]byte("ACCT:PXFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, acctIsOldPendingTransfer([]byte("ACCT:PXFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
}

func TestAcctStoreAndDeletePendingTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	msg2 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123457,
		Sequence:         789101112131416,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	err = db.AcctStorePendingTransfer(msg1)
	require.NoError(t, err)
	assert.NoError(t, db.rowExistsInDB(acctPendingTransferMsgID(msg1.MessageIDString())))

	err = db.AcctStorePendingTransfer(msg2)
	require.NoError(t, err)
	assert.NoError(t, db.rowExistsInDB(acctPendingTransferMsgID(msg2.MessageIDString())))

	err = db.AcctDeletePendingTransfer(msg1.MessageIDString())
	require.NoError(t, err)
	assert.Error(t, db.rowExistsInDB(acctPendingTransferMsgID(msg1.MessageIDString())))

	err = db.AcctDeletePendingTransfer(msg2.MessageIDString())
	require.NoError(t, err)
	assert.Error(t, db.rowExistsInDB(acctPendingTransferMsgID(msg2.MessageIDString())))

	// Delete something that doesn't exist.
	msg3 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123457,
		Sequence:         789101112131417,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	err = db.AcctDeletePendingTransfer(msg3.MessageIDString())
	require.NoError(t, err)
	assert.Error(t, db.rowExistsInDB(acctPendingTransferMsgID(msg3.MessageIDString())))
}

func TestAcctGetEmptyData(t *testing.T) {
	logger := zap.NewNop()
	dbPath := t.TempDir()
	db := OpenDb(logger, &dbPath)
	defer db.Close()

	pendings, err := db.AcctGetData(logger)
	require.NoError(t, err)
	assert.Equal(t, 0, len(pendings))
}

func TestAcctGetData(t *testing.T) {
	logger := zap.NewNop()
	dbPath := t.TempDir()
	db := OpenDb(logger, &dbPath)
	defer db.Close()

	// Store some unrelated junk in the db to make sure it gets skipped.
	junk := []byte("ABC123")
	err := db.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(junk, junk); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, db.rowExistsInDB(junk))

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	msg2 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123457,
		Sequence:         789101112131416,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	err = db.AcctStorePendingTransfer(msg1)
	require.NoError(t, err)
	require.NoError(t, db.rowExistsInDB(acctPendingTransferMsgID(msg1.MessageIDString())))

	err = db.AcctStorePendingTransfer(msg2)
	require.NoError(t, err)
	require.NoError(t, db.rowExistsInDB(acctPendingTransferMsgID(msg2.MessageIDString())))

	// Store the same transfer again with an update.
	msg1a := *msg1
	msg1a.ConsistencyLevel = 17
	err = db.AcctStorePendingTransfer(&msg1a)
	require.NoError(t, err)

	pendings, err := db.AcctGetData(logger)
	require.NoError(t, err)
	require.Equal(t, 2, len(pendings))

	assert.Equal(t, msg1a, *pendings[0])
	assert.Equal(t, *msg2, *pendings[1])
}

func TestAcctLoadingWhereOldPendingGetsUpdated(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	logger, zapObserver := setupLogsCapture(t)

	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	now := time.Unix(time.Now().Unix(), 0)

	// Write the first pending event in the old format.
	pending1 := &OldMessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        now,
		Nonce:            123456,
		Sequence:         789101112131417,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ConsistencyLevel: 16,
		// IsReobservation will not be serialized. It should be set to false on reload.
	}

	err = db.acctStoreOldPendingTransfer(pending1)
	require.Nil(t, err)

	now2 := now.Add(time.Second * 5)

	// Write the second one in the new format.
	pending2 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        now2,
		Nonce:            123456,
		Sequence:         789101112131418,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ConsistencyLevel: 16,
		IsReobservation:  true,
	}

	err = db.AcctStorePendingTransfer(pending2)
	require.Nil(t, err)

	// When we reload the data, the first one should get converted and returned here.
	pendings, err := db.AcctGetData(logger)
	require.NoError(t, err)
	require.Equal(t, 2, len(pendings))

	// Verify that we converted and deleted the old one.
	loggedEntries := zapObserver.FilterMessage("converting old pending transfer to new format").All()
	require.Equal(t, 1, len(loggedEntries))
	loggedEntries = zapObserver.FilterMessage("deleting old pending transfer").All()
	require.Equal(t, 1, len(loggedEntries))

	sort.SliceStable(pendings, func(i, j int) bool {
		return pendings[i].Timestamp.Before(pendings[j].Timestamp)
	})

	assert.Equal(t, *convertOldToNew(pending1), *pendings[0])
	assert.Equal(t, *pending2, *pendings[1])

	// Make sure we can still reload things after updating the old one.
	logger, zapObserver = setupLogsCapture(t)
	pendings2, err := db.AcctGetData(logger)

	require.Nil(t, err)
	require.Equal(t, 2, len(pendings2))

	// Verify that we didn't do any conversions the second time.
	loggedEntries = zapObserver.FilterMessage("converting old pending transfer to new format").All()
	require.Equal(t, 0, len(loggedEntries))
	loggedEntries = zapObserver.FilterMessage("deleting old pending transfer").All()
	require.Equal(t, 0, len(loggedEntries))

	assert.Equal(t, *convertOldToNew(pending1), *pendings[0])
	assert.Equal(t, *pending2, *pendings[1])

	sort.SliceStable(pendings, func(i, j int) bool {
		return pendings[i].Timestamp.Before(pendings[j].Timestamp)
	})

	assert.Equal(t, *convertOldToNew(pending1), *pendings[0])
	assert.Equal(t, *pending2, *pendings[1])
}

// setupLogsCapture is a helper function for making a zap logger/observer combination for testing that certain logs have been made
func setupLogsCapture(t testing.TB) (*zap.Logger, *observer.ObservedLogs) {
	t.Helper()
	observedCore, observedLogs := observer.New(zap.InfoLevel)
	consoleLogger := zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel))
	parentLogger := zap.New(zapcore.NewTee(observedCore, consoleLogger.Core()))
	return parentLogger, observedLogs
}

func (d *Database) acctStoreOldPendingTransfer(msg *OldMessagePublication) error {
	b, _ := json.Marshal(msg)

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(acctOldPendingTransferMsgID(msg.MessageIDString()), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit old accountant pending transfer for tx %s: %w", msg.MessageIDString(), err)
	}

	return nil
}

// The standard json Marshal / Unmarshal of time.Time gets confused between local and UTC time.
func (msg *OldMessagePublication) MarshalJSON() ([]byte, error) {
	type Alias OldMessagePublication
	return json.Marshal(&struct {
		Timestamp int64
		*Alias
	}{
		Timestamp: msg.Timestamp.Unix(),
		Alias:     (*Alias)(msg),
	})
}

func (msg *OldMessagePublication) MessageIDString() string {
	return fmt.Sprintf("%v/%v/%v", uint16(msg.EmitterChain), msg.EmitterAddress, msg.Sequence)
}

func TestUnmarshalOldJSON(t *testing.T) {
	jsn := `
	{
	  "TxID": "SGVsbG8=",
		"Timestamp": 1654516425,
		"TxHash": "0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063",
		"Nonce": 123456,
		"Sequence": 789101112131415,
		"ConsistencyLevel": 32,
		"EmitterChain": 2,
		"EmitterAddress": "000000000000000000000000707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZJU04AAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAHB/kRjjOpuJmL6kHdDUbzi7lj/IAAU=",
		"IsReobservation": false,
		"Unreliable": false
	}
	`

	var oldMsg OldMessagePublication
	err := json.Unmarshal([]byte(jsn), &oldMsg)
	require.NoError(t, err)

	newMsg := convertOldToNew(&oldMsg)
	assert.Equal(t, oldMsg.TxHash.String(), newMsg.TxIDString())
}
