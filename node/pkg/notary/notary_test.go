package notary

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

// configurableMockDB is a NotaryDBInterface mock whose LoadAll result and error
// are configurable, and whose store/delete errors can be injected. It is used to
// exercise database-dependent code paths that MockNotaryDB cannot reach.
type configurableMockDB struct {
	loadResult *db.NotaryLoadResult
	loadErr    error

	storeDelayedErr     error
	storeBlackholedErr  error
	deleteDelayedErr    error
	deleteBlackholedErr error

	deleteDelayedResult    *common.PendingMessage
	deleteBlackholedResult *common.MessagePublication

	deleteDelayedErrOnce error
	deleteDelayedCalls   int
}

func (m *configurableMockDB) StoreBlackholed(*common.MessagePublication) error {
	return m.storeBlackholedErr
}
func (m *configurableMockDB) StoreDelayed(*common.PendingMessage) error { return m.storeDelayedErr }
func (m *configurableMockDB) DeleteBlackholed([]byte) (*common.MessagePublication, error) {
	return m.deleteBlackholedResult, m.deleteBlackholedErr
}
func (m *configurableMockDB) DeleteDelayed([]byte) (*common.PendingMessage, error) {
	m.deleteDelayedCalls++
	if m.deleteDelayedErrOnce != nil && m.deleteDelayedCalls == 1 {
		return nil, m.deleteDelayedErrOnce
	}
	return m.deleteDelayedResult, m.deleteDelayedErr
}
func (m *configurableMockDB) LoadAll(*zap.Logger) (*db.NotaryLoadResult, error) {
	return m.loadResult, m.loadErr
}

func makeTestNotaryWithDB(t *testing.T, database db.NotaryDBInterface) *Notary {
	t.Helper()

	n := makeTestNotary(t)
	n.database = database
	return n
}

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

