package near

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	mockserver "github.com/certusone/wormhole/node/pkg/watchers/near/nearapi/mock"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	WORMHOLE_CONTRACT   = "contract.wormhole_crypto.near"
	BLOCK_POLL_INTERVAL = time.Millisecond * 50
)

type (
	testCase struct {
		// meta
		name    string
		t       *testing.T
		timeout time.Duration

		// configuration
		wormholeContract      string
		upstreamHost          string // e.g. "https://rpc.mainnet.near.org"
		cacheDir              string
		latestFinalBlocks     []string
		obsvReq               []gossipv1.ObservationRequest
		expectedMsgObserved   []*common.MessagePublication
		expectedMsgReObserved []*common.MessagePublication

		// doneC is written to by the goroutine of testCase.run to communicate its result to the parent testCase.setupAndRun
		doneC chan error
	}
)

var BLOCKCHAIN_1 = []string{
	"A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx", // 76538229
	"9AEuLtXe4JgJGnwY6ZZE6PmkPcEYpQqqUzwDMzUsMgBT", // 76538230
	"Ad7JSCXZTGegrfWLAmqupd1qiEEphpf5azfWayWCPS8G", // 76538231
	"G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv", // 76538232	contains a wormhole transaction
	"G8kF9bVa4WSxYj5hk5YGfk6GZHhGF6eExj6MVciGosjY", // 76538233
	"6zPnFkHojNQpbRgALHgRnbzhFvp55hido4Gv645nR8zf", // 76538234	contains the wormhole transaction receipt
	"G38cqPUZ33Foaaemxtcgq3sXAd64EZark5m6LjjhQb3X", // 76538235
	"6eCgeVSC4Hwm8tAVy4qNQpnLs4S9EpzRjGtAipwZ632A", // 76538236
}

// performance config
func TestMain(m *testing.M) {
	blockPollInterval = BLOCK_POLL_INTERVAL
	initialTxProcDelay = blockPollInterval*2 + time.Millisecond
	workerCountTxProcessing = 1
	os.Exit(m.Run())
}

func portalEmitterAddress() vaa.Address {
	s := "contract.portalbridge.near"
	h := sha256.New()
	h.Write([]byte(s))
	r := h.Sum(nil)

	var a vaa.Address
	copy(a[:], r)
	return a
}

type testMessageTracker struct {
	*common.MessagePublication
	seen bool
}

