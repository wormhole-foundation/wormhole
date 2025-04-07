package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
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

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	xfer2, err := UnmarshalTransfer(bytes)
	require.NoError(t, err)

	assert.Equal(t, xfer1, xfer2)

	expectedTransferKey := "GOV:XFER3:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestPendingMsgID(t *testing.T) {
	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg1 := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   ethereumTokenBridgeAddr,
		Payload:          []byte{},
		ConsistencyLevel: 16,
	}

	assert.Equal(t, []byte("GOV:PENDING4:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), PendingMsgID(msg1))
}

func TestTransferMsgID(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	assert.Equal(t, []byte("GOV:XFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), TransferMsgID(xfer))
}

func TestIsTransfer(t *testing.T) {
	assert.Equal(t, true, IsTransfer([]byte("GOV:XFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER3:")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER3:1")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER3:1/1/1")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:XFER3:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, IsTransfer([]byte("GOV:XFER3:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsTransfer([]byte{}))
	assert.Equal(t, true, isOldTransfer([]byte("GOV:XFER2:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, isOldTransfer([]byte("GOV:XFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))

}

func TestIsPendingMsg(t *testing.T) {
	assert.Equal(t, true, IsPendingMsg([]byte("GOV:PENDING4:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:XFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING4:")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING4:"+"1")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING4:"+"1/1/1")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING4:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/")))
	assert.Equal(t, true, IsPendingMsg([]byte("GOV:PENDING4:"+"1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:PENDING3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsPendingMsg([]byte{}))
	assert.Equal(t, true, isOldPendingMsg([]byte("GOV:PENDING3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, isOldPendingMsg([]byte("GOV:PENDING4:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
}

func TestGetChainGovernorData(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	logger := zap.NewNop()

	transfers, pending, err2 := db.GetChainGovernorData(logger)

	assert.Equal(t, []*Transfer(nil), transfers)
	assert.Equal(t, []*PendingTransfer(nil), pending)
	require.NoError(t, err2)
}

func TestStoreTransfer(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	err2 := db.StoreTransfer(xfer1)
	require.NoError(t, err2)
}

func TestDeleteTransfer(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
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
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	tokenBridgeAddr, err2 := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.NoError(t, err2)

	msg := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
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
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()

	tokenBridgeAddr, err2 := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.NoError(t, err2)

	msg := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
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
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ConsistencyLevel: 16,
		IsReobservation:  true,
	}

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516425+72*60*60), 0),
		Msg:         msg,
	}

	bytes, err := pending1.Marshal()
	require.NoError(t, err)

	pending2, err := UnmarshalPendingTransfer(bytes, false)
	require.NoError(t, err)

	assert.Equal(t, pending1, pending2)

	expectedPendingKey := "GOV:PENDING4:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedPendingKey, string(PendingMsgID(&pending2.Msg)))
}

func TestStoreAndReloadTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
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
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131416",
		Hash:           "Hash2",
	}

	err = db.StoreTransfer(xfer2)
	assert.Nil(t, err)

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516435+72*60*60), 0),
		Msg: common.MessagePublication{
			TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
			Timestamp:        time.Unix(int64(1654516435), 0),
			Nonce:            123456,
			Sequence:         789101112131417,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   ethereumTokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
		},
	}

	err = db.StorePendingMsg(pending1)
	assert.Nil(t, err)

	pending2 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516440+72*60*60), 0),
		Msg: common.MessagePublication{
			TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
			Timestamp:        time.Unix(int64(1654516440), 0),
			Nonce:            123456,
			Sequence:         789101112131418,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   ethereumTokenBridgeAddr,
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

func TestMarshalUnmarshalNoMsgIdOrHash(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		// Don't set MsgID or Hash, should handle empty slices.
	}

	bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	xfer2, err := UnmarshalTransfer(bytes)
	require.NoError(t, err)
	require.Equal(t, xfer1, xfer2)
}

// Note that Transfer.Marshal can't fail, so there are no negative tests for that.

func TestUnmarshalTransferFailures(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:           "Hash1",
	}

	bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	// First make sure regular unmarshal works.
	xfer2, err := UnmarshalTransfer(bytes)
	require.NoError(t, err)
	require.Equal(t, xfer1, xfer2)

	// Truncate the timestamp.
	_, err = UnmarshalTransfer(bytes[0 : 4-1])
	assert.ErrorContains(t, err, "failed to read timestamp: ")

	// Truncate the value.
	_, err = UnmarshalTransfer(bytes[0 : 4+8-1])
	assert.ErrorContains(t, err, "failed to read value: ")

	// Truncate the origin chain.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2-1])
	assert.ErrorContains(t, err, "failed to read origin chain id: ")

	// Truncate the origin address.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32-1])
	assert.ErrorContains(t, err, "failed to read origin address")

	// Truncate the emitter chain.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2-1])
	assert.ErrorContains(t, err, "failed to read emitter chain id: ")

	// Truncate the emitter address.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32-1])
	assert.ErrorContains(t, err, "failed to read emitter address")

	// Truncate the message ID length.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2-1])
	assert.ErrorContains(t, err, "failed to read msgID length: ")

	// Truncate the message ID data.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2+3])
	assert.ErrorContains(t, err, "failed to read msg id")

	// Truncate the hash length.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2+82+2-1])
	assert.ErrorContains(t, err, "failed to read hash length: ")

	// Truncate the hash data.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2+82+2+3])
	assert.ErrorContains(t, err, "failed to read hash")

	// Truncate the target chain.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2+82+2+5+2-1])
	assert.ErrorContains(t, err, "failed to read target chain id: ")

	// Truncate the target address.
	_, err = UnmarshalTransfer(bytes[0 : 4+8+2+32+2+32+2+82+2+5+2+32-1])
	assert.ErrorContains(t, err, "failed to read target address")
}

// Note that PendingTransfer.Marshal can't fail, so there are no negative tests for that.

func TestUnmarshalPendingTransferFailures(t *testing.T) {
	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	msg := common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ConsistencyLevel: 16,
		IsReobservation:  true,
	}

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516425+72*60*60), 0),
		Msg:         msg,
	}

	bytes, err := pending1.Marshal()
	require.NoError(t, err)

	// First make sure regular unmarshal works.
	pending2, err := UnmarshalPendingTransfer(bytes, false)
	require.NoError(t, err)
	assert.Equal(t, pending1, pending2)

	// Truncate the release time.
	_, err = UnmarshalPendingTransfer(bytes[0:4-1], false)
	assert.ErrorContains(t, err, "failed to read pending transfer release time: ")

	// The remainder is the marshaled message publication as a single buffer.

	// Truncate the entire serialized message.
	_, err = UnmarshalPendingTransfer(bytes[0:4], false)
	assert.ErrorContains(t, err, "failed to read pending transfer msg")

	// Truncate some of the serialized message.
	_, err = UnmarshalPendingTransfer(bytes[0:len(bytes)-10], false)
	assert.ErrorContains(t, err, "failed to unmarshal pending transfer msg")
}