// TestNotary_AlwaysApproveNonTransferVerifierEmitters tests that all messages are approve if the emitter chain does not have a transfer verifier.
// This test can be removed if the Notary is extended to support other chains.
func TestNotary_AlwaysApproveNonTransferVerifierEmitters(t *testing.T) {
	// NOTE: Solana does not have a transfer verifier implementation
	tests := map[string]struct {
		verificationState common.VerificationState
		emitterChain      vaa.ChainID
		verdict           Verdict
	}{
		"approve non-transfer verifier when Rejected": {
			common.Rejected,
			vaa.ChainIDSolana,
			Approve,
		},
		"approve non-transfer verifier when Anomalous": {
			common.Anomalous,
			vaa.ChainIDSolana,
			Approve,
		},
		"delay non-Ethereum messages for chain with transfer verifier when Rejected": {
			common.Rejected,
			vaa.ChainIDSepolia,
			Delay,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			n := makeTestNotary(t)
			msg := makeUniqueMessagePublication(t)

			// Set the emitter address to the known token bridge address for the environment.
			msg.EmitterChain = test.emitterChain

			err := msg.SetVerificationState(test.verificationState)
			require.NoError(t, err)

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
		"delay could not verify": {
			common.CouldNotVerify,
			Delay,
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
		"CouldNotVerify gets delayed": {
			common.CouldNotVerify,
			expectedSizes{
				delayed:    1,
				blackholed: 0,
			},
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

func TestNotary_IsDelayed(t *testing.T) {
	n := makeTestNotary(t)
	msg := makeUniqueMessagePublication(t)

	require.False(t, n.IsDelayed(msg), "message should not be delayed before it is added")

	require.NoError(t, n.delay(msg, time.Hour))
	require.True(t, n.IsDelayed(msg), "message should be delayed after delay()")

	require.NoError(t, n.forget(msg))
	require.False(t, n.IsDelayed(msg), "message should not be delayed after forget()")
}

func TestNotary_LoadFromDB(t *testing.T) {
	delayedMsg := makeUniqueMessagePublication(t)
	blackholedMsg := makeUniqueMessagePublication(t)

	t.Run("loads delayed and blackholed messages", func(t *testing.T) {
		database := &configurableMockDB{
			loadResult: &db.NotaryLoadResult{
				Delayed: []*common.PendingMessage{
					{Msg: *delayedMsg, ReleaseTime: time.Now().Add(time.Hour)},
				},
				Blackholed: []*common.MessagePublication{blackholedMsg},
			},
		}
		n := makeTestNotaryWithDB(t, database)

		require.NoError(t, n.loadFromDB(n.logger))
		require.Equal(t, 1, n.delayed.Len())
		require.Equal(t, 1, n.blackholed.Len())
		require.True(t, n.IsDelayed(delayedMsg))
		require.True(t, n.IsBlackholed(blackholedMsg.MessageID()))
	})

	t.Run("empty result initializes empty collections", func(t *testing.T) {
		database := &configurableMockDB{loadResult: &db.NotaryLoadResult{}}
		n := makeTestNotaryWithDB(t, database)

		require.NoError(t, n.loadFromDB(n.logger))
		require.Equal(t, 0, n.delayed.Len())
		require.Equal(t, 0, n.blackholed.Len())
	})

	t.Run("returns error when LoadAll fails", func(t *testing.T) {
		loadErr := errors.New("boom")
		database := &configurableMockDB{loadErr: loadErr}
		n := makeTestNotaryWithDB(t, database)

		require.ErrorIs(t, n.loadFromDB(n.logger), loadErr)
	})

	t.Run("returns error when LoadAll returns nil", func(t *testing.T) {
		database := &configurableMockDB{}
		n := makeTestNotaryWithDB(t, database)

		require.Error(t, n.loadFromDB(n.logger))
	})

	t.Run("returns ErrAlreadyInitialized when delayed queue is not empty", func(t *testing.T) {
		database := &configurableMockDB{loadResult: &db.NotaryLoadResult{}}
		n := makeTestNotaryWithDB(t, database)
		require.NoError(t, n.delay(makeUniqueMessagePublication(t), time.Hour))

		require.ErrorIs(t, n.loadFromDB(n.logger), ErrAlreadyInitialized)
	})
}

// signalingGauge is a prometheus.Gauge that closes done the first time Set is
// called. Tests use it to wait until the notary's background metrics goroutine
// has completed its initial gauge update. That wait establishes a happens-before
// edge with any later mutation of the package-level gauge variables, which the
// race detector otherwise reports as a data race.
type signalingGauge struct {
	prometheus.Gauge
	once sync.Once
	done chan struct{}
}

func newSignalingGauge(name string) *signalingGauge {
	return &signalingGauge{
		Gauge: prometheus.NewGauge(prometheus.GaugeOpts{Name: name}),
		done:  make(chan struct{}),
	}
}

func (g *signalingGauge) Set(v float64) {
	g.Gauge.Set(v)
	g.once.Do(func() { close(g.done) })
}

func TestNotary_Run(t *testing.T) {
	// Initialize the real metrics first so Run's initMetrics call is a no-op and
	// does not overwrite the signaling gauges installed per subtest.
	initMetrics(zap.NewNop())

	runAndWait := func(t *testing.T, n *Notary) {
		t.Helper()

		savedDelayed := notaryDelayedMessagesGauge
		savedBlackholed := notaryBlackholedMessagesGauge
		// The blackholed gauge is written last by updateGauges, so waiting on it
		// guarantees every global gauge access in the goroutine has completed.
		sig := newSignalingGauge("notary_test_run_blackholed")
		notaryDelayedMessagesGauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: "notary_test_run_delayed"})
		notaryBlackholedMessagesGauge = sig
		t.Cleanup(func() {
			notaryDelayedMessagesGauge = savedDelayed
			notaryBlackholedMessagesGauge = savedBlackholed
		})

		require.NoError(t, n.Run())

		select {
		case <-sig.done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for the notary metrics goroutine")
		}
	}

	t.Run("GoTest environment skips database load", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		n := makeTestNotary(t)
		n.ctx = ctx

		runAndWait(t, n)
	})

	t.Run("non-test environment loads from database", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		database := &configurableMockDB{loadResult: &db.NotaryLoadResult{}}
		n := makeTestNotaryWithDB(t, database)
		n.ctx = ctx
		n.env = common.MainNet

		runAndWait(t, n)
	})
}

func TestNotary_InitMetrics_Idempotent(t *testing.T) {
	initMetrics(zap.NewNop())
	require.NotNil(t, notaryReleasedMessagesCounter)

	// A second call must take the early-return branch and not re-register.
	initMetrics(zap.NewNop())
	require.NotNil(t, notaryReleasedMessagesCounter)
}

func TestNotary_UpdateGauges(t *testing.T) {
	savedDelayed := notaryDelayedMessagesGauge
	savedBlackholed := notaryBlackholedMessagesGauge
	t.Cleanup(func() {
		notaryDelayedMessagesGauge = savedDelayed
		notaryBlackholedMessagesGauge = savedBlackholed
	})

	newGauge := func(name string) prometheus.Gauge {
		return prometheus.NewGauge(prometheus.GaugeOpts{Name: name})
	}

	t.Run("returns early when a collection is nil", func(t *testing.T) {
		notaryDelayedMessagesGauge = newGauge("notary_test_delayed_nil_collection")
		notaryBlackholedMessagesGauge = newGauge("notary_test_blackholed_nil_collection")
		n := makeTestNotary(t)
		require.NoError(t, n.delay(makeUniqueMessagePublication(t), time.Hour))
		n.blackholed = nil

		n.updateGauges()

		// The early return must leave the gauges untouched.
		require.Equal(t, float64(0), testutil.ToFloat64(notaryDelayedMessagesGauge))
	})

	t.Run("returns early when metrics are not initialized", func(t *testing.T) {
		notaryDelayedMessagesGauge = nil
		notaryBlackholedMessagesGauge = nil
		n := makeTestNotary(t)

		n.updateGauges()
	})

	t.Run("returns early when only one metric is initialized", func(t *testing.T) {
		notaryDelayedMessagesGauge = newGauge("notary_test_delayed_partial")
		notaryBlackholedMessagesGauge = nil
		n := makeTestNotary(t)

		n.updateGauges()
	})

	t.Run("updates gauges when ready", func(t *testing.T) {
		notaryDelayedMessagesGauge = newGauge("notary_test_delayed_ready")
		notaryBlackholedMessagesGauge = newGauge("notary_test_blackholed_ready")
		n := makeTestNotary(t)
		require.NoError(t, n.delay(makeUniqueMessagePublication(t), time.Hour))

		n.updateGauges()

		require.Equal(t, float64(1), testutil.ToFloat64(notaryDelayedMessagesGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(notaryBlackholedMessagesGauge))
	})
}

func TestNotary_AdminReads(t *testing.T) {
	delayedMsg := makeUniqueMessagePublication(t)
	blackholedMsg := makeUniqueMessagePublication(t)
	loadResult := &db.NotaryLoadResult{
		Delayed:    []*common.PendingMessage{{Msg: *delayedMsg, ReleaseTime: time.Now().Add(time.Hour)}},
		Blackholed: []*common.MessagePublication{blackholedMsg},
	}
	loadErr := errors.New("boom")

	t.Run("GetDelayedMessage returns the matching message", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		got, err := n.GetDelayedMessage(delayedMsg.MessageIDString())
		require.NoError(t, err)
		require.Equal(t, delayedMsg.MessageIDString(), got.Msg.MessageIDString())
	})

	t.Run("GetDelayedMessage returns ErrMsgNotFound for unknown ID", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		_, err := n.GetDelayedMessage(makeUniqueMessagePublication(t).MessageIDString())
		require.ErrorIs(t, err, ErrMsgNotFound)
	})

	t.Run("GetDelayedMessage rejects short IDs", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		_, err := n.GetDelayedMessage("short")
		require.ErrorIs(t, err, ErrInvalidMsgID)
	})

	t.Run("GetDelayedMessage propagates LoadAll errors", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadErr: loadErr})
		_, err := n.GetDelayedMessage(delayedMsg.MessageIDString())
		require.ErrorIs(t, err, loadErr)
	})

	t.Run("GetBlackholedMessage returns the matching message", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		got, err := n.GetBlackholedMessage(blackholedMsg.MessageIDString())
		require.NoError(t, err)
		require.Equal(t, blackholedMsg.MessageIDString(), got.MessageIDString())
	})

	t.Run("GetBlackholedMessage returns ErrMsgNotFound for unknown ID", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		_, err := n.GetBlackholedMessage(makeUniqueMessagePublication(t).MessageIDString())
		require.ErrorIs(t, err, ErrMsgNotFound)
	})

	t.Run("GetBlackholedMessage rejects short IDs", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		_, err := n.GetBlackholedMessage("short")
		require.ErrorIs(t, err, ErrInvalidMsgID)
	})

	t.Run("GetBlackholedMessage propagates LoadAll errors", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadErr: loadErr})
		_, err := n.GetBlackholedMessage(blackholedMsg.MessageIDString())
		require.ErrorIs(t, err, loadErr)
	})

	t.Run("ListDelayedMessages returns all IDs", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		ids, err := n.ListDelayedMessages()
		require.NoError(t, err)
		require.Equal(t, []string{delayedMsg.MessageIDString()}, ids)
	})

	t.Run("ListDelayedMessages propagates LoadAll errors", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadErr: loadErr})
		_, err := n.ListDelayedMessages()
		require.ErrorIs(t, err, loadErr)
	})

	t.Run("ListBlackholedMessages returns all IDs", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: loadResult})
		ids, err := n.ListBlackholedMessages()
		require.NoError(t, err)
		require.Equal(t, []string{blackholedMsg.MessageIDString()}, ids)
	})

	t.Run("ListBlackholedMessages propagates LoadAll errors", func(t *testing.T) {
		n := makeTestNotaryWithDB(t, &configurableMockDB{loadErr: loadErr})
		_, err := n.ListBlackholedMessages()
		require.ErrorIs(t, err, loadErr)
	})
}

