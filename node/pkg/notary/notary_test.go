package notary

import (
	"context"
	"errors"
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

// errMockKeyNotFound is the error that MockNotaryDB returns when a delete names a missing key.
var errMockKeyNotFound = errors.New("mock notary db: key not found")

// MockNotaryDB is an in-memory implementation of the NotaryDB interface. It follows the real
// database: a store overwrites the entry, and a delete of a missing key returns an error.
// Where possible, database behavior should be tested in the Notary database's own unit tests, not here.
type MockNotaryDB struct {
	mu         sync.Mutex
	delayed    map[string]common.PendingMessage
	blackholed map[string]common.MessagePublication
}

func newMockNotaryDB() *MockNotaryDB {
	return &MockNotaryDB{
		delayed:    make(map[string]common.PendingMessage),
		blackholed: make(map[string]common.MessagePublication),
	}
}

func (md *MockNotaryDB) StoreBlackholed(m *common.MessagePublication) error {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.blackholed[m.MessageIDString()] = *m
	return nil
}

func (md *MockNotaryDB) StoreDelayed(p *common.PendingMessage) error {
	md.mu.Lock()
	defer md.mu.Unlock()
	md.delayed[p.Msg.MessageIDString()] = *p
	return nil
}

func (md *MockNotaryDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	md.mu.Lock()
	defer md.mu.Unlock()
	m, ok := md.blackholed[string(msgID)]
	if !ok {
		return nil, errMockKeyNotFound
	}
	delete(md.blackholed, string(msgID))
	return &m, nil
}

func (md *MockNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	md.mu.Lock()
	defer md.mu.Unlock()
	p, ok := md.delayed[string(msgID)]
	if !ok {
		return nil, errMockKeyNotFound
	}
	delete(md.delayed, string(msgID))
	return &p, nil
}

func (md *MockNotaryDB) LoadAll(l *zap.Logger) (*db.NotaryLoadResult, error) {
	md.mu.Lock()
	defer md.mu.Unlock()
	result := &db.NotaryLoadResult{}
	for _, p := range md.delayed {
		result.Delayed = append(result.Delayed, &p)
	}
	for _, m := range md.blackholed {
		result.Blackholed = append(result.Blackholed, &m)
	}
	return result, nil
}

func makeTestNotary(t *testing.T) *Notary {
	t.Helper()

	return &Notary{
		ctx:    context.Background(),
		logger: zap.NewNop(),
		state:  newNotaryState(zap.NewNop(), newMockNotaryDB()),
		env:    common.GoTest,
	}
}

func TestNotary_NilInputsAndDependencies(t *testing.T) {
	logger := zap.NewNop()

	notary, err := NewNotary(context.Background(), logger, nil, common.GoTest)
	require.Error(t, err)
	require.Nil(t, notary)

	guardianDB := db.OpenDb(logger, nil)
	t.Cleanup(func() { _ = guardianDB.Close() })

	notary, err = NewNotary(context.Background(), nil, guardianDB, common.GoTest)
	require.Error(t, err)
	require.Nil(t, notary)

	notary, err = NewNotary(context.Background(), logger, guardianDB, common.GoTest)
	require.NoError(t, err)
	require.NotNil(t, notary)

	verdict, err := notary.ProcessMsg(nil)
	require.Error(t, err)
	require.Equal(t, Unknown, verdict)
	require.False(t, notary.IsDelayed(nil))
}

// TestNewNotary_LoadsPersistedState checks that NewNotary returns a Notary that already holds the
// messages stored in the database, without a call to Run.
func TestNewNotary_LoadsPersistedState(t *testing.T) {
	logger := zap.NewNop()
	guardianDB := db.OpenDb(logger, nil)
	t.Cleanup(func() { _ = guardianDB.Close() })

	delayedMsg := makeUniqueMessagePublication(t)
	blackholedMsg := makeUniqueMessagePublication(t)

	notaryDB := db.NewNotaryDB(guardianDB.Conn())
	require.NoError(t, notaryDB.StoreDelayed(&common.PendingMessage{
		ReleaseTime: time.Unix(time.Now().Add(time.Hour).Unix(), 0),
		Msg:         *delayedMsg,
	}))
	require.NoError(t, notaryDB.StoreBlackholed(blackholedMsg))

	n, err := NewNotary(context.Background(), logger, guardianDB, common.GoTest)
	require.NoError(t, err)

	require.True(t, n.IsDelayed(delayedMsg))
	require.True(t, n.IsBlackholed(blackholedMsg.MessageID()))
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
				n.state.delayed.Len(),
				fmt.Sprintf("delayed count did not match. verificationState %s", msg.VerificationState().String()),
			)
			require.Equal(
				t,
				test.expectedSizes.blackholed,
				n.state.blackholed.Len(),
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
			n.state.delayed = common.NewPendingMessageQueue()

			currentLength := n.state.delayed.Len()
			for pMsg := range slices.Values(tt.delayed) {
				require.NotNil(t, pMsg)
				n.state.delayed.Push(pMsg)
				// Store the message too, as delay does, so that the release can delete it.
				require.NoError(t, n.state.database.StoreDelayed(pMsg))
				// Ensure that the queue grows after each push.
				require.Greater(t, n.state.delayed.Len(), currentLength)
				currentLength = n.state.delayed.Len()
			}
			require.Equal(t, len(tt.delayed), n.state.delayed.Len())

			readyMsgs := n.ReleaseReadyMessages()
			require.Equal(t, tt.expectedReadyCount, len(readyMsgs), "ready length does not match")
			require.Equal(t, tt.expectedDelayCount, n.state.delayed.Len(), "delayed length does not match")
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
			"remove from delayed queue",
			makeUniqueMessagePublication(t),
			0,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.state.delayed = common.NewPendingMessageQueue()
			n.state.blackholed = NewSet()

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 0, n.state.blackholed.Len())

			err := n.state.delay(tt.msg, time.Hour)
			require.NoError(t, err)

			require.Equal(t, 1, n.state.delayed.Len())
			require.Equal(t, 0, n.state.blackholed.Len())

			// Modify the set and the database manually because calling the blackhole function will
			// remove the message from the delayed queue.
			n.state.blackholed.Add(tt.msg.MessageID())
			require.NoError(t, n.state.database.StoreBlackholed(tt.msg))

			require.Equal(t, 1, n.state.delayed.Len())
			require.Equal(t, 1, n.state.blackholed.Len())

			forgetErr := n.state.forget(tt.msg)
			require.NoError(t, forgetErr)

			require.Equal(t, tt.expectedDelayCount, n.state.delayed.Len())
			require.Equal(t, tt.expectedBlackholed, n.state.blackholed.Len())
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
			"remove from delayed queue",
			makeUniqueMessagePublication(t),
			0,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set-up
			n := makeTestNotary(t)
			n.state.delayed = common.NewPendingMessageQueue()
			n.state.blackholed = NewSet()

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 0, n.state.blackholed.Len())

			err := n.state.delay(tt.msg, time.Hour)
			require.NoError(t, err)

			require.Equal(t, 1, n.state.delayed.Len())
			require.Equal(t, 0, n.state.blackholed.Len())

			blackholeErr := n.state.blackhole(tt.msg)
			require.NoError(t, blackholeErr)

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 1, n.state.blackholed.Len())
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
			n.state.delayed = common.NewPendingMessageQueue()
			n.state.blackholed = NewSet()

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 0, n.state.blackholed.Len())

			err := n.state.blackhole(tt.msg)
			require.NoError(t, err)

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 1, n.state.blackholed.Len())

			err = n.state.delay(tt.msg, time.Hour)
			require.ErrorIs(t, err, ErrAlreadyBlackholed)

			require.Equal(t, 0, n.state.delayed.Len())
			require.Equal(t, 1, n.state.blackholed.Len())
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
			n.state.delayed = common.NewPendingMessageQueue()
			n.state.blackholed = NewSet()

			require.Equal(t, 0, n.state.delayed.Len())

			// Delay a message; ensure no messages are ready
			delayErr := n.state.delay(tt.msg, time.Hour)
			require.NoError(t, delayErr)
			require.Equal(t, 1, n.state.delayed.Len())
			require.Empty(t, n.ReleaseReadyMessages())
			require.Equal(t, 1, n.state.delayed.Len())

			// Release the message
			releaseErr := n.state.reschedule(tt.msg.MessageID(), 0)
			require.NoError(t, releaseErr)

			// Check that a new message is ready
			require.Len(t, n.ReleaseReadyMessages(), 1)
			require.Equal(t, 0, n.state.delayed.Len())
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

