package notary

import (
	"fmt"
	"sync"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"golang.org/x/net/context"
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