func TestNotary_AdminMutations(t *testing.T) {
	const shortID = "short"

	t.Run("BlackholeDelayedMsg rejects short IDs", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.BlackholeDelayedMsg(shortID), ErrInvalidMsgID)
	})

	t.Run("BlackholeDelayedMsg returns ErrMsgNotFound for unknown ID", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.BlackholeDelayedMsg(makeUniqueMessagePublication(t).MessageIDString()), ErrMsgNotFound)
	})

	t.Run("BlackholeDelayedMsg blackholes a delayed message", func(t *testing.T) {
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.NoError(t, n.BlackholeDelayedMsg(msg.MessageIDString()))
		require.Equal(t, 0, n.delayed.Len())
		require.True(t, n.IsBlackholed(msg.MessageID()))
	})

	t.Run("BlackholeDelayedMsg propagates blackhole errors", func(t *testing.T) {
		storeErr := errors.New("store failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{storeBlackholedErr: storeErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.ErrorIs(t, n.BlackholeDelayedMsg(msg.MessageIDString()), storeErr)
	})

	t.Run("ReleaseDelayedMsg rejects short IDs", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.ReleaseDelayedMsg(shortID), ErrInvalidMsgID)
	})

	t.Run("ReleaseDelayedMsg returns ErrMsgNotFound for unknown ID", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.ReleaseDelayedMsg(makeUniqueMessagePublication(t).MessageIDString()), ErrMsgNotFound)
	})

	t.Run("ReleaseDelayedMsg makes a delayed message ready", func(t *testing.T) {
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))
		require.Empty(t, n.ReleaseReadyMessages())

		require.NoError(t, n.ReleaseDelayedMsg(msg.MessageIDString()))
		require.Len(t, n.ReleaseReadyMessages(), 1)
	})

	t.Run("RemoveBlackholedMsg rejects short IDs", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.RemoveBlackholedMsg(shortID), ErrInvalidMsgID)
	})

	t.Run("RemoveBlackholedMsg moves a blackholed message to delayed", func(t *testing.T) {
		msg := makeUniqueMessagePublication(t)
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteBlackholedResult: msg})
		require.NoError(t, n.blackhole(msg))
		require.True(t, n.IsBlackholed(msg.MessageID()))

		require.NoError(t, n.RemoveBlackholedMsg(msg.MessageIDString()))
		require.False(t, n.IsBlackholed(msg.MessageID()))
		require.True(t, n.IsDelayed(msg))
	})

	t.Run("RemoveBlackholedMsg returns ErrInvalidMsg for unknown ID", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.RemoveBlackholedMsg(makeUniqueMessagePublication(t).MessageIDString()), ErrInvalidMsg)
	})

	t.Run("RemoveBlackholedMsg propagates delete errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteBlackholedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.blackhole(msg))

		require.ErrorIs(t, n.RemoveBlackholedMsg(msg.MessageIDString()), deleteErr)
	})

	t.Run("ResetReleaseTimer rejects short IDs", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.ResetReleaseTimer(shortID, 1), ErrInvalidMsgID)
	})

	t.Run("ResetReleaseTimer rejects delays over the max", func(t *testing.T) {
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.ErrorIs(t, n.ResetReleaseTimer(msg.MessageIDString(), MaxDelayDays+1), ErrDelayExceedsMax)
	})

	t.Run("ResetReleaseTimer accepts the maximum delay", func(t *testing.T) {
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.NoError(t, n.ResetReleaseTimer(msg.MessageIDString(), MaxDelayDays))
	})

	t.Run("ResetReleaseTimer returns ErrMsgNotFound for unknown ID", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.ResetReleaseTimer(makeUniqueMessagePublication(t).MessageIDString(), 1), ErrMsgNotFound)
	})

	t.Run("ResetReleaseTimer propagates setDuration errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteDelayedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.ErrorIs(t, n.ResetReleaseTimer(msg.MessageIDString(), 1), deleteErr)
	})

	t.Run("InjectDelayedMessage rejects non-dev environments", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.MainNet
		_, err := n.InjectDelayedMessage(1)
		require.ErrorIs(t, err, ErrNotDevMode)
	})

	t.Run("InjectDelayedMessage rejects delays over the max", func(t *testing.T) {
		n := makeTestNotary(t)
		_, err := n.InjectDelayedMessage(MaxDelayDays + 1)
		require.ErrorIs(t, err, ErrDelayExceedsMax)
	})

	t.Run("InjectDelayedMessage injects in GoTest", func(t *testing.T) {
		n := makeTestNotary(t)
		id, err := n.InjectDelayedMessage(1)
		require.NoError(t, err)
		require.NotEmpty(t, id)
		require.Equal(t, 1, n.delayed.Len())
	})

	t.Run("InjectDelayedMessage injects in devnet", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.UnsafeDevNet
		id, err := n.InjectDelayedMessage(1)
		require.NoError(t, err)
		require.NotEmpty(t, id)
	})

	t.Run("InjectDelayedMessage propagates delay errors", func(t *testing.T) {
		storeErr := errors.New("store failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{storeDelayedErr: storeErr})
		_, err := n.InjectDelayedMessage(1)
		require.ErrorIs(t, err, storeErr)
	})

	t.Run("InjectBlackholedMessage rejects non-dev environments", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.MainNet
		_, err := n.InjectBlackholedMessage()
		require.ErrorIs(t, err, ErrNotDevMode)
	})

	t.Run("InjectBlackholedMessage injects in GoTest", func(t *testing.T) {
		n := makeTestNotary(t)
		id, err := n.InjectBlackholedMessage()
		require.NoError(t, err)
		require.NotEmpty(t, id)
		require.Equal(t, 1, n.blackholed.Len())
	})

	t.Run("InjectBlackholedMessage injects in devnet", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.UnsafeDevNet
		id, err := n.InjectBlackholedMessage()
		require.NoError(t, err)
		require.NotEmpty(t, id)
	})

	t.Run("InjectBlackholedMessage propagates blackhole errors", func(t *testing.T) {
		storeErr := errors.New("store failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{storeBlackholedErr: storeErr})
		_, err := n.InjectBlackholedMessage()
		require.ErrorIs(t, err, storeErr)
	})
}

