package common_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestPendingMessageQueue_Push(t *testing.T) {
	tests := []struct { // description of this test case
		name string
		msg  *common.PendingMessage
	}{
		{
			"single message",
			makeUniquePendingMessage(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			q := common.NewPendingMessageQueue()

			require.Equal(t, 0, q.Len())
			require.Nil(t, q.Peek())

			q.Push(tt.msg)

			require.Equal(t, 1, q.Len())
			// Ensure the first message is at the top of the queue
			require.Equal(t, tt.msg, q.Peek())
		})
	}
}

func TestPendingMessage_RoundTripMarshal(t *testing.T) {
	orig := makeUniquePendingMessage(t)
	var loaded common.PendingMessage

	bz, writeErr := orig.MarshalBinary()
	require.NoError(t, writeErr)

	readErr := loaded.UnmarshalBinary(bz)
	require.NoError(t, readErr)

	require.Equal(t, *orig, loaded)
}

func TestPendingMessage_MarshalError(t *testing.T) {

	type test struct {
		label  string
		input  common.MessagePublication
		errMsg string
	}

	// Set up.
	var (
		longTxID = bytes.NewBuffer(make([]byte, math.MaxUint8+1))
	)
	emitter, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	require.NoError(t, err)

	tests := []test{
		{
			label: "txID too long",
			input: common.MessagePublication{
				TxID: longTxID.Bytes(),
			},
			errMsg: "wrong size: TxID too long",
		},
		{
			label: "txID too short",
			input: common.MessagePublication{
				TxID:             []byte{},
				Timestamp:        time.Unix(int64(1654516425), 0),
				Nonce:            123456,
				Sequence:         789101112131415,
				EmitterChain:     vaa.ChainIDEthereum,
				EmitterAddress:   emitter,
				Payload:          []byte{},
				ConsistencyLevel: 32,
				Unreliable:       true,
				IsReobservation:  true,
			},
			errMsg: "wrong size: TxID too short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			pMsg := &common.PendingMessage{
				ReleaseTime: time.Now(),
				Msg:         tc.input,
			}

			bz, writeErr := pMsg.MarshalBinary()
			require.ErrorContains(t, writeErr, tc.errMsg)
			require.Nil(t, bz)
		})
	}

}

func TestPendingMessageQueue_NoDuplicates(t *testing.T) {
	q := common.NewPendingMessageQueue()

	// Create two messages with the same sequence number.
	msg1 := *makeUniquePendingMessage(t)
	msg2 := *makeUniquePendingMessage(t)
	msg2.Msg.Sequence = msg1.Msg.Sequence

	q.Push(&msg1)
	require.Equal(t, 1, q.Len())

	msg2.ReleaseTime = msg1.ReleaseTime.Add(time.Hour)
	require.True(t, msg1.ReleaseTime.Before(msg2.ReleaseTime))

	// Pushing two messages with the same Message ID should not add a duplicate.
	q.Push(&msg2)
	require.Equal(t, 1, q.Len())
}

func TestPendingMessage_HeapInvariants(t *testing.T) {

	msg1 := *makeUniquePendingMessage(t)
	msg2 := *makeUniquePendingMessage(t)
	msg3 := *makeUniquePendingMessage(t)

	// Modify release times, in ascending (past-to-future) order: msg1 < msg2 < msg3
	msg2.ReleaseTime = msg1.ReleaseTime.Add(time.Hour)
	msg3.ReleaseTime = msg1.ReleaseTime.Add(time.Hour * 2)

	require.True(t, msg1.ReleaseTime.Before(msg2.ReleaseTime))
	require.True(t, msg2.ReleaseTime.Before(msg3.ReleaseTime))

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		order []*common.PendingMessage
	}{
		{
			"ascending order",
			[]*common.PendingMessage{&msg1, &msg2, &msg3},
		},
		{
			"mixed order A",
			[]*common.PendingMessage{&msg2, &msg3, &msg1},
		},
		{
			"mixed order B",
			[]*common.PendingMessage{&msg3, &msg1, &msg2},
		},
	}

	// Try different variations of adding messages to the heap.
	// After each variation, the first element returned should be equal
	// to the smallest/oldest message publication, which is msg1.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			q := common.NewPendingMessageQueue()
			for pMsg := range slices.Values(tt.order) {
				q.Push(pMsg)
			}
			require.Equal(t, len(tt.order), q.Len())

			res := consumeHeapAndAssertOrdering(t, q)
			require.Equal(t, 0, q.Len())
			require.True(t, &msg1 == res[0])
			assertSliceOrdering(t, res)
		})
	}

	// Ensure that calling RemoveItem doesn't change the ordering.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			q := common.NewPendingMessageQueue()
			for pMsg := range slices.Values(tt.order) {
				q.Push(pMsg)
			}
			require.Equal(t, len(tt.order), q.Len())

			removed, err := q.RemoveItem(msg2.Msg.MessageID())
			require.NoError(t, err)
			require.NotNil(t, removed)
			require.Equal(t, &msg2, removed, "removed message does not match expected message")

			require.Equal(t, len(tt.order)-1, q.Len())

			res := consumeHeapAndAssertOrdering(t, q)
			require.Equal(t, 0, q.Len())
			require.True(t, &msg1 == res[0])
			assertSliceOrdering(t, res)
		})
	}

}

