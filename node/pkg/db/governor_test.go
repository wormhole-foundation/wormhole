package db

import (
	"fmt"
	"os"
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

	xfer1Bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	xfer2, err := UnmarshalTransfer(xfer1Bytes)
	require.NoError(t, err)

	assert.Equal(t, xfer1, xfer2)

	expectedTransferKey := "GOV:XFER4:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestPendingMsgIDV5(t *testing.T) {
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

	assert.Equal(t, []byte("GOV:PENDING5:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), PendingMsgID(msg1))
}

func TestTransferMsgIDV4(t *testing.T) {
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

	assert.Equal(t, []byte("GOV:XFER4:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), TransferMsgID(xfer))
}

// Deprecated: This function does not unmarshal the Unreliable or verificationState fields.
func TestTransferMsgIDV3(t *testing.T) {
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

	assert.Equal(t, []byte("GOV:XFER3:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), oldTransferMsgID(xfer))
}

// TestIsTransferV4 tests the IsTransfer function for the current transfer format.
// The V4 suffix matches the "GOV:XFER4:" prefix used by the current transfer implementation.
func TestIsTransferV4(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid transfer message",
			input:    []byte("GOV:XFER4:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: true,
		},
		{
			name:     "previous message format",
			input:    []byte("GOV:XFER3:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "transfer prefix only",
			input:    []byte("GOV:XFER4:"),
			expected: false,
		},
		{
			name:     "transfer with single digit",
			input:    []byte("GOV:XFER4:1"),
			expected: false,
		},
		{
			name:     "transfer with a msgID that is too small",
			input:    []byte("GOV:XFER4:1/1/1"),
			expected: false,
		},
		{
			name:     "transfer with missing sequence",
			input:    []byte("GOV:XFER4:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/"),
			expected: false,
		},
		{
			name:     "valid transfer with sequence 0",
			input:    []byte("GOV:XFER4:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"),
			expected: true,
		},
		{
			name:     "pending message (not transfer)",
			input:    []byte("GOV:PENDING:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransfer(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsTransferV3 tests the isOldTransfer function for the legacy transfer format.
// The V3 suffix matches the "GOV:XFER3:" prefix used by the legacy transfer implementation.
func TestIsTransferV3(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "old transfer message",
			input:    []byte("GOV:XFER3:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: true,
		},
		{
			name:     "new transfer message",
			input:    []byte("GOV:XFER4:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "old transfer prefix only",
			input:    []byte("GOV:XFER3:"),
			expected: false,
		},
		{
			name:     "old transfer with single digit",
			input:    []byte("GOV:XFER3:1"),
			expected: false,
		},
		{
			name:     "old transfer with a msgID that is too small",
			input:    []byte("GOV:XFER3:1/1/1"),
			expected: false,
		},
		{
			name:     "old transfer with missing sequence",
			input:    []byte("GOV:XFER3:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/"),
			expected: false,
		},
		{
			name:     "old transfer with sequence 0",
			input:    []byte("GOV:XFER3:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"),
			expected: true,
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOldTransfer(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsPendingMsgV5 tests the IsPendingMsg function for the current pending message format.
// The V5 suffix matches the "GOV:PENDING5:" prefix used by the current pending message implementation.
func TestIsPendingMsgV5(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid pending message",
			input:    []byte("GOV:PENDING5:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: true,
		},
		{
			name:     "transfer message (not pending)",
			input:    []byte("GOV:XFER4:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "pending prefix only",
			input:    []byte("GOV:PENDING5:"),
			expected: false,
		},
		{
			name:     "pending with single digit",
			input:    []byte("GOV:PENDING5:" + "1"),
			expected: false,
		},
		{
			name:     "pending with a msgID that is too small",
			input:    []byte("GOV:PENDING5:" + "1/1/1"),
			expected: false,
		},
		{
			name:     "pending with missing sequence",
			input:    []byte("GOV:PENDING5:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/"),
			expected: false,
		},
		{
			name:     "valid pending with sequence 0",
			input:    []byte("GOV:PENDING5:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"),
			expected: true,
		},
		{
			name:     "old pending version",
			input:    []byte("GOV:PENDING4:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPendingMsg(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsPendingMsgV4 tests the isOldPendingMsg function for the legacy pending message format.
// The V4 suffix matches the "GOV:PENDING4:" prefix used by the legacy pending message implementation.
func TestIsPendingMsgV4(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "old pending message",
			input:    []byte("GOV:PENDING4:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: true,
		},
		{
			name:     "new pending message",
			input:    []byte("GOV:PENDING5:" + "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"),
			expected: false,
		},
		{
			name:     "old pending prefix only",
			input:    []byte("GOV:PENDING4:"),
			expected: false,
		},
		{
			name:     "old pending with single digit",
			input:    []byte("GOV:PENDING4:" + "1"),
			expected: false,
		},
		{
			name:     "old pending with a msgID that is too small",
			input:    []byte("GOV:PENDING4:" + "1/1/1"),
			expected: false,
		},
		{
			name:     "old pending with missing sequence",
			input:    []byte("GOV:PENDING4:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/"),
			expected: false,
		},
		{
			name:     "old pending with sequence 0",
			input:    []byte("GOV:PENDING4:" + "1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"),
			expected: true,
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOldPendingMsg(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
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
	vStateErr := msg.SetVerificationState(common.Valid)
	require.NoError(t, vStateErr)

	pending1 := &PendingTransfer{
		ReleaseTime: time.Unix(int64(1654516425+72*60*60), 0),
		Msg:         msg,
	}

	pending1Bytes, err := pending1.Marshal()
	require.NoError(t, err, fmt.Sprintf("Failed to marshal pending transfer: %v", err))

	pending2, err := UnmarshalPendingTransfer(pending1Bytes, false)
	require.NoError(t, err, fmt.Sprintf("Failed to unmarshal pending transfer: %v", err))

	assert.Equal(t, pending1, pending2)

	expectedPendingKey := "GOV:PENDING5:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedPendingKey, string(PendingMsgID(&pending2.Msg)))
}

func TestStoreAndReloadTransfersAndPendingMessages(t *testing.T) {
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

	xfer1Bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	xfer2, err := UnmarshalTransfer(xfer1Bytes)
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

	xfer1Bytes, err := xfer1.Marshal()
	require.NoError(t, err)

	// First make sure regular unmarshal works.
	xfer2, err := UnmarshalTransfer(xfer1Bytes)
	require.NoError(t, err)
	require.Equal(t, xfer1, xfer2)

	// Truncate the timestamp.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4-1])
	assert.ErrorContains(t, err, "failed to read timestamp: ")

	// Truncate the value.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8-1])
	assert.ErrorContains(t, err, "failed to read value: ")

	// Truncate the origin chain.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2-1])
	assert.ErrorContains(t, err, "failed to read origin chain id: ")

	// Truncate the origin address.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32-1])
	assert.ErrorContains(t, err, "failed to read origin address")

	// Truncate the emitter chain.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2-1])
	assert.ErrorContains(t, err, "failed to read emitter chain id: ")

	// Truncate the emitter address.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32-1])
	assert.ErrorContains(t, err, "failed to read emitter address")

	// Truncate the message ID length.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2-1])
	assert.ErrorContains(t, err, "failed to read msgID length: ")

	// Truncate the message ID data.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2+3])
	assert.ErrorContains(t, err, "failed to read msg id")

	// Truncate the hash length.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2+82+2-1])
	assert.ErrorContains(t, err, "failed to read hash length: ")

	// Truncate the hash data.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2+82+2+3])
	assert.ErrorContains(t, err, "failed to read hash")

	// Truncate the target chain.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2+82+2+5+2-1])
	assert.ErrorContains(t, err, "failed to read target chain id: ")

	// Truncate the target address.
	_, err = UnmarshalTransfer(xfer1Bytes[0 : 4+8+2+32+2+32+2+82+2+5+2+32-1])
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

	pending1Bytes, err := pending1.Marshal()
	require.NoError(t, err)

	// First make sure regular unmarshal works.
	pending2, err := UnmarshalPendingTransfer(pending1Bytes, false)
	require.NoError(t, err)
	assert.Equal(t, pending1, pending2)

	// Truncate the release time.
	_, err = UnmarshalPendingTransfer(pending1Bytes[0:4-1], false)
	assert.ErrorContains(t, err, "failed to read pending transfer release time: ")

	// The remainder is the marshaled message publication as a single buffer.

	// Truncate the entire serialized message.
	_, err = UnmarshalPendingTransfer(pending1Bytes[0:4], false)
	assert.ErrorContains(t, err, "failed to read pending transfer msg")

	// Truncate some of the serialized message.
	_, err = UnmarshalPendingTransfer(pending1Bytes[0:len(pending1Bytes)-10], false)
	assert.ErrorContains(t, err, "failed to unmarshal pending transfer msg")
}