func TestNotary_ProcessMsg_UnknownTokenBridge(t *testing.T) {
	initMetrics(zap.NewNop())

	n := makeTestNotary(t)
	n.env = common.MainNet

	msg := makeUniqueMessagePublication(t)
	// Sepolia has a transfer verifier but is not a known mainnet token bridge emitter.
	msg.EmitterChain = vaa.ChainIDSepolia
	require.NoError(t, msg.SetVerificationState(common.Valid))

	before := testutil.ToFloat64(notaryErrors.WithLabelValues("unknown_token_bridge"))

	verdict, err := n.ProcessMsg(msg)
	require.Error(t, err)
	require.Equal(t, Unknown, verdict)

	after := testutil.ToFloat64(notaryErrors.WithLabelValues("unknown_token_bridge"))
	require.Equal(t, before+1, after, "unknown token bridge error counter should increment")
}

func TestNotary_ReleaseReadyMessages_SkipsBadEntries(t *testing.T) {
	t.Run("continues past a delete error", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteDelayedErrOnce: deleteErr})
		n.delayed = common.NewPendingMessageQueue()

		first := makeUniqueMessagePublication(t)
		second := makeUniqueMessagePublication(t)
		n.delayed.Push(&common.PendingMessage{Msg: *first, ReleaseTime: time.Now().Add(-2 * time.Hour)})
		n.delayed.Push(&common.PendingMessage{Msg: *second, ReleaseTime: time.Now().Add(-time.Hour)})

		ready := n.ReleaseReadyMessages()
		require.Len(t, ready, 1)
		require.Equal(t, second.MessageIDString(), ready[0].MessageIDString())
	})

	t.Run("continues past a blackholed message in the delayed queue", func(t *testing.T) {
		n := makeTestNotary(t)
		n.delayed = common.NewPendingMessageQueue()

		blackholed := makeUniqueMessagePublication(t)
		ready := makeUniqueMessagePublication(t)
		n.delayed.Push(&common.PendingMessage{Msg: *blackholed, ReleaseTime: time.Now().Add(-2 * time.Hour)})
		n.delayed.Push(&common.PendingMessage{Msg: *ready, ReleaseTime: time.Now().Add(-time.Hour)})
		n.blackholed.Add(blackholed.MessageID())

		got := n.ReleaseReadyMessages()
		require.Len(t, got, 1)
		require.Equal(t, ready.MessageIDString(), got[0].MessageIDString())
	})
}

