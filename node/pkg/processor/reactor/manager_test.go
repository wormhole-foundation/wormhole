package reactor

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"

	common2 "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type (
	managerTestContext struct {
		guardianKeys []*ecdsa.PrivateKey
		gs           *common2.GuardianSet

		obsC       chan<- *testObservation
		signedObsC chan<- *gossipv1.SignedObservation

		storage      *testConsensusStorage
		eventHandler *testManagerEventHandler[*testObservation]
	}

	managerTestAction interface {
		Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext)
	}

	managerInjectForeignObservationAction struct {
		hash          [32]byte
		guardianIndex int
		invalid       bool
	}

	managerInjectOwnObservationAction struct {
		observation *testObservation
	}

	managerAssertNumReactorsAction struct {
		numReactors int
	}

	managerAssertHasReactorAction struct {
		digest common.Hash
	}

	managerAssertQuorumEventAction struct {
		observation *testObservation
	}

	managerAssertFinalizationEventAction struct {
		observation *testObservation
	}

	managerAssertTimeoutEventAction struct {
		previousState State
		observation   *testObservation
		lenSignatures int
	}

	managerAssertNoEvents struct {
	}

	managerAssertMessageInStorageAction struct {
		id             string
		shouldNotExist bool
	}

	managerWaitAction struct {
		duration time.Duration
	}
)

func (m managerAssertNoEvents) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	testContext.eventHandler.l.Lock()
	defer testContext.eventHandler.l.Unlock()

	require.Empty(t, testContext.eventHandler.quorumEvents, m)
	require.Empty(t, testContext.eventHandler.timeoutEvents, m)
	require.Empty(t, testContext.eventHandler.finalizationEvents, m)
}

func (m managerAssertMessageInStorageAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	_, _, found, err := testContext.storage.GetSignedObservation(m.id)
	require.NoError(t, err, m)
	require.Equal(t, !m.shouldNotExist, found, m)
}

func (m managerAssertQuorumEventAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	testContext.eventHandler.l.Lock()
	defer testContext.eventHandler.l.Unlock()

	require.GreaterOrEqual(t, len(testContext.eventHandler.quorumEvents), 1, m)
	event := testContext.eventHandler.quorumEvents[0]

	require.Equal(t, m.observation, event.observation, m)
	require.GreaterOrEqual(t, len(event.signatures), vaa.CalculateQuorum(len(testContext.gs.Keys)), m)

	testContext.eventHandler.quorumEvents = testContext.eventHandler.quorumEvents[1:]
}

func (m managerAssertFinalizationEventAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	testContext.eventHandler.l.Lock()
	defer testContext.eventHandler.l.Unlock()

	require.GreaterOrEqual(t, len(testContext.eventHandler.finalizationEvents), 1, m)
	event := testContext.eventHandler.finalizationEvents[0]

	require.Equal(t, m.observation, event.observation, m)
	require.GreaterOrEqual(t, len(event.signatures), vaa.CalculateQuorum(len(testContext.gs.Keys)), m)

	testContext.eventHandler.finalizationEvents = testContext.eventHandler.finalizationEvents[1:]
}

func (m managerAssertTimeoutEventAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	testContext.eventHandler.l.Lock()
	defer testContext.eventHandler.l.Unlock()

	require.GreaterOrEqual(t, len(testContext.eventHandler.timeoutEvents), 1, m)
	event := testContext.eventHandler.timeoutEvents[0]

	require.Equal(t, m.observation, event.observation, m)
	require.Equal(t, len(event.signatures), m.lenSignatures, m)
	require.Equal(t, event.previousState, m.previousState, m)

	testContext.eventHandler.timeoutEvents = testContext.eventHandler.timeoutEvents[1:]
}

func (m managerAssertNumReactorsAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	n := 0
	r.IterateReactors(func(digest common.Hash, reactor *ConsensusReactor[*testObservation]) {
		n++
	})

	require.Equal(t, m.numReactors, n, m)
}

func (m managerAssertHasReactorAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	has := false
	r.IterateReactors(func(digest common.Hash, reactor *ConsensusReactor[*testObservation]) {
		if digest == m.digest {
			has = true
		}
	})

	require.True(t, has, m)
}

func (i managerInjectOwnObservationAction) Evaluate(t *testing.T, m *Manager[*testObservation], testContext *managerTestContext) {
	testContext.obsC <- i.observation
}

