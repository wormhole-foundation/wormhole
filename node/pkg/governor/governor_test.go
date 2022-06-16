package governor

import (
	"context"
	"encoding/binary"
	// "encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func TestTrimEmptyTransfers(t *testing.T) {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	assert.Nil(t, err)

	var transfers []db.Transfer
	sum, updatedTransfers := TrimAndSumValue(transfers, now, nil)
	assert.Equal(t, uint64(0), sum)
	assert.Equal(t, 0, len(updatedTransfers))
}

func TestSumAllFromToday(t *testing.T) {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	assert.Nil(t, err)

	var transfers []db.Transfer
	transferTime, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 11:00am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 125000, Timestamp: transferTime})
	sum, updatedTransfers := TrimAndSumValue(transfers, now.Add(-time.Hour*24), nil)
	assert.Equal(t, uint64(125000), sum)
	assert.Equal(t, 1, len(updatedTransfers))
}

func TestTrimOneOfTwoTransfers(t *testing.T) {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	assert.Nil(t, err)

	var transfers []db.Transfer

	// The first transfer should be expired.
	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:59am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 125000, Timestamp: transferTime1})

	// But the second should not.
	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 1:00pm (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 225000, Timestamp: transferTime2})
	assert.Equal(t, 2, len(transfers))

	sum, updatedTransfers := TrimAndSumValue(transfers, now.Add(-time.Hour*24), nil)
	assert.Equal(t, 1, len(updatedTransfers))
	assert.Equal(t, uint64(225000), sum)
}

func TestTrimSeveralTransfers(t *testing.T) {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	assert.Nil(t, err)

	var transfers []db.Transfer

	// The first two transfers should be expired.
	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 10:00am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 125000, Timestamp: transferTime1})

	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:00am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 135000, Timestamp: transferTime2})

	// But the next three should not.
	transferTime3, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 1:00pm (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 145000, Timestamp: transferTime3})

	transferTime4, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 2:00pm (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 155000, Timestamp: transferTime4})

	transferTime5, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 2:00pm (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 165000, Timestamp: transferTime5})

	assert.Equal(t, 5, len(transfers))

	sum, updatedTransfers := TrimAndSumValue(transfers, now.Add(-time.Hour*24), nil)
	assert.Equal(t, 3, len(updatedTransfers))
	assert.Equal(t, uint64(465000), sum)
}

func TestTrimmingAllTransfersShouldReturnZero(t *testing.T) {
	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	assert.Nil(t, err)

	var transfers []db.Transfer

	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:00am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 125000, Timestamp: transferTime1})

	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:45am (CST)")
	assert.Nil(t, err)
	transfers = append(transfers, db.Transfer{Value: 225000, Timestamp: transferTime2})
	assert.Equal(t, 2, len(transfers))

	sum, updatedTransfers := TrimAndSumValue(transfers, now, nil)
	assert.Equal(t, 0, len(updatedTransfers))
	assert.Equal(t, uint64(0), sum)
}

func newChainGovernorForTest(ctx context.Context) (*ChainGovernor, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx is nil")
	}

	gov := NewChainGovernorForTest()

	err := gov.Run(ctx)
	if err != nil {
		return gov, nil
	}

	emitterAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	if err != nil {
		return gov, nil
	}

	tokenAddr, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	if err != nil {
		return gov, nil
	}

	gov.InitConfigForTest(
		vaa.ChainIDEthereum,
		emitterAddr,
		1000000,
		vaa.ChainIDEthereum,
		tokenAddr,
		"WETH",
		1774.62,
		8,
	)

	return gov, nil
}

func hashFromString(str string) eth_common.Hash {
	if (len(str) > 2) && (str[0] == '0') && (str[1] == 'x') {
		str = str[2:]
	}

	return eth_common.HexToHash(str)
}

func TestVaaForUninterestingEmitterChain(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	var payload = []byte{1, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payload,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
}

func TestVaaForUninterestingEmitterAddress(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	var payload = []byte{1, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payload,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
}

func TestVaaForUninterestingPayloadType(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	var payload = []byte{2, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payload,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
}

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

func TestBuidMockTransferPayload(t *testing.T) {
	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	payload, err := vaa.DecodeTransferPayloadHdr(payloadBytes)

	expectedTokenAddr, _ := vaa.StringToAddress(tokenAddrStr)
	expectedToAddr, _ := vaa.StringToAddress(toAddrStr)

	expectedAmount := big.NewInt(125000000)

	assert.Nil(t, err)
	assert.Equal(t, uint8(1), payload.Type)
	assert.Equal(t, vaa.ChainIDEthereum, payload.OriginChain)
	assert.Equal(t, expectedTokenAddr, payload.OriginAddress)
	assert.Equal(t, vaa.ChainIDPolygon, payload.TargetChain)
	assert.Equal(t, expectedToAddr, payload.TargetAddress)
	assert.Equal(t, 0, expectedAmount.Cmp(payload.Amount))
}

func TestVaaForUninterestingToken(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	uninterestingTokenAddrStr := "0x42"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		uninterestingTokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
}

func TestTransfersUpToAndOverTheLimit(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	gov.SetDayLengthInMinutes(24 * 60)
	gov.SetTokenPriceForTesting(vaa.ChainIDEthereum, tokenAddrStr, 1774.62)

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	// The first two transfers should be accepted.
	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(2218), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	canPost, err = gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(4436), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	// But the third one should be queued up.
	payloadBytes2 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1250,
	)

	msg.Payload = payloadBytes2

	canPost, err = gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(4436), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(2218274), valuePending)

	// Once we have something queued, small things should be queued as well.
	msg.Payload = payloadBytes1
	canPost, err = gov.ProcessMsgForTime(&msg, time.Now())
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(4436), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(2218274+2218), valuePending)
}

func TestPendingTransferBeingReleased(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	gov.SetDayLengthInMinutes(24 * 60)
	gov.SetTokenPriceForTesting(vaa.ChainIDEthereum, tokenAddrStr, 1774.62)

	// The first VAA should be accepted.
	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		270,
	)

	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(479147), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	// And so should the second.
	payloadBytes2 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		275,
	)

	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes2,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 6:00pm (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	// But the third one should be queued up.
	payloadBytes3 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		280,
	)

	msg3 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes3,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 2:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg3, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(496893), valuePending)

	// And so should the fourth one.
	payloadBytes4 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		300,
	)

	msg4 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes4,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 8:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg4, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(496893+532385), valuePending)

	// If we check pending before noon, nothing should happen.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 9:00am (CST)")
	toBePublished, err := gov.CheckPendingForTime(now, false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(496893+532385), valuePending)

	// But at 3pm, the first one should drop off and the first queued one should get posted.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 3:00pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Nil(t, err)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(488020+496893), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(532385), valuePending)
}

func TestIfAnythingIsQueuedEverythingIsQueued(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	assert.Nil(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	gov.SetDayLengthInMinutes(24 * 60)
	gov.SetTokenPriceForTesting(vaa.ChainIDEthereum, tokenAddrStr, 1774.62)

	// A bigg VAA should be queued.
	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1000,
	)

	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(1774619), valuePending)

	// And once we've started queuing things, even a small one should be queued.
	payloadBytes2 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1,
	)

	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes2,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 6:00pm (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	assert.Nil(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.GetStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(1774619+1774), valuePending)
}