func (d *Database) storeOldPendingMsg(t *testing.T, p *PendingTransfer) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(p.ReleaseTime.Unix())) // #nosec G115 -- This conversion is safe until year 2106

	b := marshalOldMessagePublication(&p.Msg)

	vaa.MustWrite(buf, binary.BigEndian, b)

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(oldPendingMsgID(&p.Msg), buf.Bytes()); err != nil {
			return err
		}
		return nil
	})

	require.NoError(t, err)
}

func marshalOldMessagePublication(msg *common.MessagePublication) []byte {
	buf := new(bytes.Buffer)

	buf.Write(msg.TxID[:])
	vaa.MustWrite(buf, binary.BigEndian, uint32(msg.Timestamp.Unix())) // #nosec G115 -- This conversion is safe until year 2106
	vaa.MustWrite(buf, binary.BigEndian, msg.Nonce)
	vaa.MustWrite(buf, binary.BigEndian, msg.Sequence)
	vaa.MustWrite(buf, binary.BigEndian, msg.ConsistencyLevel)
	vaa.MustWrite(buf, binary.BigEndian, msg.EmitterChain)
	buf.Write(msg.EmitterAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, msg.IsReobservation)
	buf.Write(msg.Payload)

	return buf.Bytes()
}

func TestLoadingOldPendingTransfers(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
	require.NoError(t, err)

	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	oldXfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		// Don't set TargetChain or TargetAddress.
		MsgID: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:  "Hash1",
	}

	err = db.storeOldTransfer(oldXfer1)
	require.NoError(t, err)

	newXfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516426), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131416",
		Hash:           "Hash1",
	}

	err = db.StoreTransfer(newXfer1)
	require.NoError(t, err)

	oldXfer2 := &Transfer{
		Timestamp:      time.Unix(int64(1654516427), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		// Don't set TargetChain or TargetAddress.
		MsgID: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131417",
		Hash:  "Hash2",
	}

	err = db.storeOldTransfer(oldXfer2)
	require.NoError(t, err)

	newXfer2 := &Transfer{
		Timestamp:      time.Unix(int64(1654516428), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131418",
		Hash:           "Hash2",
	}

	err = db.StoreTransfer(newXfer2)
	require.NoError(t, err)

	// Write the first pending event in the old format.
	now := time.Unix(time.Now().Unix(), 0)
	pending1 := &PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 71), // Setting it to 71 hours so we can confirm it didn't get set to the default.,
		Msg: common.MessagePublication{
			TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
			Timestamp:        now,
			Nonce:            123456,
			Sequence:         789101112131417,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   ethereumTokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
			// IsReobservation will not be serialized. It should be set to false on reload.
		},
	}

	db.storeOldPendingMsg(t, pending1)
	require.NoError(t, err)

	// Write the second one in the new format.
	now = now.Add(time.Second * 5)
	pending2 := &PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 71), // Setting it to 71 hours so we can confirm it didn't get set to the default.
		Msg: common.MessagePublication{
			TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
			Timestamp:        now,
			Nonce:            123456,
			Sequence:         789101112131418,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   ethereumTokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
			IsReobservation:  true,
		},
	}

	err = db.StorePendingMsg(pending2)
	require.NoError(t, err)

	// Write the third pending event in the old format.
	now = now.Add(time.Second * 5)
	pending3 := &PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 71), // Setting it to 71 hours so we can confirm it didn't get set to the default.,
		Msg: common.MessagePublication{
			TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064").Bytes(),
			Timestamp:        now,
			Nonce:            123456,
			Sequence:         789101112131419,
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   ethereumTokenBridgeAddr,
			Payload:          []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			ConsistencyLevel: 16,
			// IsReobservation will not be serialized. It should be set to false on reload.
		},
	}

	db.storeOldPendingMsg(t, pending3)
	require.NoError(t, err)

	logger, zapObserver := setupLogsCapture(t)

	xfers, pendings, err := db.GetChainGovernorDataForTime(logger, now)

	require.NoError(t, err)
	require.Equal(t, 4, len(xfers))
	require.Equal(t, 3, len(pendings))

	// Verify that we converted the two old pending transfers and the two old completed transfers.
	loggedEntries := zapObserver.FilterMessage("updating format of database entry for pending vaa").All()
	require.Equal(t, 2, len(loggedEntries))
	loggedEntries = zapObserver.FilterMessage("updating format of database entry for completed transfer").All()
	require.Equal(t, 2, len(loggedEntries))

	sort.SliceStable(xfers, func(i, j int) bool {
		return xfers[i].Timestamp.Before(xfers[j].Timestamp)
	})

	assert.Equal(t, oldXfer1, xfers[0])
	assert.Equal(t, newXfer1, xfers[1])
	assert.Equal(t, oldXfer2, xfers[2])
	assert.Equal(t, newXfer2, xfers[3])

	// Updated old pending events get placed at the end, so we need to sort into timestamp order.
	sort.SliceStable(pendings, func(i, j int) bool {
		return pendings[i].Msg.Timestamp.Before(pendings[j].Msg.Timestamp)
	})

	assert.Equal(t, pending1.Msg, pendings[0].Msg)
	assert.Equal(t, pending2.Msg, pendings[1].Msg)
	assert.Equal(t, pending3.Msg, pendings[2].Msg)

	// Make sure we can reload the updated pendings.

	logger, zapObserver = setupLogsCapture(t)

	xfers2, pendings2, err := db.GetChainGovernorDataForTime(logger, now)

	require.NoError(t, err)
	require.Equal(t, 4, len(xfers2))
	require.Equal(t, 3, len(pendings2))

	// This time we shouldn't have updated anything.
	loggedEntries = zapObserver.FilterMessage("updating format of database entry for pending vaa").All()
	require.Equal(t, 0, len(loggedEntries))
	loggedEntries = zapObserver.FilterMessage("updating format of database entry for completed transfer").All()
	require.Equal(t, 0, len(loggedEntries))

	sort.SliceStable(xfers2, func(i, j int) bool {
		return xfers2[i].Timestamp.Before(xfers2[j].Timestamp)
	})

	assert.Equal(t, oldXfer1, xfers2[0])
	assert.Equal(t, newXfer1, xfers2[1])
	assert.Equal(t, oldXfer2, xfers2[2])
	assert.Equal(t, newXfer2, xfers2[3])

	assert.Equal(t, pending1.Msg, pendings2[0].Msg)
	assert.Equal(t, pending2.Msg, pendings2[1].Msg)
}