func (i managerInjectForeignObservationAction) Evaluate(t *testing.T, m *Manager[*testObservation], testContext *managerTestContext) {
	sig, err := ethcrypto.Sign(i.hash[:], testContext.guardianKeys[i.guardianIndex])
	require.NoError(t, err, m)
	signedObservation := &gossipv1.SignedObservation{
		Addr:      testContext.gs.Keys[i.guardianIndex].Bytes(),
		Hash:      i.hash[:],
		Signature: sig,
		TxHash:    []byte{12, 13},
		MessageId: "test/message",
	}
	if i.invalid {
		n, err := rand.Read(signedObservation.Signature)
		require.NoError(t, err, m)
		require.Equal(t, 65, n, m)
	}
	testContext.signedObsC <- signedObservation
}

func (w managerWaitAction) Evaluate(t *testing.T, r *Manager[*testObservation], testContext *managerTestContext) {
	time.Sleep(w.duration)
}

type (
	testConsensusStorage struct {
		signatures map[string]testConsensusStorageEntry
		l          sync.Mutex
	}

	testConsensusStorageEntry struct {
		observation *testObservation
		signatures  []*vaa.Signature
	}
)

func (t *testConsensusStorage) StoreSignedObservation(observation *testObservation, signatures []*vaa.Signature) error {
	t.l.Lock()
	defer t.l.Unlock()
	t.signatures[observation.MessageID()] = testConsensusStorageEntry{
		observation: observation,
		signatures:  signatures,
	}

	return nil
}

func (t *testConsensusStorage) GetSignedObservation(id string) (observation *testObservation, signatures []*vaa.Signature, found bool, err error) {
	t.l.Lock()
	defer t.l.Unlock()
	v, exists := t.signatures[id]
	if !exists {
		return nil, nil, false, nil
	}

	return v.observation, v.signatures, true, nil
}

type (
	testManagerEventHandler[K Observation] struct {
		quorumEvents []struct {
			observation K
			signatures  []*vaa.Signature
		}
		finalizationEvents []struct {
			observation K
			signatures  []*vaa.Signature
		}
		timeoutEvents []struct {
			previousState State
			digest        common.Hash
			observation   K
			signatures    []*vaa.Signature
		}
		l sync.Mutex
	}
)

func (t *testManagerEventHandler[K]) HandleQuorum(observation K, signatures []*vaa.Signature) {
	t.l.Lock()
	defer t.l.Unlock()
	t.quorumEvents = append(t.quorumEvents, struct {
		observation K
		signatures  []*vaa.Signature
	}{observation: observation, signatures: signatures})
}

func (t *testManagerEventHandler[K]) HandleFinalization(observation K, signatures []*vaa.Signature) {
	t.l.Lock()
	defer t.l.Unlock()
	t.finalizationEvents = append(t.finalizationEvents, struct {
		observation K
		signatures  []*vaa.Signature
	}{observation: observation, signatures: signatures})
}

func (t *testManagerEventHandler[K]) HandleTimeout(previousState State, digest common.Hash, observation K, signatures []*vaa.Signature) {
	t.l.Lock()
	defer t.l.Unlock()
	t.timeoutEvents = append(t.timeoutEvents, struct {
		previousState State
		digest        common.Hash
		observation   K
		signatures    []*vaa.Signature
	}{previousState: previousState, digest: digest, observation: observation, signatures: signatures})
}