func TestNotary_NewNotary(t *testing.T) {
	guardianDB := db.OpenDb(zap.NewNop(), nil)
	t.Cleanup(func() { require.NoError(t, guardianDB.Close()) })

	n := NewNotary(context.Background(), zap.NewNop(), guardianDB, common.GoTest)
	require.NotNil(t, n)
	require.NotNil(t, n.database)
	require.NotNil(t, n.delayed)
	require.Nil(t, n.blackholed)
	require.Equal(t, common.GoTest, n.env)
}

func TestVerdict_String(t *testing.T) {
	tests := map[Verdict]string{
		Approve:     "Approve",
		Delay:       "Delay",
		Blackhole:   "Blackhole",
		Unknown:     "Unknown",
		Verdict(99): "Unknown",
	}
	for verdict, want := range tests {
		require.Equal(t, want, verdict.String())
	}
}

func TestNotary_Run_LoadError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loadErr := errors.New("load failed")
	n := makeTestNotaryWithDB(t, &configurableMockDB{loadErr: loadErr})
	n.ctx = ctx
	n.env = common.MainNet

	require.ErrorIs(t, n.Run(), loadErr)
}

func TestNotary_ProcessMsg_EmitterChecks(t *testing.T) {
	t.Run("approves a transfer from a non-token-bridge emitter on a known chain", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.MainNet

		msg := makeUniqueMessagePublication(t)
		msg.EmitterChain = vaa.ChainIDEthereum
		other, err := vaa.StringToAddress("0x1111111111111111111111111111111111111111")
		require.NoError(t, err)
		msg.EmitterAddress = other
		require.NoError(t, msg.SetVerificationState(common.Valid))

		verdict, err := n.ProcessMsg(msg)
		require.NoError(t, err)
		require.Equal(t, Approve, verdict)
	})

	t.Run("delays an anomalous transfer from the token bridge", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.MainNet

		msg := makeUniqueMessagePublication(t)
		msg.EmitterChain = vaa.ChainIDEthereum
		tokenBridge := sdk.KnownTokenbridgeEmitters[vaa.ChainIDEthereum]
		msg.EmitterAddress = vaa.Address(tokenBridge)
		require.NoError(t, msg.SetVerificationState(common.Anomalous))

		verdict, err := n.ProcessMsg(msg)
		require.NoError(t, err)
		require.Equal(t, Delay, verdict)
	})

	t.Run("returns Unknown for a devnet chain with no known devnet token bridge", func(t *testing.T) {
		n := makeTestNotary(t)
		n.env = common.UnsafeDevNet

		msg := makeUniqueMessagePublication(t)
		msg.EmitterChain = vaa.ChainIDSepolia
		require.NoError(t, msg.SetVerificationState(common.Valid))

		verdict, err := n.ProcessMsg(msg)
		require.Error(t, err)
		require.Equal(t, Unknown, verdict)
	})

	t.Run("returns Blackhole for an already-blackholed message", func(t *testing.T) {
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.blackhole(msg))

		verdict, err := n.ProcessMsg(msg)
		require.NoError(t, err)
		require.Equal(t, Blackhole, verdict)
	})

	t.Run("increments the non-approve counter for delayed messages", func(t *testing.T) {
		initMetrics(zap.NewNop())
		n := makeTestNotary(t)
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, msg.SetVerificationState(common.Anomalous))

		before := testutil.ToFloat64(notaryTokenTransferNonApprove.WithLabelValues(Delay.String()))
		verdict, err := n.ProcessMsg(msg)
		require.NoError(t, err)
		require.Equal(t, Delay, verdict)
		after := testutil.ToFloat64(notaryTokenTransferNonApprove.WithLabelValues(Delay.String()))
		require.Equal(t, before+1, after)
	})
}