// TestNotary_ReobservationDoesNotBypassDelay verifies that once a message is delayed, a
// reobservation of that message cannot release it early or duplicate its entry in the delayed
// queue. Reobservation re-runs the transfer verifier from scratch, so the same message can
// re-enter ProcessMsg carrying a different verification state.
func TestNotary_ReobservationDoesNotBypassDelay(t *testing.T) {
	// Verification states that a reobserved message might carry on re-entry.
	tests := map[string]common.VerificationState{
		"reobserved as Valid":       common.Valid,
		"reobserved as Rejected":    common.Rejected,
		"reobserved as Anomalous":   common.Anomalous,
		"reobserved as NotVerified": common.NotVerified,
	}

	for name, reobservedState := range tests {
		t.Run(name, func(t *testing.T) {
			n := makeTestNotary(t)

			msg := makeUniqueMessagePublication(t)
			require.NoError(t, msg.SetVerificationState(common.CouldNotVerify))

			verdict, err := n.ProcessMsg(msg)
			require.NoError(t, err)
			require.Equal(t, Delay, verdict)
			require.True(t, n.IsDelayed(msg))
			require.Equal(t, 1, n.state.delayed.Len())

			releaseTime := n.state.delayed.Peek().ReleaseTime

			// Reobservation republishes the message with the same message ID and has the transfer
			// verifier evaluate it again from the default state.
			reobserved := makeUniqueMessagePublication(t)
			reobserved.Sequence = msg.Sequence
			reobserved.IsReobservation = true
			if reobservedState != common.NotVerified {
				// SetVerificationState refuses to set the state it already holds.
				require.NoError(t, reobserved.SetVerificationState(reobservedState))
			}
			require.Equal(t, msg.MessageIDString(), reobserved.MessageIDString())

			verdict, err = n.ProcessMsg(reobserved)
			require.NoError(t, err)
			require.Equal(t, Delay, verdict, "reobservation must not release a delayed message early")

			// The original delayed entry is untouched, so delay() was not called a second time.
			require.Equal(t, 1, n.state.delayed.Len())
			require.Equal(t, releaseTime, n.state.delayed.Peek().ReleaseTime)
		})
	}
}
