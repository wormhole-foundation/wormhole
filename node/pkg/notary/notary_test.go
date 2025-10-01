package notary

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	eth_common "github.com/ethereum/go-ethereum/common"
)

// MockNotaryDB is a mock implementation of the NotaryDB interface.
// It returns nil for all operations, so it can be used to test the Notary's
// core logic but certain DB-related operations are not covered.
// Where possible, these should be tested in the Notary database's own unit tests, not here.
type MockNotaryDB struct{}

func (md MockNotaryDB) StoreBlackholed(m *common.MessagePublication) error { return nil }
func (md MockNotaryDB) StoreDelayed(p *common.PendingMessage) error        { return nil }
func (md MockNotaryDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	return nil, nil
}
func (md MockNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) { return nil, nil }
func (md MockNotaryDB) LoadAll(l *zap.Logger) (*db.NotaryLoadResult, error)        { return nil, nil }

func makeTestNotary(t *testing.T) *Notary {
	t.Helper()

	return &Notary{
		ctx:        context.Background(),
		logger:     zap.NewNop(),
		mutex:      sync.RWMutex{},
		database:   MockNotaryDB{},
		delayed:    &common.PendingMessageQueue{},
		blackholed: NewSet(),
		env:        common.GoTest,
	}
}

func TestNotary_ProcessMessageCorrectVerdict(t *testing.T) {

	// NOTE: This test should be exhaustive over VerificationState variants.
	tests := map[string]struct {
		verificationState common.VerificationState
		verdict           Verdict
	}{
		"approve N/A": {
			common.NotApplicable,
			Approve,
		},
		"approve not verified": {
			common.NotVerified,
			Approve,
		},
		"approve valid": {
			common.Valid,
			Approve,
		},
		"approve could not verify": {
			common.CouldNotVerify,
			Approve,
		},
		// Blackhole verdict is not being used for Rejected messages in the initial implementation
		"delay rejected": {
			common.Rejected,
			Delay,
		},
		"delay anomalous": {
			common.Anomalous,
			Delay,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			n := makeTestNotary(t)
			msg := makeUniqueMessagePublication(t)

			err := msg.SetVerificationState(test.verificationState)
			if test.verificationState != common.NotVerified {
				// SetVerificationState fails if the old status is equal to the new one.
				require.NoError(t, err)
			}

			require.True(t, vaa.IsTransfer(msg.Payload))

			verdict, err := n.ProcessMsg(msg)
			require.NoError(t, err)
			require.Equal(
				t,
				test.verdict,
				verdict,
				fmt.Sprintf("verificationState=%s verdict=%s", msg.VerificationState().String(), verdict.String()),
			)
		})
	}
}
func TestNotary_ProcessMsgUpdatesCollections(t *testing.T) {

	// NOTE: This test should be exhaustive over VerificationState variants.
	type expectedSizes struct {
		delayed    int
		blackholed int
	}
	tests := map[string]struct {
		verificationState common.VerificationState
		expectedSizes
	}{
		"Valid has no effect": {
			common.Valid,
			expectedSizes{},
		},
		"NotVerified has no effect": {
			common.NotVerified,
			expectedSizes{},
		},
		"NotApplicable has no effect": {
			common.NotApplicable,
			expectedSizes{},
		},
		"CouldNotVerify has no effect": {
			common.CouldNotVerify,
			expectedSizes{},
		},
		"Anomalous gets delayed": {
			common.Anomalous,
			expectedSizes{
				delayed:    1,
				blackholed: 0,
			},
		},

		// Blackhole verdict is not being used for Rejected messages in the initial implementation
		"Rejected gets delayed": {
			common.Rejected,
			expectedSizes{
				delayed:    1,
				blackholed: 0,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Set-up
			var (
				n   = makeTestNotary(t)
				msg = makeUniqueMessagePublication(t)
				err = msg.SetVerificationState(test.verificationState)
			)
			if test.verificationState != common.NotVerified {
				// SetVerificationState fails if the old status is equal to the new one.
				require.NoError(t, err)
			}
			require.Equal(t, test.verificationState, msg.VerificationState())
			require.True(t, vaa.IsTransfer(msg.Payload))

			// Ensure that the collections are properly updated.
			_, err = n.ProcessMsg(msg)
			require.NoError(t, err)
			require.Equal(
				t,
				test.expectedSizes.delayed,
				n.delayed.Len(),
				fmt.Sprintf("delayed count did not match. verificationState %s", msg.VerificationState().String()),
			)
			require.Equal(
				t,
				test.expectedSizes.blackholed,
				n.blackholed.Len(),
				fmt.Sprintf("blackholed count did not match. verificationState %s", msg.VerificationState().String()),
			)

		})
	}
}