func TestNotary_ReleaseReadyMessages_NilReceiver(t *testing.T) {
	var n *Notary
	require.Nil(t, n.ReleaseReadyMessages())
}

func TestNotary_Blackhole_Errors(t *testing.T) {
	t.Run("rejects nil message", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.blackhole(nil), ErrInvalidMsg)
	})

	t.Run("propagates removeDelayed errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteDelayedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.ErrorIs(t, n.blackhole(msg), deleteErr)
	})
}

func TestNotary_Forget_Errors(t *testing.T) {
	t.Run("rejects nil message", func(t *testing.T) {
		n := makeTestNotary(t)
		require.ErrorIs(t, n.forget(nil), ErrInvalidMsg)
	})

	t.Run("propagates removeDelayed errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteDelayedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		require.ErrorIs(t, n.forget(msg), deleteErr)
	})

	t.Run("propagates removeBlackholed errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteBlackholedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.blackhole(msg))

		require.ErrorIs(t, n.forget(msg), deleteErr)
	})
}

func TestNotary_RemoveBlackholed_Nil(t *testing.T) {
	n := makeTestNotary(t)
	_, err := n.removeBlackholed(nil)
	require.ErrorIs(t, err, ErrInvalidMsg)
}

func TestNotary_Release_SetDuration_Nil(t *testing.T) {
	n := makeTestNotary(t)
	require.ErrorIs(t, n.release(nil), ErrInvalidMsg)
	require.ErrorIs(t, n.setDuration(nil, 0), ErrInvalidMsg)
}

