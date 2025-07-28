package notary

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type MockNotaryDB struct{}

func (md MockNotaryDB) StoreBlackhole(m *common.MessagePublication) error   { return nil }
func (md MockNotaryDB) StoreDelayed(p *common.PendingMessage) error         { return nil }
func (md MockNotaryDB) DeleteBlackholed(m *common.MessagePublication) error { return nil }
func (md MockNotaryDB) DeleteDelayed(p *common.PendingMessage) error        { return nil }
func (md MockNotaryDB) LoadAll(l *zap.Logger) (*db.NotaryLoadResult, error) { return nil, nil }

func makeTestNotary(t *testing.T) *Notary {
	t.Helper()

	return &Notary{
		ctx:        context.Background(),
		logger:     zap.NewNop(),
		mutex:      sync.Mutex{},
		database:   MockNotaryDB{},
		delayed:    &common.PendingMessageQueue{},
		blackholed: NewSet(),
		env:        common.GoTest,
	}
}

// makeNewMsgPub returns a MessagePublication that has a token transfer payload
// but otherwise has default values.
func makeNewMsgPub(t *testing.T) *common.MessagePublication {
	t.Helper()
	msg := &common.MessagePublication{
		// Required to mark this as a token transfer.
		TxID:    []byte{0x01},
		Payload: []byte{0x01},
	}

	require.True(t, vaa.IsTransfer(msg.Payload))
	return msg
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
		"blackhole rejected": {
			common.Rejected,
			Blackhole,
		},
		"delay anomalous": {
			common.Anomalous,
			Delay,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			n := makeTestNotary(t)
			msg := makeNewMsgPub(t)

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
		"Rejected gets blackholed": {
			common.Rejected,
			expectedSizes{
				delayed:    0,
				blackholed: 1,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Set-up
			var (
				n   = makeTestNotary(t)
				msg = makeNewMsgPub(t)
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
			msg := makeNewMsgPub(t)

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

	msg := makeNewMsgPub(t)
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
					Msg:         *msg,
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
					Msg:         *msg,
				},
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         *msg,
				},
				{
					ReleaseTime: time.Now().Add(-time.Hour),
					Msg:         *msg,
				},
				{
					ReleaseTime: time.Now().Add(2 * time.Hour),
					Msg:         *msg,
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
					Msg:         *msg,
				},
				{
					ReleaseTime: time.Now().Add(-1 * time.Hour),
					Msg:         *msg,
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

			for pMsg := range slices.Values(tt.delayed) {
				require.NotNil(t, pMsg)
				n.delayed.Push(pMsg)
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
			makeNewMsgPub(t),
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
			n.blackholed.Add(tt.msg.VAAHash())

			require.Equal(t, 1, n.delayed.Len())
			require.Equal(t, 1, n.blackholed.Len())

			err = n.forget(tt.msg)
			require.NoError(t, err)

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
			makeNewMsgPub(t),
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

			err = n.blackhole(tt.msg)
			require.NoError(t, err)

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
			makeNewMsgPub(t),
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