func TestNotary_ProcessMessageAlwaysApprovesNonTokenTransfers(t *testing.T) {
	n := makeTestNotary(t)

	// NOTE: This test should be exhaustive over VerificationState variants.
	tests := map[string]struct {
		verificationState common.VerificationState
	}{
		"approve non-token transfer: NotVerified": {
			common.NotVerified,
		},
		"approve non-token transfer: CouldNotVerify": {
			common.CouldNotVerify,
		},
		"approve non-token transfer: Anomalous": {
			common.Anomalous,
		},
		"approve non-token transfer: Rejected": {
			common.Rejected,
		},
		"approve non-token transfer: NotApplicable": {
			common.NotApplicable,
		},
		"approve non-token transfer: Valid": {
			common.Valid,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			msg := makeUniqueMessagePublication(t)

			// Change the payload to something other than a token transfer.
			msg.Payload = []byte{0x02}
			require.False(t, vaa.IsTransfer(msg.Payload))

			if msg.VerificationState() != common.NotVerified {
				// SetVerificationState fails if the old status is equal to the new one.
				err := msg.SetVerificationState(test.verificationState)
				require.NoError(t, err)
			}

			verdict, err := n.ProcessMsg(msg)
			require.NoError(t, err)
			require.Equal(t, Approve, verdict)
		})
	}
}