func TestNotary_SetDuration_PropagatesDelayError(t *testing.T) {
	storeErr := errors.New("store failed")
	n := makeTestNotary(t)
	msg := makeUniqueMessagePublication(t)
	require.NoError(t, n.delay(msg, time.Hour))

	// Swap in a database that fails the next StoreDelayed call.
	n.database = &configurableMockDB{storeDelayedErr: storeErr}

	require.ErrorIs(t, n.setDuration(msg.MessageID(), time.Hour), storeErr)
}

func TestNotary_RemoveDelayed_Errors(t *testing.T) {
	t.Run("rejects nil msgID", func(t *testing.T) {
		n := makeTestNotary(t)
		_, err := n.removeDelayed(nil)
		require.ErrorIs(t, err, ErrInvalidMsg)
	})

	t.Run("propagates DeleteDelayed errors", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		n := makeTestNotaryWithDB(t, &configurableMockDB{deleteDelayedErr: deleteErr})
		msg := makeUniqueMessagePublication(t)
		require.NoError(t, n.delay(msg, time.Hour))

		_, err := n.removeDelayed(msg.MessageID())
		require.ErrorIs(t, err, deleteErr)
	})
}

func TestMsgPubSet_NilReceiver(t *testing.T) {
	var s *msgPubSet
	require.Equal(t, 0, s.Len())
	require.False(t, s.Contains([]byte("x")))
	s.Add([]byte("x"))
	s.Remove([]byte("x"))
}