func TestManager(t *testing.T) {
	testObservation2 := &testObservation{
		id: []byte{1, 5, 0, 2, 3},
	}
	tests := []struct {
		name         string
		actions      []managerTestAction
		config       *Config
		numGuardians int
		notAGuardian bool
	}{
		{
			name: "NormalFlow2Guardians",
			actions: []managerTestAction{
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 1},
				managerAssertNoEvents{},
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertQuorumEventAction{observation: &testObservation{}},
				managerAssertMessageInStorageAction{id: (&testObservation{}).MessageID()},
				managerWaitAction{duration: time.Millisecond * 24},
				managerAssertFinalizationEventAction{observation: &testObservation{}},
				managerAssertNumReactorsAction{0},
			},
		},
		{
			name: "ForeignFirst2Guardians",
			actions: []managerTestAction{
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 1},
				managerAssertNoEvents{},
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertQuorumEventAction{observation: &testObservation{}},
				managerAssertMessageInStorageAction{id: (&testObservation{}).MessageID()},
				managerWaitAction{duration: time.Millisecond * 24},
				managerAssertFinalizationEventAction{observation: &testObservation{}},
				managerAssertNumReactorsAction{0},
			},
		},
		{
			name: "NoQuorum2Guardians",
			actions: []managerTestAction{
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 1},
				managerAssertNoEvents{},
				managerWaitAction{duration: time.Millisecond * 24},
				managerAssertNumReactorsAction{numReactors: 0},
				managerAssertTimeoutEventAction{observation: &testObservation{}, previousState: StateObserved, lenSignatures: 1},
				managerAssertMessageInStorageAction{id: (&testObservation{}).MessageID(), shouldNotExist: true},
			},
		},
		{
			name: "DuplicateProtectionUsingStorage",
			actions: []managerTestAction{
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 1},
				managerWaitAction{duration: time.Millisecond * 40},
				managerAssertFinalizationEventAction{observation: &testObservation{}},
				managerAssertNumReactorsAction{0},
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 0},
			},
		},
		{
			name: "SignatureVerification",
			actions: []managerTestAction{
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
					invalid:       true,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 0},
			},
		},
		{
			name: "MultipleReactors",
			actions: []managerTestAction{
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 1,
				},
				managerInjectForeignObservationAction{
					hash:          testObservationHash(),
					guardianIndex: 2,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 1},
				managerInjectOwnObservationAction{observation: testObservation2},
				managerInjectForeignObservationAction{
					hash:          testObservation2.SigningDigest(),
					guardianIndex: 2,
				},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertNumReactorsAction{numReactors: 2},
				managerAssertHasReactorAction{digest: testObservationHash()},
				managerAssertHasReactorAction{digest: testObservation2.SigningDigest()},
				managerAssertNoEvents{},
				managerInjectOwnObservationAction{observation: &testObservation{}},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertQuorumEventAction{observation: &testObservation{}},
				managerAssertNoEvents{},
				managerAssertMessageInStorageAction{id: (&testObservation{}).MessageID()},
				managerInjectForeignObservationAction{hash: testObservation2.SigningDigest(), guardianIndex: 1},
				managerWaitAction{duration: time.Millisecond * 4},
				managerAssertQuorumEventAction{observation: testObservation2},
				managerAssertNoEvents{},
				managerWaitAction{time.Millisecond * 24},
				managerAssertNumReactorsAction{0},
				managerAssertFinalizationEventAction{&testObservation{}},
				managerAssertFinalizationEventAction{observation: testObservation2},
				managerAssertNoEvents{},
			},
			numGuardians: 4,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.numGuardians == 0 {
				test.numGuardians = 2
			}
			gst := &common2.GuardianSetState{}
			gs := &common2.GuardianSet{}
			keys := make([]*ecdsa.PrivateKey, test.numGuardians)
			for i := 0; i < test.numGuardians; i++ {
				var err error
				keys[i], err = ethcrypto.GenerateKey()
				require.NoError(t, err)
				gs.Keys = append(gs.Keys, ethcrypto.PubkeyToAddress(keys[i].PublicKey))
			}
			gst.Set(gs)

			var signer Signer
			if !test.notAGuardian {
				signer = &testSigner{privKey: keys[0]}
			}

			config := &Config{
				RetransmitFrequency: time.Millisecond * 10,
				QuorumGracePeriod:   time.Millisecond * 20,
				QuorumTimeout:       time.Millisecond * 20,
				UnobservedTimeout:   time.Millisecond * 20,
				Signer:              signer,
			}
			if test.config != nil {
				config = test.config
				// Make sure we set a signer
				if config.Signer == nil {
					config.Signer = signer
				}
			}
			obsC := make(chan *testObservation, 10)
			signedObsC := make(chan *gossipv1.SignedObservation, 10)
			storage := &testConsensusStorage{
				signatures: make(map[string]testConsensusStorageEntry),
			}
			eventHandler := &testManagerEventHandler[*testObservation]{}
			r := NewManager[*testObservation]("test", obsC, signedObsC, gst, *config, eventHandler, storage)

			tCtx := &managerTestContext{
				guardianKeys: keys,
				gs:           gs,
				obsC:         obsC,
				signedObsC:   signedObsC,
				storage:      storage,
				eventHandler: eventHandler,
			}

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
