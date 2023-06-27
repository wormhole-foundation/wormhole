//nolint:forcetypeassert
package reactor

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"

	"go.uber.org/zap"

	wormhole_common "github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethereum_common "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type testContext struct {
	guardianKeys     []*ecdsa.PrivateKey
	gs               *wormhole_common.GuardianSet
	clock            *clock.Mock
	stateTransitions chan *StateTransition[*testObservation]
}

// testObservation is a mock implementation of the Observation interface that returns the constant `testObservationHash()` as SigningDigest().
type testObservation struct {
	id []byte
}

func (t *testObservation) MessageID() string {
	if t.id == nil {
		return "test/message"
	} else {
		return "test/message/" + hex.EncodeToString(t.id)
	}
}

func testObservationHash() [32]byte {

	return [32]byte{1, 2, 3} // because golang doesn't support constant slices
}

func (t *testObservation) SigningDigest() ethereum_common.Hash {
	if t.id == nil {
		return testObservationHash()
	} else {
		var out [32]byte
		copy(out[:], t.id)
		return out
	}
}

// testGossipSender is a mock implementation of the NetworkAdapter interface.
// Instead of sending out observations, it keeps them in the sentMessages array.
type testGossipSender struct {
	sentMessages     []*gossipv1.SignedObservation
	sentMessagesLock sync.Mutex
}

func (t *testGossipSender) BroadcastObservation(ctx context.Context, observation *gossipv1.SignedObservation) error {
	t.sentMessagesLock.Lock()
	defer t.sentMessagesLock.Unlock()
	t.sentMessages = append(t.sentMessages, observation)
	return nil
}

type testSigner struct {
	privKey *ecdsa.PrivateKey
}

func (t testSigner) Sign(ctx context.Context, digest []byte) (signature []byte, err error) {
	return ethcrypto.Sign(digest, t.privKey)
}

func (t testSigner) Address(ctx context.Context) (ethereum_common.Address, error) {
	return ethcrypto.PubkeyToAddress(t.privKey.PublicKey), nil
}

// Define different types of Test Actions
type (
	// Tests consist of multiple `Actions` executed in sequence.
	// An Action may modify the state of the reactor or make assertions on the state of the reactor.
	// By using this pattern, we can easily define and test different sequences of events.
	testAction interface {
		Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext)
	}

	// waitAction sleeps for `duration`.
	waitAction struct {
		duration time.Duration
	}

	// injectForeignObservationAction injects a `SignedObservation` into the reactor as if it was coming from a different Guardian
	injectForeignObservationAction struct {
		hash          [32]byte
		guardianIndex int  // Index of the Guardian this observation was coming from in testContext.guardianKeys
		invalid       bool // If invalid = true, the Signature on the SignedObservation will be random data instead of a valid signature.
	}

	// injectOwnObservationAction injects a testObservation as if it was observed by the local Guardian.
	injectOwnObservationAction struct {
		observation *testObservation
	}

	// assertObservation asserts the observation in the reactor state
	assertObservation struct {
		observation *testObservation
	}

	// assertLenVAASignatures asserts that the reactor has `expectedLen` many signatures.
	assertLenVAASignatures struct {
		expectedLen int
	}

	// assertSignedObservationOnGossip asserts that the `testGossipSender` has `observation` in its buffer and nothing else.
	// it then clears the buffer.
	assertSignedObservationOnGossip struct {
		observation Observation
	}

	// assertStateAction asserts that the Reactor is in a certain state.
	assertStateAction struct {
		expectedState State
	}

	// assertStateTransitionAction asserts that the Reactor transitions from `fromState` into `expectedState` within `timeout`.
	// If `fromState` is nil, it will only assert `expectedState`.
	// The default `timeout` is 10 * time.Millisecond.
	assertStateTransitionAction struct {
		fromState     *State
		expectedState State
		timeout       time.Duration
	}

	// assertStateTransitionAction asserts that the Reactor does not do a state transition within timeout.
	assertNoStateTransitionAction struct {
		timeout time.Duration
	}
)