func marshalOldTransfer(xfer *Transfer) ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(xfer.Timestamp.Unix())) // #nosec G115 -- This conversion is safe until year 2106
	vaa.MustWrite(buf, binary.BigEndian, xfer.Value)
	vaa.MustWrite(buf, binary.BigEndian, xfer.OriginChain)
	buf.Write(xfer.OriginAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, xfer.EmitterChain)
	buf.Write(xfer.EmitterAddress[:])
	if len(xfer.MsgID) > math.MaxUint16 {
		return nil, fmt.Errorf("failed to marshal MsgID, length too long: %d", len(xfer.MsgID))
	}
	vaa.MustWrite(buf, binary.BigEndian, uint16(len(xfer.MsgID))) // #nosec G115 -- This conversion is checked above
	if len(xfer.MsgID) > 0 {
		buf.Write([]byte(xfer.MsgID))
	}
	if len(xfer.Hash) > math.MaxUint16 {
		return nil, fmt.Errorf("failed to marshal Hash, length too long: %d", len(xfer.Hash))
	}
	vaa.MustWrite(buf, binary.BigEndian, uint16(len(xfer.Hash))) // #nosec G115 -- This conversion is checked above
	if len(xfer.Hash) > 0 {
		buf.Write([]byte(xfer.Hash))
	}
	return buf.Bytes(), nil
}

func (d *Database) storeOldTransfer(xfer *Transfer) error {
	key := []byte(fmt.Sprintf("%v%v", oldTransfer, xfer.MsgID))
	b, err := marshalOldTransfer(xfer)

	if err != nil {
		return err
	}

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

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: ethereumTokenBridgeAddr,
		// Don't set TargetChain or TargetAddress.
		MsgID: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
		Hash:  "Hash1",
	}

	bytes, err := marshalOldTransfer(xfer1)
	require.NoError(t, err)

	xfer2, err := unmarshalOldTransfer(bytes)
	require.NoError(t, err)

	assert.Equal(t, xfer1, xfer2)

	expectedTransferKey := "GOV:XFER3:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestOldTransfersUpdatedWhenReloading(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	ethereumTokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	bscTokenBridgeAddr, err := vaa.StringToAddress("0x26b4afb60d6c903165150c6f0aa14f8016be4aec")
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
		EmitterAddress: ethereumTokenBridgeAddr,
		// Don't set TargetChain or TargetAddress.
		MsgID: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
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
		EmitterAddress: ethereumTokenBridgeAddr,
		TargetChain:    vaa.ChainIDBSC,
		TargetAddress:  bscTokenBridgeAddr,
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