func TestNotary_ProcessReadyMessages(t *testing.T) {

	tests := []struct {
		name               string                   // description of this test case
		delayed            []*common.PendingMessage // initial messages in delayed queue
		expectedDelayCount int
		expectedReadyCount int
	}{
		{
			"no messages ready",
			[]*common.PendingMessage{
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
			},
			1,
			0,
		},
		{
			"some messages ready",
			[]*common.PendingMessage{
				{
					ReleaseTime: time.Now().Add(-2 * time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
				{
					ReleaseTime: time.Now().Add(-time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
				{
					ReleaseTime: time.Now().Add(2 * time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
			},
			2,
			2,
		},
		{
			"all messages ready",
			[]*common.PendingMessage{
				{
					ReleaseTime: time.Now().Add(-2 * time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
				{
					ReleaseTime: time.Now().Add(-1 * time.Hour),
					Msg:         *makeUniqueMessagePublication(t),
				},
			},
			0,
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.delayed = common.NewPendingMessageQueue()

			currentLength := n.delayed.Len()
			for pMsg := range slices.Values(tt.delayed) {
				require.NotNil(t, pMsg)
				n.delayed.Push(pMsg)
				// Ensure that the queue grows after each push.
				require.Greater(t, n.delayed.Len(), currentLength)
				currentLength = n.delayed.Len()
			}
			require.Equal(t, len(tt.delayed), n.delayed.Len())

			readyMsgs := n.ReleaseReadyMessages()
			require.Equal(t, tt.expectedReadyCount, len(readyMsgs), "ready length does not match")
			require.Equal(t, tt.expectedDelayCount, n.delayed.Len(), "delayed length does not match")
		})
	}
}

func TestNotary_Forget(t *testing.T) {
	tests := []struct { // description of this test case
		name               string
		msg                *common.MessagePublication
		expectedDelayCount int
		expectedBlackholed int
	}{
		{
			"remove from delayed list",
			makeUniqueMessagePublication(t),
			0,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.delayed = common.NewPendingMessageQueue()
			n.blackholed = NewSet()

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 0, n.blackholed.Len())

			err := n.delay(tt.msg, time.Hour)
			require.NoError(t, err)

			require.Equal(t, 1, n.delayed.Len())
			require.Equal(t, 0, n.blackholed.Len())

			// Modify the set manually because calling the blackhole function will remove the message from the delayed list.
			n.blackholed.Add(tt.msg.MessageID())

			require.Equal(t, 1, n.delayed.Len())
			require.Equal(t, 1, n.blackholed.Len())

			forgetErr := n.forget(tt.msg)
			require.NoError(t, forgetErr)

			require.Equal(t, tt.expectedDelayCount, n.delayed.Len())
			require.Equal(t, tt.expectedBlackholed, n.blackholed.Len())
		})
	}
}

func TestNotary_BlackholeRemovesFromDelayedList(t *testing.T) {
	tests := []struct { // description of this test case
		name               string
		msg                *common.MessagePublication
		expectedDelayCount int
		expectedBlackholed int
	}{
		{
			"remove from delayed list",
			makeUniqueMessagePublication(t),
			0,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.delayed = common.NewPendingMessageQueue()
			n.blackholed = NewSet()

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 0, n.blackholed.Len())

			err := n.delay(tt.msg, time.Hour)
			require.NoError(t, err)

			require.Equal(t, 1, n.delayed.Len())
			require.Equal(t, 0, n.blackholed.Len())

			blackholeErr := n.blackhole(tt.msg)
			require.NoError(t, blackholeErr)

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 1, n.blackholed.Len())
		})
	}
}

func TestNotary_DelayFailsIfMessageAlreadyBlackholed(t *testing.T) {
	tests := []struct { // description of this test case
		name string
		msg  *common.MessagePublication
	}{
		{
			"delay fails if message is already blackholed",
			makeUniqueMessagePublication(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.delayed = common.NewPendingMessageQueue()
			n.blackholed = NewSet()

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 0, n.blackholed.Len())

			err := n.blackhole(tt.msg)
			require.NoError(t, err)

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 1, n.blackholed.Len())

			err = n.delay(tt.msg, time.Hour)
			require.ErrorIs(t, err, ErrAlreadyBlackholed)

			require.Equal(t, 0, n.delayed.Len())
			require.Equal(t, 1, n.blackholed.Len())
		})
	}
}

func TestNotary_releaseChangesReleaseTime(t *testing.T) {
	tests := []struct { // description of this test case
		name                string
		msg                 *common.MessagePublication
		expectedReleaseTime time.Time
	}{
		{
			"release changes release time",
			makeUniqueMessagePublication(t),
			time.Now().Add(time.Hour),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.delayed = common.NewPendingMessageQueue()
			n.blackholed = NewSet()

			require.Equal(t, 0, n.delayed.Len())

			// Delay a message; ensure no messages are ready
			delayErr := n.delay(tt.msg, time.Hour)
			require.NoError(t, delayErr)
			require.Equal(t, 1, n.delayed.Len())
			require.Empty(t, n.ReleaseReadyMessages())
			require.Equal(t, 1, n.delayed.Len())

			// Release the message
			releaseErr := n.release(tt.msg.MessageID())
			require.NoError(t, releaseErr)

			// Check that a new message is ready
			require.Len(t, n.ReleaseReadyMessages(), 1)
			require.Equal(t, 0, n.delayed.Len())
		})
	}
}

// Helper function that returns a valid PendingMessage. It creates identical messages publications
// with different sequence numbers.
func makeUniqueMessagePublication(t *testing.T) *common.MessagePublication {
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
		// verificationState is set to NotVerified by default.
	}

	return msgpub
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
