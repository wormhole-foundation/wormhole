package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/dgraph-io/badger/v3"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func (d *Database) rowExistsInDB(key []byte) error {
	return d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
}

func TestSerializeAndDeserializeOfTransfer(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	xfer2, err := UnmarshalTransfer(bytes)
	require.NoError(t, err)

	assert.Equal(t, xfer1, xfer2)

	expectedTransferKey := "GOV:XFER2:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestPendingMsgID(t *testing.T) {
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

	assert.Equal(t, []byte("GOV:PENDING2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), PendingMsgID(msg1))
}

func TestTransferMsgID(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	assert.Equal(t, []byte("GOV:XFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), TransferMsgID(xfer))
}

func TestIsTransfer(t *testing.T) {
	assert.Equal(t, true, IsTransfer([]byte("GOV:XFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER2:")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER2:1")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER2:1/1/1")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER2:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, IsTransfer([]byte("GOV:XFER2:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsTransfer([]byte{}))
	assert.Equal(t, true, isOldTransfer([]byte("GOV:XFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, isOldTransfer([]byte("GOV:XFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))

}

func TestIsPendingMsg(t *testing.T) {
	assert.Equal(t, true, IsPendingMsg([]byte("GOV:PENDING2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:XFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING2:")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING2:"+"1")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING2:"+"1/1/1")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING2:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, IsPendingMsg([]byte("GOV:PENDING2:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsPendingMsg([]byte{}))
	assert.Equal(t, true, isOldPendingMsg([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, isOldPendingMsg([]byte("GOV:PENDING2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
}

func TestGetChainGovernorData(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	logger, _ := zap.NewDevelopment()

	transfers, pending, err2 := db.GetChainGovernorData(logger)

	assert.Equal(t, []*Transfer(nil), transfers)
	assert.Equal(t, []*PendingTransfer(nil), pending)
	require.NoError(t, err2)
}

func TestStoreTransfer(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	err2 := db.StoreTransfer(xfer1)
	require.NoError(t, err2)
}

func TestDeleteTransfer(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	err2 := db.StoreTransfer(xfer1)
	require.NoError(t, err2)

	// Make sure the xfer exists in the db.
	assert.NoError(t, db.rowExistsInDB(TransferMsgID(xfer1)))

	err3 := db.DeleteTransfer(xfer1)
	require.NoError(t, err3)

	// Make sure the xfer is no longer in the db.
	assert.ErrorIs(t, badger.ErrKeyNotFound, db.rowExistsInDB(TransferMsgID(xfer1)))
}

func TestStorePendingMsg(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	tokenBridgeAddr, err2 := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.NoError(t, err2)

	msg := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	pending := &PendingTransfer{ReleaseTime: msg.Timestamp.Add(time.Hour * 72), Msg: *msg}

	err3 := db.StorePendingMsg(pending)
	require.NoError(t, err3)
}

func TestDeletePendingMsg(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()

	tokenBridgeAddr, err2 := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.NoError(t, err2)

	msg := &common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	pending := &PendingTransfer{ReleaseTime: msg.Timestamp.Add(time.Hour * 72), Msg: *msg}

	err3 := db.StorePendingMsg(pending)
	require.NoError(t, err3)

	// Make sure the pending transfer exists in the db.
	assert.NoError(t, db.rowExistsInDB(PendingMsgID(msg)))

	err4 := db.DeletePendingMsg(pending)
	assert.Nil(t, err4)

	// Make sure the pending transfer is no longer in the db.
	assert.ErrorIs(t, badger.ErrKeyNotFound, db.rowExistsInDB(PendingMsgID(msg)))
}

func TestSerializeAndDeserializeOfPendingTransfer(t *testing.T) {
	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg := common.MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ConsistencyLevel: 16,
	}

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516425+72*60*60), 0),
		Msg:         msg,
	}

	bytes, err := pending1.Marshal()
	require.NoError(t, err)

	pending2, err := UnmarshalPendingTransfer(bytes)
	require.NoError(t, err)

	assert.Equal(t, pending1, pending2)

	expectedPendingKey := "GOV:PENDING2:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedPendingKey, string(PendingMsgID(&pending2.Msg)))
}

func TestStoreAndReloadTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	err = db.StoreTransfer(xfer1)
	assert.Nil(t, err)

	xfer2 := &Transfer{
		Timestamp:      time.Unix(int64(1654516430), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131416",
		Hash:           "Hash2",
	}

	err = db.StoreTransfer(xfer2)
	assert.Nil(t, err)

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516435+72*60*60), 0),
		Msg: common.MessagePublication{
			TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        time.Unix(int64(1654516435), 0),
			Nonce:            123456,
			Sequence:         789101112131417,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   tokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
		},
	}

	err = db.StorePendingMsg(pending1)
	assert.Nil(t, err)

	pending2 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516440+72*60*60), 0),
		Msg: common.MessagePublication{
			TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        time.Unix(int64(1654516440), 0),
			Nonce:            123456,
			Sequence:         789101112131418,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   tokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
		},
	}

	err = db.StorePendingMsg(pending2)
	assert.Nil(t, err)

	logger := zap.NewNop()
	xfers, pending, err := db.GetChainGovernorData(logger)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(xfers))
	assert.Equal(t, 2, len(pending))

	assert.Equal(t, xfer1, xfers[0])
	assert.Equal(t, xfer2, xfers[1])
	assert.Equal(t, pending1, pending[0])
	assert.Equal(t, pending2, pending[1])
}

func (d *Database) storeOldPendingMsg(t *testing.T, k *common.MessagePublication) {
	b, _ := k.Marshal()

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(oldPendingMsgID(k), b); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
}

func TestLoadingOldPendingTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	err = db.StoreTransfer(xfer1)
	require.Nil(t, err)

	xfer2 := &Transfer{
		Timestamp:      time.Unix(int64(1654516430), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131416",
		Hash:           "Hash2",
	}

	err = db.StoreTransfer(xfer2)
	require.Nil(t, err)

	now := time.Unix(time.Now().Unix(), 0)

	// Write the first pending event in the old format.
	pending1 := &PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 72), // Since we are writing this in the old format, this will not get stored, but computed on reload.
		Msg: common.MessagePublication{
			TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        now,
			Nonce:            123456,
			Sequence:         789101112131417,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   tokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
		},
	}

	db.storeOldPendingMsg(t, &pending1.Msg)
	require.Nil(t, err)

	now2 := now.Add(time.Second * 5)

	// Write the second one in the new format.
	pending2 := &PendingTransfer{
		ReleaseTime: now2.Add(time.Hour * 71), // Setting it to 71 hours so we can confirm it didn't get set to the default.
		Msg: common.MessagePublication{
			TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        now2,
			Nonce:            123456,
			Sequence:         789101112131418,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   tokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
		},
	}

	err = db.StorePendingMsg(pending2)
	require.Nil(t, err)

	logger := zap.NewNop()
	xfers, pendings, err := db.GetChainGovernorDataForTime(logger, now)

	require.Nil(t, err)
	require.Equal(t, 2, len(xfers))
	require.Equal(t, 2, len(pendings))

	// Updated old pending events get placed at the end, so we need to sort into timestamp order.
	sort.SliceStable(pendings, func(i, j int) bool {
		return pendings[i].Msg.Timestamp.Before(pendings[j].Msg.Timestamp)
	})

	assert.Equal(t, xfer1, xfers[0])
	assert.Equal(t, xfer2, xfers[1])
	assert.Equal(t, pending1.Msg, pendings[0].Msg)
	assert.Equal(t, pending2.Msg, pendings[1].Msg)

	// Make sure we can reload the updated pendings.

	xfers2, pendings2, err := db.GetChainGovernorDataForTime(logger, now)

	require.Nil(t, err)
	require.Equal(t, 2, len(xfers2))
	require.Equal(t, 2, len(pendings2))

	assert.Equal(t, xfer1, xfers2[0])
	assert.Equal(t, xfer2, xfers2[1])
	assert.Equal(t, pending1.Msg, pendings2[0].Msg)
	assert.Equal(t, pending2.Msg, pendings2[1].Msg)
}

func marshalOldTransfer(xfer *Transfer) []byte {
	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, uint32(xfer.Timestamp.Unix()))
	vaa.MustWrite(buf, binary.BigEndian, xfer.Value)
	vaa.MustWrite(buf, binary.BigEndian, xfer.OriginChain)
	buf.Write(xfer.OriginAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, xfer.EmitterChain)
	buf.Write(xfer.EmitterAddress[:])
	buf.Write([]byte(xfer.MsgID))
	return buf.Bytes()
}

func (d *Database) storeOldTransfer(xfer *Transfer) error {
	key := []byte(fmt.Sprintf("%v%v", oldTransfer, xfer.MsgID))
	b := marshalOldTransfer(xfer)

	return d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(key, b); err != nil {
			return err
		}
		return nil
	})
}