/*
Stages of the test:
1) The watcher is allowed to make some RPC calls and observe messages
2) Check that all observed messages are correct
3) Run re-observation requests
4) Check that all re-observed messages are correct
*/
func (testCase *testCase) run(ctx context.Context) error {
	logger := zaptest.NewLogger(testCase.t).Named("near.test")

	// Run the mock server
	mockServer := mockserver.NewForwardingCachingServer(logger, testCase.upstreamHost, testCase.cacheDir, testCase.latestFinalBlocks)
	mockHttpServer := httptest.NewServer(mockServer)
	defer mockHttpServer.Close()

	// Setup a watcher
	msgC := make(chan *common.MessagePublication)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)
	mainnet := true // we set mainnet to true because the testdata was collected on mainnet.
	w := NewWatcher(mockHttpServer.URL, testCase.wormholeContract, msgC, obsvReqC, mainnet)

	// Run the watcher
	if err := supervisor.Run(ctx, "nearwatch", w.Run); err != nil {
		testCase.doneC <- err
		return err
	}

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	// assert that messages were observed correctly...
	expectedMsgObserved := map[string]*testMessageTracker{}
	for _, em := range testCase.expectedMsgObserved {
		expectedMsgObserved[em.MessageIDString()] = &testMessageTracker{MessagePublication: em, seen: false}
	}

	for i := 0; i < len(expectedMsgObserved); i++ {
		select {
		case msg := <-msgC:
			assert.Contains(testCase.t, expectedMsgObserved, msg.MessageIDString(), "unexpected message: %v", msg)
			assert.Equal(testCase.t, expectedMsgObserved[msg.MessageIDString()].seen, false, "already observed message: %v", msg)
			assert.Equal(testCase.t, expectedMsgObserved[msg.MessageIDString()].MessagePublication, msg)
			expectedMsgObserved[msg.MessageIDString()].seen = true
		case <-ctx.Done():
			testCase.doneC <- fmt.Errorf("parent ctx cancel/timeout while waiting for normal messages")
			return ctx.Err()
		}
	}

	for publication, b := range expectedMsgObserved {
		if !b.seen {
			assert.Fail(testCase.t, "message not observed: %v", publication)
		}
	}

	// feed in the re-observation requests.
	for k := range testCase.obsvReq {
		select {
		case obsvReqC <- &testCase.obsvReq[k]:
		default:
			err := fmt.Errorf("obsvReqC is full")
			testCase.doneC <- err
			return err
		}
	}

	// assert that messages were re-observed correctly...
	expectedMsgReObserved := map[string]*testMessageTracker{}
	for _, em := range testCase.expectedMsgReObserved {
		em.IsReobservation = true
		expectedMsgReObserved[em.MessageIDString()] = &testMessageTracker{MessagePublication: em, seen: false}
	}

	for i := 0; i < len(expectedMsgReObserved); i++ {
		select {
		case msg := <-msgC:
			assert.Contains(testCase.t, expectedMsgReObserved, msg.MessageIDString(), "unexpected message: %v", msg)
			assert.Equal(testCase.t, expectedMsgReObserved[msg.MessageIDString()].seen, false, "already reobserved message: %v", msg)
			assert.Equal(testCase.t, expectedMsgReObserved[msg.MessageIDString()].MessagePublication, msg)
			expectedMsgReObserved[msg.MessageIDString()].seen = true
		case <-ctx.Done():
			testCase.doneC <- fmt.Errorf("parent ctx cancel/timeout while waiting for re-observed messages")
			return ctx.Err()
		}
	}

	for publication, b := range expectedMsgReObserved {
		if !b.seen {
			assert.Fail(testCase.t, "message not reobserved: %v", publication)
		}
	}

	// there should be no messages left now
	assert.Equal(testCase.t, len(msgC), 0)

	// signal that we're done here
	supervisor.Signal(ctx, supervisor.SignalDone)
	testCase.doneC <- nil
	return nil
}

func (testCase *testCase) setupAndRun(logger *zap.Logger) {
	// Setup context (with timeout) and logger
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), testCase.timeout)
	defer rootCtxCancel()

	testCase.doneC = make(chan error)

	// run the test
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "near-"+testCase.name, testCase.run); err != nil {
			assert.NoError(testCase.t, err)
			return err
		}
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		<-rootCtx.Done()
		supervisor.Signal(ctx, supervisor.SignalDone)
		return nil
	}, supervisor.WithPropagatePanic)

	// wait for result or timeout. Note that if ctx.Done(), then the test runner will send to testCase.doneC as well, so we don't need to check for it here.
	err := <-testCase.doneC
	rootCtxCancel()
	assert.NotEqual(testCase.t, err, context.DeadlineExceeded) // throw an error if timeout
	assert.NoError(testCase.t, err)

	// wait 10ms for things to shut down gracefully
	time.Sleep(time.Millisecond * 10)
}