func TestNotary_AdminBoundaryMsgIDs(t *testing.T) {
	boundaryID := strings.Repeat("a", common.MinMsgIdLen)
	require.Len(t, boundaryID, common.MinMsgIdLen)

	n := makeTestNotaryWithDB(t, &configurableMockDB{loadResult: &db.NotaryLoadResult{}})

	require.NotErrorIs(t, n.BlackholeDelayedMsg(boundaryID), ErrInvalidMsgID)
	require.NotErrorIs(t, n.ReleaseDelayedMsg(boundaryID), ErrInvalidMsgID)
	require.NotErrorIs(t, n.RemoveBlackholedMsg(boundaryID), ErrInvalidMsgID)
	require.NotErrorIs(t, n.ResetReleaseTimer(boundaryID, 1), ErrInvalidMsgID)

	_, err := n.GetDelayedMessage(boundaryID)
	require.NotErrorIs(t, err, ErrInvalidMsgID)
	_, err = n.GetBlackholedMessage(boundaryID)
	require.NotErrorIs(t, err, ErrInvalidMsgID)
}

func TestNotary_ResetReleaseTimer_DelayValue(t *testing.T) {
	n := makeTestNotary(t)
	msg := makeUniqueMessagePublication(t)
	require.NoError(t, n.delay(msg, time.Hour))

	require.NoError(t, n.ResetReleaseTimer(msg.MessageIDString(), 2))

	pMsg := n.delayed.Peek()
	require.NotNil(t, pMsg)
	require.WithinDuration(t, time.Now().Add(2*24*time.Hour), pMsg.ReleaseTime, time.Minute)
}

func TestNotary_InjectDelayedMessage_DelayValue(t *testing.T) {
	t.Run("accepts the maximum delay", func(t *testing.T) {
		n := makeTestNotary(t)
		id, err := n.InjectDelayedMessage(MaxDelayDays)
		require.NoError(t, err)
		require.NotEmpty(t, id)
	})

	t.Run("sets the requested release time", func(t *testing.T) {
		n := makeTestNotary(t)
		_, err := n.InjectDelayedMessage(2)
		require.NoError(t, err)

		pMsg := n.delayed.Peek()
		require.NotNil(t, pMsg)
		require.WithinDuration(t, time.Now().Add(2*24*time.Hour), pMsg.ReleaseTime, time.Minute)
	})
}

func TestCreateTestMessagePublication(t *testing.T) {
	msg := createTestMessagePublication()
	require.NotNil(t, msg)
	require.True(t, msg.Unreliable)
	require.True(t, msg.IsReobservation)
	require.True(t, vaa.IsTransfer(msg.Payload))
}

func TestEncodePayloadBytes_MaxAmount(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	require.NoError(t, err)
	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	maxAmount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	require.Len(t, maxAmount.Bytes(), 32)

	payload := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        maxAmount,
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	require.NotPanics(t, func() { _ = encodePayloadBytes(payload) })
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
	tokenBridge := sdk.KnownDevnetTokenbridgeEmitters[vaa.ChainIDEthereum]
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
