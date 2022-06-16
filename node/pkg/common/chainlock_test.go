package common

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// Note this method assumes 18 decimals for the amount.
func buildMockTransferPayloadBytes(
	t uint8,
	tokenChainID vaa.ChainID,
	tokenAddrStr string,
	toChainID vaa.ChainID,
	toAddrStr string,
	amtFloat float64,
) []byte {
	bytes := make([]byte, 101)
	bytes[0] = t

	amtBigFloat := big.NewFloat(amtFloat)
	amtBigFloat = amtBigFloat.Mul(amtBigFloat, big.NewFloat(100000000))
	amount, _ := amtBigFloat.Int(nil)
	amtBytes := amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}
	copy(bytes[33-len(amtBytes):33], amtBytes)

	tokenAddr, _ := vaa.StringToAddress(tokenAddrStr)
	copy(bytes[33:65], tokenAddr.Bytes())
	binary.BigEndian.PutUint16(bytes[65:67], uint16(tokenChainID))
	toAddr, _ := vaa.StringToAddress(toAddrStr)
	copy(bytes[67:99], toAddr.Bytes())
	binary.BigEndian.PutUint16(bytes[99:101], uint16(toChainID))
	// fmt.Printf("Bytes: [%v]", hex.EncodeToString(bytes))
	return bytes
}

func TestSerializeAndDeserializeOfMessagePublication(t *testing.T) {
	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		270,
	)

	msg1 := &MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          payloadBytes1,
		ConsistencyLevel: 16,
	}

	bytes, err := msg1.Marshal()
	assert.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bytes)
	assert.NoError(t, err)

	assert.Equal(t, msg1.TxHash, msg2.TxHash)
	assert.Equal(t, msg1.Timestamp, msg2.Timestamp)
	assert.Equal(t, msg1.Nonce, msg2.Nonce)
	assert.Equal(t, msg1.Sequence, msg2.Sequence)
	assert.Equal(t, msg1.EmitterChain, msg2.EmitterChain)
	assert.Equal(t, msg1.EmitterAddress, msg2.EmitterAddress)
	assert.Equal(t, msg1.ConsistencyLevel, msg2.ConsistencyLevel)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	assert.NoError(t, err)

	expectTokenAddr, err := vaa.StringToAddress(tokenAddrStr)
	assert.NoError(t, err)

	expectToAddr, err := vaa.StringToAddress(toAddrStr)
	assert.NoError(t, err)

	assert.Equal(t, uint8(1), payload2.Type)
	assert.Equal(t, vaa.ChainIDEthereum, payload2.OriginChain)
	assert.Equal(t, expectTokenAddr, payload2.OriginAddress)
	assert.Equal(t, vaa.ChainIDPolygon, payload2.TargetChain)
	assert.Equal(t, expectToAddr, payload2.TargetAddress)
	assert.Equal(t, 0, big.NewInt(27000000000).Cmp(payload2.Amount))
}

func TestMessageIDString(t *testing.T) {
	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		270,
	)

	msg := &MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          payloadBytes1,
		ConsistencyLevel: 16,
	}

	assert.Equal(t, "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415", msg.MessageIDString())
}

func TestMessageID(t *testing.T) {
	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		270,
	)

	msg := &MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		Payload:          payloadBytes1,
		ConsistencyLevel: 16,
	}

	assert.Equal(t, []byte("2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/789101112131415"), msg.MessageID())
}
