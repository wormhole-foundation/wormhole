package db

import (
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func TestSerializeAndDeserializeOfTransfer(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	assert.Nil(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.Nil(t, err)

	xfer1 := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
	}

	bytes, err := xfer1.Marshal()
	assert.Nil(t, err)

	xfer2, err := UnmarshalTransfer(bytes)
	assert.Nil(t, err)

	assert.Equal(t, xfer1.Timestamp, xfer2.Timestamp)
	assert.Equal(t, xfer1.Value, xfer2.Value)
	assert.Equal(t, xfer1.OriginChain, xfer2.OriginChain)
	assert.Equal(t, xfer1.OriginAddress, xfer2.OriginAddress)
	assert.Equal(t, xfer1.EmitterChain, xfer2.EmitterChain)
	assert.Equal(t, xfer1.EmitterAddress, xfer2.EmitterAddress)
	assert.Equal(t, xfer1.MsgID, xfer2.MsgID)

	expectedTransferKey := "GOV:XFER:2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"
	assert.Equal(t, expectedTransferKey, string(TransferMsgID(xfer2)))
}

func TestPendingMsgID(t *testing.T) {
	tokenBridgeAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.Nil(t, err)

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

	assert.Equal(t, []byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), PendingMsgID(msg1))
}

func TestTransferMsgID(t *testing.T) {
	tokenAddr, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	assert.Nil(t, err)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	assert.Nil(t, err)

	xfer := &Transfer{
		Timestamp:      time.Unix(int64(1654516425), 0),
		Value:          125000,
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddr,
		MsgID:          "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415",
	}

	assert.Equal(t, []byte("GOV:XFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), TransferMsgID(xfer))
}

func TestIsTransfer(t *testing.T) {
	assert.Equal(t, true, IsTransfer([]byte("GOV:XFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsTransfer([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsTransfer([]byte{}))
}

func TestIsPendingMsg(t *testing.T) {
	assert.Equal(t, true, IsPendingMsg([]byte("GOV:PENDING:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte("GOV:XFER:"+"2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415")))
	assert.Equal(t, false, IsPendingMsg([]byte{0x01, 0x02, 0x03, 0x04}))
	assert.Equal(t, false, IsPendingMsg([]byte{}))
}