func (w waitAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	testContext.clock.Set(testContext.clock.Now().Add(w.duration))
}

func (i injectForeignObservationAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	sig, err := ethcrypto.Sign(i.hash[:], testContext.guardianKeys[i.guardianIndex])
	require.NoError(t, err)
	signedObservation := &gossipv1.SignedObservation{
		Addr:      testContext.gs.Keys[i.guardianIndex].Bytes(),
		Hash:      i.hash[:],
		Signature: sig,
		TxHash:    []byte{12, 13},
		MessageId: "test/message",
	}
	if i.invalid {
		n, err := rand.Read(signedObservation.Signature)
		require.NoError(t, err)
		require.Equal(t, 65, n)
	}
	r.ForeignObservationChannel() <- signedObservation
}

func (i injectOwnObservationAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	r.ObservationChannel() <- i.observation
}

func (a assertObservation) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	require.Equal(t, a.observation, r.Observation(), a)
}

func (a assertLenVAASignatures) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	require.Len(t, r.VAASignatures(), a.expectedLen, a)
}

func (a assertSignedObservationOnGossip) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	// Sleep a little to make sure timeouts can process
	time.Sleep(time.Millisecond * 10)

	r.config.NetworkAdapter.(*testGossipSender).sentMessagesLock.Lock()
	defer r.config.NetworkAdapter.(*testGossipSender).sentMessagesLock.Unlock()
	if a.observation == nil {
		require.Len(t, r.config.NetworkAdapter.(*testGossipSender).sentMessages, 0, a)
		return
	}
	require.Len(t, r.config.NetworkAdapter.(*testGossipSender).sentMessages, 1, a)

	msg := r.config.NetworkAdapter.(*testGossipSender).sentMessages[0]
	require.Equal(t, a.observation.SigningDigest().Bytes(), msg.Hash, a)
	require.Equal(t, a.observation.MessageID(), msg.MessageId, a)

	r.config.NetworkAdapter.(*testGossipSender).sentMessages = nil
}

func (a assertStateAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	require.Equal(t, a.expectedState, r.State(), a)
}

func (a assertStateTransitionAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	timeout := a.timeout
	if timeout == 0 {
		timeout = 10 * time.Millisecond
	}
	select {
	case lastStateTransition := <-testContext.stateTransitions:
		if a.fromState != nil {
			require.Equal(t, *a.fromState, lastStateTransition.FromState, a)
		}
		require.Equal(t, a.expectedState, lastStateTransition.ToState, a)
	case <-time.After(timeout):
		require.Fail(t, "state transition timeout", a)
	}
}

func (a assertNoStateTransitionAction) Evaluate(t *testing.T, r *ConsensusReactor[*testObservation], testContext *testContext) {
	timeout := a.timeout
	if timeout == 0 {
		timeout = 10 * time.Millisecond
	}
	select {
	case lastStateTransition := <-testContext.stateTransitions:
		require.Fail(t, "unexpected state transition", lastStateTransition)
	case <-time.After(timeout):
		return
	}
}