// TestWatcherSimple() tests the most simple case: "final" API only retruns one block which contains a Wormhole transaction. No re-observation requests.
func TestWatcherSimple(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")
	txHashBytes, _ := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")

	tc := testCase{
		name:             "TestWatcherSimple",
		timeout:          time.Second * 2,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/success/",
		latestFinalBlocks: []string{
			BLOCKCHAIN_1[3],
		},
		expectedMsgObserved: []*common.MessagePublication{
			{
				TxID:             txHashBytes,
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142886047190991)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherSimple2() tests the case where the "final" API returns a sequence of real blocks which contain a single Wormhole transaction. No re-observation requests.
func TestWatcherSimple2(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")
	txHashBytes, _ := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")

	tc := testCase{
		name:             "TestWatcherSimple2",
		timeout:          time.Second * 2,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/success/",
		latestFinalBlocks: []string{
			BLOCKCHAIN_1[0],
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[2],
			BLOCKCHAIN_1[3],
			BLOCKCHAIN_1[4],
			BLOCKCHAIN_1[5],
			BLOCKCHAIN_1[6],
			BLOCKCHAIN_1[7],
		},
		expectedMsgObserved: []*common.MessagePublication{
			{
				TxID:             txHashBytes,
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142886047190991)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherReobservation() tests the simple re-observation case: The "final" endpoint returns
// the same unrelated block and there is a re-observation request for past data.
func TestWatcherReobservation(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")
	txHashBytes, _ := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")

	tc := testCase{
		name:             "TestWatcherReobservation",
		timeout:          time.Second * 5,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/success/",
		latestFinalBlocks: []string{
			"FdJXkyscWxFk8zrZHgahTGCBEcpo4huJNNnuxQ9hgFbW",
		},
		expectedMsgReObserved: []*common.MessagePublication{
			{
				TxID:             txHashBytes,
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142886047190991)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
		obsvReq: []gossipv1.ObservationRequest{
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  txHashBytes,
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherDelayedFinal() tests the case where a block cannot be finalized by a parent having it as
// last_final_block and instead needs to be finalized by having it observed as finalized during polling
func TestWatcherDelayedFinal(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")
	txHashBytes, _ := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")

	lfb := make([]string, 30)
	for i := 0; i < 30; i++ {
		lfb[i] = BLOCKCHAIN_1[3]
	}
	lfb[0] = BLOCKCHAIN_1[0]
	lfb[1] = BLOCKCHAIN_1[1]
	lfb[2] = BLOCKCHAIN_1[2]
	lfb[3] = BLOCKCHAIN_1[3]
	lfb[29] = BLOCKCHAIN_1[7]

	tc := testCase{
		name:              "TestWatcherDelayedFinal",
		timeout:           time.Second * 2,
		t:                 t,
		wormholeContract:  WORMHOLE_CONTRACT,
		upstreamHost:      "",
		cacheDir:          "nearapi/mock/success_mod1/",
		latestFinalBlocks: lfb,
		expectedMsgObserved: []*common.MessagePublication{
			{
				TxID:             txHashBytes,
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142886047190991)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherDelayedFinalAndGaps() tests the case where a block cannot be finalized by a parent having it as
// last_final_block and instead needs to be finalized by having it observed as finalized during polling
// additionally, there is a large gap between polls
func TestWatcherDelayedFinalAndGaps(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")
	txHashBytes, _ := hex.DecodeString("88029cf0e7432cec04c266a3e72903ee6650b4624c7f9c8e22b04d78e18e87f8")

	tc := testCase{
		name:             "TestWatcherDelayedFinalAndGaps",
		timeout:          time.Second * 2,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/success_mod1/",
		latestFinalBlocks: []string{
			BLOCKCHAIN_1[0],
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[7],
		},
		expectedMsgObserved: []*common.MessagePublication{
			{
				TxID:             txHashBytes,
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142886047190991)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherSynthetic(): Case where there are three wormhole messages. Test data is generated (not real)
/*
"A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx", // 76538229 block 0: tx1
"9AEuLtXe4JgJGnwY6ZZE6PmkPcEYpQqqUzwDMzUsMgBT", // 76538230 block 1: tx2 & tx1 receipt
"Ad7JSCXZTGegrfWLAmqupd1qiEEphpf5azfWayWCPS8G", // 76538231 block 2:
"G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv", // 76538232	block 3: tx2 receipt
"G8kF9bVa4WSxYj5hk5YGfk6GZHhGF6eExj6MVciGosjY", // 76538233 block 4:
"6zPnFkHojNQpbRgALHgRnbzhFvp55hido4Gv645nR8zf", // 76538234 block 5:
"G38cqPUZ33Foaaemxtcgq3sXAd64EZark5m6LjjhQb3X", // 76538235 block 6: tx3
"6eCgeVSC4Hwm8tAVy4qNQpnLs4S9EpzRjGtAipwZ632A", // 76538236 block 7: tx3 receipt
*/
func TestWatcherSynthetic(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")

	tc := testCase{
		name:             "TestWatcherSynthetic",
		timeout:          time.Second * 2,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/synthetic/",
		latestFinalBlocks: []string{
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[7],
		},
		expectedMsgReObserved: []*common.MessagePublication{
			{
				TxID:             []byte("_____________________________TX1"),
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142881679761455)/1_000_000_000, 0),
				Unreliable:       false,
			},
			{
				TxID:             []byte("_____________________________TX2"),
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         262,
				Timestamp:        time.Unix(int64(1666142883857047319)/1_000_000_000, 0),
				Unreliable:       false,
			},
			{
				TxID:             []byte("_____________________________TX3"),
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         263,
				Timestamp:        time.Unix(int64(1666142889057341406)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
		obsvReq: []gossipv1.ObservationRequest{
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("TX0_wrong_block_________________"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("TX0_wrong_sequence______________"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX1"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX2"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX3"),
			},
		},
	}

	tc.setupAndRun(logger)
}

// TestWatcherUnfinalized(): Same as synthetic, but one of the blocks is not finalized and that message has to be excluded.
// 1: gets finalized
// 2: doesn't get finalized
// 3: gets finalized
/*
"A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx", // 76538229 block 0: tx1
"9AEuLtXe4JgJGnwY6ZZE6PmkPcEYpQqqUzwDMzUsMgBT", // 76538230 block 1: tx2 & tx1 receipt
"Ad7JSCXZTGegrfWLAmqupd1qiEEphpf5azfWayWCPS8G", // 76538231 block 2:
"G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv", // 76538232	block 3: (not finalized) tx2 receipt
"G8kF9bVa4WSxYj5hk5YGfk6GZHhGF6eExj6MVciGosjY", // 76538233 block 4:
"6zPnFkHojNQpbRgALHgRnbzhFvp55hido4Gv645nR8zf", // 76538234 block 5:
"G38cqPUZ33Foaaemxtcgq3sXAd64EZark5m6LjjhQb3X", // 76538235 block 6: tx3
"6eCgeVSC4Hwm8tAVy4qNQpnLs4S9EpzRjGtAipwZ632A", // 76538236 block 7: tx3 receipt
*/
func TestWatcherUnfinalized(t *testing.T) {
	logger := zaptest.NewLogger(t)

	pl, _ := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000")

	tc := testCase{
		name:             "TestWatcherUnfinalized",
		timeout:          time.Second * 2,
		t:                t,
		wormholeContract: WORMHOLE_CONTRACT,
		upstreamHost:     "",
		cacheDir:         "nearapi/mock/unfinalized/",
		latestFinalBlocks: []string{
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[1],
			BLOCKCHAIN_1[7],
		},
		expectedMsgReObserved: []*common.MessagePublication{
			{
				TxID:             []byte("_____________________________TX1"),
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         261,
				Timestamp:        time.Unix(int64(1666142881679761455)/1_000_000_000, 0),
				Unreliable:       false,
			},
			{
				TxID:             []byte("_____________________________TX3"),
				EmitterAddress:   portalEmitterAddress(),
				ConsistencyLevel: 0,
				EmitterChain:     vaa.ChainIDNear,
				Nonce:            76538233,
				Payload:          pl,
				Sequence:         263,
				Timestamp:        time.Unix(int64(1666142889057341406)/1_000_000_000, 0),
				Unreliable:       false,
			},
		},
		obsvReq: []gossipv1.ObservationRequest{
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("TX0_wrong_block_________________"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("TX0_wrong_sequence______________"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX1"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX2"),
			},
			{
				ChainId: uint32(vaa.ChainIDNear),
				TxHash:  []byte("_____________________________TX3"),
			},
		},
	}

	tc.setupAndRun(logger)
}

func TestSuccessValueToInt(t *testing.T) {

	type test struct {
		input  string
		output uint64
	}

	testsPositive := []test{
		{"MjU=", 25},
		{"MjQ4", 248},
	}

	testsNegative := []test{
		{"", 0},
		{"?", 0},
		{"MjQ4=", 0},
		{"eAo=", 0},
		{"Cg==", 0},
	}

	for _, tc := range testsPositive {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToUint64(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NoError(t, err)
		})
	}

	for _, tc := range testsNegative {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToUint64(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NotNil(t, err)
		})
	}
}
