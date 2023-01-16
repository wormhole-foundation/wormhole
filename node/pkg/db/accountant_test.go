package db

import (
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/dgraph-io/badger/v3"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestAcctPendingTransferMsgID(t *testing.T) {
	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	assert.Equal(t, []byte("ACCT:PXFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), acctPendingTransferMsgID(msg1.MessageIDString()))
}

func TestAcctIsPendingTransfer(t *testing.T) {
	assert.Equal(t, true, acctIsPendingTransfer([]byte("ACCT:PXFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER:")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER:1")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER:1/1/1")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("ACCT:PXFER:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, acctIsPendingTransfer([]byte("ACCT:PXFER:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, acctIsPendingTransfer([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, acctIsPendingTransfer([]byte{}))
}

func TestAcctStoreAndDeletePendingTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	msg2 := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
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
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
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
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	logger, _ := zap.NewDevelopment()

	pendingTransfers, err := db.AcctGetData(logger)
	require.NoError(t, err)
	assert.Equal(t, 0, len(pendingTransfers))
}

func TestAcctGetData(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	logger, _ := zap.NewDevelopment()

	// Store some unrelated junk in the db to make sure it gets skipped.
	junk := []byte("ABC123")
	err = db.db.Update(func(txn *badger.Txn) error {
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
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	msg2 := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
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

	pendingTransfers, err := db.AcctGetData(logger)
	require.NoError(t, err)
	require.Equal(t, 2, len(pendingTransfers))

	assert.Equal(t, msg1a, *pendingTransfers[0])
	assert.Equal(t, *msg2, *pendingTransfers[1])
}
