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
func (md MockNotaryDB) LoadAll() (*db.NotaryLoadResult, error)              { return nil, nil }

func makeTestNotary(t *testing.T) *Notary {
	t.Helper()

	return &Notary{
		ctx:        context.Background(),
		logger:     zap.NewNop(),
		mutex:      sync.Mutex{},
		database:   MockNotaryDB{},
		delayed:    &common.PendingMessageQueue{},
		ready:      []*common.MessagePublication{},
		blackholed: []*common.MessagePublication{},
		env:        common.GoTest,
	}
}

// makeNewMsgPub returns a MessagePublication that has a token transfer payload
// but otherwise has default values.
func makeNewMsgPub(t *testing.T) *common.MessagePublication {
	t.Helper()
	msg := &common.MessagePublication{
		// Required to mark this as a token transfer.
		Payload: []byte{0x01},
	}

	require.True(t, vaa.IsTransfer(msg.Payload))
	return msg
}

func TestNotary_ProcessMessage(t *testing.T) {
	n := makeTestNotary(t)

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
				fmt.Sprintf("verificationState=%s", msg.VerificationState().String()),
			)
		})
	}
}

func TestNotary_ProcessMessageAlwaysApprovesNonTokenTransfers(t *testing.T) {
	n := makeTestNotary(t)

	tests := map[string]struct {
		verificationState common.VerificationState
	}{
		"approve non-token transfer: NotVerified": {
			common.NotVerified,
		},
		"approve non-token transfer: Anomalous": {
			common.Anomalous,
		},
		"approve non-token transfer: Rejected": {
			common.Rejected,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			msg := makeNewMsgPub(t)
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
			n.ready = make([]*common.MessagePublication, 0, len(tt.delayed))
			n.delayed = common.NewPendingMessageQueue()

			for pMsg := range slices.Values(tt.delayed) {
				require.NotNil(t, pMsg)
				n.delayed.Push(pMsg)
			}
			require.Equal(t, len(tt.delayed), n.delayed.Len())
			require.Equal(t, int(0), len(n.ready))

			n.ProcessReadyMessages()
			require.Equal(t, tt.expectedReadyCount, len(n.ready), "ready length does not match")
			require.Equal(t, tt.expectedDelayCount, n.delayed.Len(), "delayed length does not match")
		})
	}
}

func TestNotary_Getters(t *testing.T) {
	var (
		msg1 = makeNewMsgPub(t)
		msg2 = *msg1
		msg3 = *msg1
	)
	// Make messages trivially different
	msg2.Sequence = 12345
	msg3.EmitterChain = vaa.ChainIDAvalanche

	tests := []struct {
		name       string                       // description of this test case
		delayed    []*common.PendingMessage     // initial messages in delayed queue
		ready      []*common.MessagePublication // initial messages in delayed queue
		blackholed []*common.MessagePublication // initial messages in delayed queue
	}{
		{
			"one of each",
			[]*common.PendingMessage{
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         *msg1,
				},
			},
			[]*common.MessagePublication{
				&msg2,
			},
			[]*common.MessagePublication{
				&msg3,
			},
		},
		{
			"two delayed, zero ready, one blackholed",
			[]*common.PendingMessage{
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         *msg1,
				},
				{
					ReleaseTime: time.Now().Add(time.Hour),
					Msg:         msg2,
				},
			},
			nil,
			[]*common.MessagePublication{
				&msg3,
			},
		},
		{
			"all zero",
			nil,
			nil,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.ready = tt.ready
			n.blackholed = tt.blackholed
			n.delayed = common.NewPendingMessageQueue()

			for pMsg := range slices.Values(tt.delayed) {
				require.NotNil(t, pMsg)
				n.delayed.Push(pMsg)
			}
			require.Equal(t, len(tt.delayed), n.delayed.Len())

			require.Equal(t, tt.delayed, n.Delayed(), "delayed getter does not match")
			require.Equal(t, tt.ready, n.Ready(), "ready getter does not match")
			require.Equal(t, tt.blackholed, n.Blackholed(), "blackhole getter does not match")
		})
	}
}