func Test(t *testing.T) {
	tests := []struct {
		name         string
		actions      []testAction
		config       *Config
		numGuardians int
		notAGuardian bool // TODO what is this?
	}{
		{
			name: "NormalFlow2Guardians",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertLenVAASignatures{expectedLen: 1},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				assertLenVAASignatures{expectedLen: 2},
				waitAction{time.Millisecond * 11},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
		},
		{
			name: "NormalFlow4Guardians",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				// Make sure the quorum grace period is considered
				assertNoStateTransitionAction{},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
		},
		{
			name: "TimeoutNoQuorum",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertNoStateTransitionAction{},
				waitAction{time.Millisecond * 20},
				assertStateTransitionAction{expectedState: StateTimedOut},
			},
			numGuardians: 4,
		},
		{
			name: "TimeoutNoObservation",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateUnobserved},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateTimedOut},
			},
			numGuardians: 4,
		},
		{
			name: "LateObservation",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateUnobserved},
				assertLenVAASignatures{expectedLen: 1},
				assertObservation{observation: nil},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertObservation{observation: &testObservation{}},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
		},
		{
			name: "LateObservationNoQuorum",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateUnobserved},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateTimedOut},
			},
			numGuardians: 4,
		},
		{
			name: "NoObservationQuorum",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateUnobserved},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 3,
				},
				assertStateTransitionAction{expectedState: StateQuorumUnobserved},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateTimedOut},
			},
			numGuardians: 4,
		},
		{
			name: "NoObservationQuorumLateObservation",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateUnobserved},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 3,
				},
				assertStateTransitionAction{expectedState: StateQuorumUnobserved},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateQuorum},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
		},
		{
			name: "Resubmission",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateAction{expectedState: StateObserved},
				waitAction{duration: time.Millisecond * 17},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 45},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
			config: &Config{
				RetransmitFrequency: time.Millisecond * 10,
				QuorumGracePeriod:   time.Millisecond * 30,
				QuorumTimeout:       time.Millisecond * 30,
				UnobservedTimeout:   time.Millisecond * 30,
			},
		},
		{
			name: "NoAGuardianObservation",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: nil},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertNoStateTransitionAction{},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				assertNoStateTransitionAction{},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 3,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
			notAGuardian: true,
		},
		{
			name: "SingleGuardian",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 1,
		},
		{
			name: "DuplicateObservation",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertNoStateTransitionAction{},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
		},
		{
			name: "InvalidSignature",
			actions: []testAction{
				assertStateAction{expectedState: StateInitialized},
				injectOwnObservationAction{observation: &testObservation{}},
				assertStateTransitionAction{expectedState: StateObserved},
				assertSignedObservationOnGossip{observation: &testObservation{}},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
					invalid:       true,
				},
				assertNoStateTransitionAction{},
				injectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				assertStateTransitionAction{expectedState: StateQuorum},
				waitAction{time.Millisecond * 15},
				assertStateTransitionAction{expectedState: StateFinalized},
			},
			numGuardians: 4,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.numGuardians == 0 {
				test.numGuardians = 2
			}
			gs := &wormhole_common.GuardianSet{}
			keys := make([]*ecdsa.PrivateKey, test.numGuardians)
			for i := 0; i < test.numGuardians; i++ {
				var err error
				keys[i], err = ethcrypto.GenerateKey()
				require.NoError(t, err)
				gs.Keys = append(gs.Keys, ethcrypto.PubkeyToAddress(keys[i].PublicKey))
			}
			tCtx := &testContext{
				guardianKeys:     keys,
				gs:               gs,
				stateTransitions: make(chan *StateTransition[*testObservation], 10),
				clock:            clock.NewMock(),
			}

			var signer Signer
			isAGuardian := !test.notAGuardian
			if isAGuardian {
				signer = &testSigner{privKey: keys[0]}
			}

			// If no test-specific Reactor config is given, use the default one
			if test.config == nil {

				defaultReactorConfig := &Config{
					RetransmitFrequency: time.Millisecond * 10,
					QuorumGracePeriod:   time.Millisecond * 10,
					QuorumTimeout:       time.Millisecond * 10,
					UnobservedTimeout:   time.Millisecond * 10,
				}

				test.config = defaultReactorConfig
			}

			// Make sure we set a signer
			if test.config.Signer == nil {
				test.config.Signer = signer
			}

			// Set the NetworkAdapter to &testGossipSender{} unless it was overwritten in the test config
			if test.config.NetworkAdapter == nil {
				test.config.NetworkAdapter = &testGossipSender{}
			}

			r := NewReactor[*testObservation]("test", test.config, gs, tCtx.stateTransitions)
			r.clock = tCtx.clock

			func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				logger, err := zap.NewDevelopment()
				require.NoError(t, err)
				supervisor.New(ctx, logger, r.Run)

				for _, action := range test.actions {
					action.Evaluate(t, r, tCtx)
				}
			}()
		})
	}
}