func TestPendingMessageQueue_Peek(t *testing.T) {
	q := common.NewPendingMessageQueue()

	msg1 := *makeUniquePendingMessage(t)
	msg2 := *makeUniquePendingMessage(t)
	msg3 := *makeUniquePendingMessage(t)

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

func TestPendingMessageQueue_RemoveItem(t *testing.T) {
	msgInQueue := makeUniquePendingMessage(t).Msg
	msgNotInQueue := makeUniquePendingMessage(t).Msg
	msgNotInQueue.TxID = []byte{0xff}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		target *common.MessagePublication
		want   *common.PendingMessage
	}{
		{
			"successful removal",
			&msgInQueue,
			&common.PendingMessage{Msg: msgInQueue},
		},
		{
			"remove an item that is not in the queue",
			&msgNotInQueue,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			q := common.NewPendingMessageQueue()

			q.Push(&common.PendingMessage{Msg: msgInQueue})

			got, gotErr := q.RemoveItem(tt.target.MessageID())
			require.NoError(t, gotErr)

			if tt.want != nil {
				require.NotNil(t, got)
				require.Equal(t, 0, q.Len())
				// The RemoveItem function relies on comparing TxIDs.
				require.Equal(t, tt.want.Msg.TxID, got.Msg.TxID)
			} else {
				require.Nil(t, got)
				require.Equal(t, 1, q.Len(), "item should not have been removed from queue")
			}

		})
	}
}

// TestPendingMessageQueue_DangerousOperations ensures that dangerous operations
// on the queue do not panic or cause unexpected behavior.
func TestPendingMessageQueue_DangerousOperations(t *testing.T) {
	q := common.NewPendingMessageQueue()

	// Popping an empty queue should not panic or alter the queue.
	element := q.Pop()
	require.Nil(t, element)
	require.Equal(t, 0, q.Len())

	// Peeking an empty queue should not panic or alter the queue.
	element = q.Peek()
	require.Nil(t, element)
	require.Equal(t, 0, q.Len())

	// Build some state for the next test.
	msg1 := *makeUniquePendingMessage(t)
	msg2 := *makeUniquePendingMessage(t)
	msg3 := *makeUniquePendingMessage(t)

	q.Push(&msg1)
	q.Push(&msg2)
	q.Push(&msg3)
	require.Equal(t, 3, q.Len())

	// Add nil to the queue and ensure that it is ignored.
	q.Push(nil)
	require.Equal(t, 3, q.Len())
}

func assertSliceOrdering(t *testing.T, s []*common.PendingMessage) {
	for i := range len(s) - 1 {
		require.True(t, s[i].ReleaseTime.Before(s[i+1].ReleaseTime))
	}
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

		if next == nil {
			continue
		}

		require.True(t, earliest.ReleaseTime.Before(q.Peek().ReleaseTime))
	}
	return res
}

func encodePayloadBytes(payload *vaa.TransferPayloadHdr) []byte {
	bz := make([]byte, 101)
	bz[0] = payload.Type

	amtBytes := payload.Amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}
	copy(bz[33-len(amtBytes):33], amtBytes)

	copy(bz[33:65], payload.OriginAddress.Bytes())
	binary.BigEndian.PutUint16(bz[65:67], uint16(payload.OriginChain))
	copy(bz[67:99], payload.TargetAddress.Bytes())
	binary.BigEndian.PutUint16(bz[99:101], uint16(payload.TargetChain))
	return bz
}

// Helper function that returns a valid PendingMessage. It creates identical messages publications
// with different sequence numbers.
func makeUniquePendingMessage(t *testing.T) *common.PendingMessage {
	t.Helper()

	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E") //nolint:gosec
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// Required as the Notary checks the emitter address.
	tokenBridge := sdk.KnownTokenbridgeEmitters[vaa.ChainIDEthereum]
	tokenBridgeAddress := vaa.Address(tokenBridge)
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

	// Should be unique for each test with high probability.
	// #nosec: G404 -- Cryptographically secure pseudo-random number generator not needed.
	var sequence = rand.Uint64()
	msgpub := &common.MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         sequence,
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