func TestDeserializeOfOldTransfer(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		// Do not set the Hash.
	}

	bytes := marshalOldTransfer(xfer1)

	xfer2, err := unmarshalOldTransfer(bytes)
	require.NoError(t, err)

	assert.Equal(t, xfer1, xfer2)

	expectedTransferKey := "GOV:XFER2:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestOldTransfersUpdatedWhenReloading(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// Write the first transfer in the old format.
	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		// Do not set the Hash.
	}

	err = db.storeOldTransfer(xfer1)
	require.NoError(t, err)

	// Write the second one in the new format.
	xfer2 := &Transfer{
		Timestamp:      time.Unix(int64(1654516430), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131416",
		Hash:           "Hash2",
	}

	err = db.StoreTransfer(xfer2)
	require.NoError(t, err)

	now := time.Unix(time.Now().Unix(), 0)

	logger := zap.NewNop()
	xfers, pendings, err := db.GetChainGovernorDataForTime(logger, now)

	require.NoError(t, err)
	require.Equal(t, 2, len(xfers))
	require.Equal(t, 0, len(pendings))

	// Updated old pending events get placed at the end, so we need to sort into timestamp order.
	sort.SliceStable(xfers, func(i, j int) bool {
		return xfers[i].Timestamp.Before(xfers[j].Timestamp)
	})

	assert.Equal(t, xfer1, xfers[0])
	assert.Equal(t, xfer2, xfers[1])

	// Make sure the old transfer got dropped from the database and rewritten in the new format.
	assert.ErrorIs(t, badger.ErrKeyNotFound, db.rowExistsInDB(oldTransferMsgID(xfer1)))
	assert.NoError(t, db.rowExistsInDB(TransferMsgID(xfer1)))

	// And make sure the other transfer is still there.
	assert.NoError(t, db.rowExistsInDB(TransferMsgID(xfer2)))

	// Make sure we can still read the database after the conversion.
	xfers, pendings, err = db.GetChainGovernorDataForTime(logger, now)

	require.NoError(t, err)
	require.Equal(t, 2, len(xfers))
	require.Equal(t, 0, len(pendings))

	// Updated old pending events get placed at the end, so we need to sort into timestamp order.
	sort.SliceStable(xfers, func(i, j int) bool {
		return xfers[i].Timestamp.Before(xfers[j].Timestamp)
	})

	assert.Equal(t, xfer1, xfers[0])
	assert.Equal(t, xfer2, xfers[1])
}
