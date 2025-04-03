package common_test

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func encodePayloadBytes(payload *vaa.TransferPayloadHdr) []byte {
	bytes := make([]byte, 101)
	bytes[0] = payload.Type

	amtBytes := payload.Amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}
	copy(bytes[33-len(amtBytes):33], amtBytes)

	copy(bytes[33:65], payload.OriginAddress.Bytes())
	binary.BigEndian.PutUint16(bytes[65:67], uint16(payload.OriginChain))
	copy(bytes[67:99], payload.TargetAddress.Bytes())
	binary.BigEndian.PutUint16(bytes[99:101], uint16(payload.TargetChain))
	return bytes
}

// helper function that returns a valid PendingMessage.
func makeTestPendingMessage(t *testing.T) *common.PendingMessage {
	t.Helper()

	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E") //nolint:gosec
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes := encodePayloadBytes(payload)

	msgpub := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes,
		ConsistencyLevel: 32,
		Unreliable:       true,
		IsReobservation:  true,
	}
	setErr := msgpub.SetVerificationState(common.Anomalous)
	require.NoError(t, setErr)

	// The nanoseconds are not important to us and are not serialized.
	releaseTime := time.Unix(int64(1654516425), 0)
	return &common.PendingMessage{
		ReleaseTime: releaseTime,
		Msg:         *msgpub,
	}
}

func TestPendingMessage_RoundTripMarshal(t *testing.T) {
	orig := makeTestPendingMessage(t)
	var loaded common.PendingMessage

	bytes, writeErr := orig.MarshalBinary()
	require.NoError(t, writeErr)

	readErr := loaded.UnmarshalBinary(bytes)
	require.NoError(t, readErr)

	require.Equal(t, *orig, loaded)
}

// consumeHeapAndAssertOrdering takes heap and pops every element, ensuring that
// each popped element is smaller than the next one on the heap.
// Returns the elements in order of when they were popped. This should result
// in a slice of strictly ascending values.
func consumeHeapAndAssertOrdering(t *testing.T, q *common.PendingMessageQueue) []*common.PendingMessage {
	require.True(t, q.Len() > 0, "programming error: can't process empty queue")

	res := make([]*common.PendingMessage, 0, q.Len())

	// Pop all entries from the heap. Ensure that the element on top of the heap
	// is always the earliest (smallest timestamp).
	for q.Len() > 0 {
		// length changes automatically after popping.
		earliest := q.Pop()
		res = append(res, earliest)

		next := q.Peek()

		// Expect next to not be nil unless we just popped the last element.
		if q.Len() > 0 {
			require.NotNil(t, next)
		}

		if next != nil {
			require.True(t, earliest.ReleaseTime.Before(q.Peek().ReleaseTime))
		}
	}
	return res
}

func assertSliceOrdering(t *testing.T, s []*common.PendingMessage) {
	for i := range len(s) - 1 {
		require.True(t, s[i].ReleaseTime.Before(s[i+1].ReleaseTime))
	}
}

func TestPendingMessage_HeapInvariants(t *testing.T) {
	q := common.NewPendingMessageQueue()

	msg1 := *makeTestPendingMessage(t)
	msg2 := msg1
	msg3 := msg1

	// Modify release times, in ascending (past-to-future) order: msg1 < msg2 < msg3
	msg2.ReleaseTime = msg1.ReleaseTime.Add(time.Hour)
	msg3.ReleaseTime = msg1.ReleaseTime.Add(time.Hour * 2)

	require.True(t, msg1.ReleaseTime.Before(msg2.ReleaseTime))
	require.True(t, msg2.ReleaseTime.Before(msg3.ReleaseTime))

	// Try different variations of adding messages to the heap.
	// After each variation, the first element returned should be equal
	// to the smallest/oldest message publication, which is msg1.
	q.Push(&msg1)
	q.Push(&msg2)
	q.Push(&msg3)
	require.Equal(t, 3, q.Len())
	res := consumeHeapAndAssertOrdering(t, q)
	require.Equal(t, 0, q.Len())
	require.True(t, &msg1 == res[0])
	assertSliceOrdering(t, res)

	q.Push(&msg3)
	q.Push(&msg1)
	q.Push(&msg2)
	require.Equal(t, 3, q.Len())
	res = consumeHeapAndAssertOrdering(t, q)
	require.Equal(t, 0, q.Len())
	require.True(t, &msg1 == res[0])
	assertSliceOrdering(t, res)

	q.Push(&msg2)
	q.Push(&msg3)
	q.Push(&msg1)
	require.Equal(t, 3, q.Len())
	res = consumeHeapAndAssertOrdering(t, q)
	require.Equal(t, 0, q.Len())
	require.True(t, &msg1 == res[0])
	assertSliceOrdering(t, res)
}

func TestPendingMessageQueue_Peek(t *testing.T) {
	q := common.NewPendingMessageQueue()

	msg1 := *makeTestPendingMessage(t)
	msg2 := msg1
	msg3 := msg1

	// Modify release times, in ascending (past-to-future) order: msg1 < msg2 < msg3
	msg2.ReleaseTime = msg1.ReleaseTime.Add(time.Hour)
	msg3.ReleaseTime = msg1.ReleaseTime.Add(time.Hour * 2)

	require.True(t, msg1.ReleaseTime.Before(msg2.ReleaseTime))
	require.True(t, msg2.ReleaseTime.Before(msg3.ReleaseTime))

	// Push elements in an arbitrary order.
	// Assert that Peek() returns msg1 because it is the smallest.
	q.Push(&msg2)
	q.Push(&msg3)
	q.Push(&msg1)
	require.Equal(t, 3, q.Len())
	require.Equal(t, &msg1, q.Peek())

}
