package node

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	math_rand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/mock"
	eth_crypto "github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2p_peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	eth_common "github.com/ethereum/go-ethereum/common"
)

const LOCAL_RPC_PORTRANGE_START = 10000
const LOCAL_P2P_PORTRANGE_START = 10100
const LOCAL_STATUS_PORTRANGE_START = 10200

var PROMETHEUS_METRIC_VALID_HEARTBEAT_RECEIVED = []byte("wormhole_p2p_broadcast_messages_received_total{type=\"valid_heartbeat\"}")

const WAIT_FOR_LOGS = true
const WAIT_FOR_METRICS = false

type mockGuardian struct {
	p2pKey           libp2p_crypto.PrivKey
	MockObservationC chan *common.MessagePublication
	MockSetC         chan *common.GuardianSet
	gk               *ecdsa.PrivateKey
	guardianAddr     eth_common.Address
	ready            bool
}

func newMockGuardianSet(n int) []*mockGuardian {
	gs := make([]*mockGuardian, n)

	for i := 0; i < n; i++ {
		// generate guardian key
		gk, err := ecdsa.GenerateKey(eth_crypto.S256(), rand.Reader)
		if err != nil {
			panic(err)
		}

		gs[i] = &mockGuardian{
			p2pKey:           devnet.DeterministicP2PPrivKeyByIndex(int64(i)),
			MockObservationC: make(chan *common.MessagePublication),
			MockSetC:         make(chan *common.GuardianSet),
			gk:               gk,
			guardianAddr:     ethcrypto.PubkeyToAddress(gk.PublicKey),
		}
	}

	return gs
}

func mockGuardianSetToGuardianAddrList(gs []*mockGuardian) []eth_common.Address {
	result := make([]eth_common.Address, len(gs))
	for i, g := range gs {
		result[i] = g.guardianAddr
	}
	return result
}

func mockPublicSocket(mockGuardianIndex uint) string {
	return fmt.Sprintf("/tmp/test_guardian_%d_public.socket", mockGuardianIndex)
}

func mockAdminStocket(mockGuardianIndex uint) string {
	return fmt.Sprintf("/tmp/test_guardian_%d_admin.socket", mockGuardianIndex)
}

func mockPublicRpc(mockGuardianIndex uint) string {
	return fmt.Sprintf("127.0.0.1:%d", mockGuardianIndex+LOCAL_RPC_PORTRANGE_START)
}

func mockStatusPort(mockGuardianIndex uint) uint {
	return mockGuardianIndex + LOCAL_STATUS_PORTRANGE_START
}

// mockGuardianRunnable returns a runnable that first sets up a mock guardian an then runs it.
func mockGuardianRunnable(gs []*mockGuardian, mockGuardianIndex uint, obsDb mock.ObservationDb) supervisor.Runnable {
	return func(ctx context.Context) error {
		// Create a sub-context with cancel function that we can pass to G.run.
		ctx, ctxCancel := context.WithCancel(ctx)
		defer ctxCancel()
		logger := supervisor.Logger(ctx)

		// setup db
		dataDir := fmt.Sprintf("/tmp/test_guardian_%d", mockGuardianIndex)
		_ = os.RemoveAll(dataDir) // delete any pre-existing data
		db := db.OpenDb(logger, &dataDir)
		defer db.Close()

		// set environment
		env := common.GoTest

		// setup a mock watcher
		var watcherConfigs = []watchers.WatcherConfig{
			&mock.WatcherConfig{
				NetworkID:        "mock",
				ChainID:          vaa.ChainIDSolana,
				MockObservationC: gs[mockGuardianIndex].MockObservationC,
				MockSetC:         gs[mockGuardianIndex].MockSetC,
				ObservationDb:    obsDb, // TODO(future work) add observation DB to support re-observation request
			},
		}

		// configure p2p
		nodeName := fmt.Sprintf("g-%d", mockGuardianIndex)
		networkID := "/wormhole/localdev"
		zeroPeerId, err := libp2p_peer.IDFromPublicKey(gs[0].p2pKey.GetPublic())
		if err != nil {
			return err
		}
		bootstrapPeers := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic/p2p/%s", LOCAL_P2P_PORTRANGE_START, zeroPeerId.String())
		p2pPort := uint(LOCAL_P2P_PORTRANGE_START + mockGuardianIndex)

		// configure publicRpc
		publicSocketPath := mockPublicSocket(mockGuardianIndex)
		publicRpc := mockPublicRpc(mockGuardianIndex)

		// configure adminservice
		adminSocketPath := mockAdminStocket(mockGuardianIndex)
		rpcMap := make(map[string]string)

		// assemble all the options
		guardianOptions := []*GuardianOption{
			GuardianOptionDatabase(db),
			GuardianOptionWatchers(watcherConfigs, nil),
			GuardianOptionAccountant("", "", false), // effectively disable accountant
			GuardianOptionGovernor(false),           // disable governor
			GuardianOptionP2P(gs[mockGuardianIndex].p2pKey, networkID, bootstrapPeers, nodeName, false, p2pPort, func() string { return "" }),
			GuardianOptionPublicRpcSocket(publicSocketPath, common.GrpcLogDetailFull),
			GuardianOptionPublicrpcTcpService(publicRpc, common.GrpcLogDetailFull),
			GuardianOptionAdminService(adminSocketPath, nil, nil, rpcMap),
			GuardianOptionStatusServer(fmt.Sprintf("[::]:%d", mockStatusPort(mockGuardianIndex))),
		}

		guardianNode := NewGuardianNode(
			env,
			gs[mockGuardianIndex].gk,
			nil,
		)

		if err = supervisor.Run(ctx, "g", guardianNode.Run(ctxCancel, guardianOptions...)); err != nil {
			panic(err)
		}

		<-ctx.Done()

		// cleanup
		// _ = os.RemoveAll(dataDir) // we don't do this for now since this could run before BadgerDB's flush(), causing an error; Meh

		return nil
	}
}

// setupLogsCapture is a helper function for making a zap logger/observer combination for testing that certain logs have been made
func setupLogsCapture() (*zap.Logger, *observer.ObservedLogs) {
	observedCore, logs := observer.New(zap.DebugLevel)
	logger, _ := zap.NewDevelopment(zap.WrapCore(func(c zapcore.Core) zapcore.Core { return zapcore.NewTee(c, observedCore) }))
	return logger, logs
}

func waitForHeartbeatsInLogs(t *testing.T, zapObserver *observer.ObservedLogs, gs []*mockGuardian) {
	// example log entry that we're looking for:
	// 		DEBUG	root.g-2.g.p2p	p2p/p2p.go:465	valid signed heartbeat received	{"value": "node_name:\"g-0\"  timestamp:1685677055425243683  version:\"development\"  guardian_addr:\"0xeF2a03eAec928DD0EEAf35aD31e34d2b53152c07\"  boot_timestamp:1685677040424855922  p2p_node_id:\"\\x00$\\x08\\x01\\x12 \\x97\\xf3\\xbd\\x87\\x13\\x15(\\x1e\\x8b\\x83\\xedǩ\\xfd\\x05A\\x06aTD\\x90p\\xcc\\xdb<\\xddB\\xcfi\\xccވ\"", "from": "12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"}
	// TODO maybe instead of looking at log entries, we could determine this status through prometheus metrics, which might be more stable
	re := regexp.MustCompile("g-[0-9]+")

	for readyCounter := 0; readyCounter < len(gs); {
		// read log messages
		for _, loggedEntry := range zapObserver.FilterMessage("valid signed heartbeat received").All() {
			for _, f := range loggedEntry.Context {
				if f.Key == "value" {
					s, ok := f.Interface.(fmt.Stringer)
					assert.True(t, ok)
					match := re.FindStringSubmatch(s.String())
					assert.NotZero(t, len(match))
					guardianId, err := strconv.Atoi(match[0][2:])
					assert.NoError(t, err)
					assert.True(t, guardianId < len(gs))

					if gs[guardianId].ready == false {
						gs[guardianId].ready = true
						readyCounter++
					}
				}
			}
		}
		time.Sleep(time.Microsecond * 100)
	}
}

func waitForHeartbeatsInMetrics(t *testing.T, ctx context.Context, gs []*mockGuardian) {
	requests := make([]*http.Request, len(gs))
	//logger := supervisor.Logger(ctx)

	// create the prom api clients
	for i := range gs {
		url := fmt.Sprintf("http://localhost:%d/metrics", mockStatusPort(uint(i)))
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		assert.NoError(t, err)
		requests[i] = req
	}

	// query them
	for readyCounter := 0; readyCounter < len(gs); {
		for i, g := range gs {
			if g.ready {
				continue
			}

			resp, err := http.DefaultClient.Do(requests[i])
			if err != nil {
				resp.Body.Close()
				continue
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Bytes()
				if bytes.HasPrefix(line, PROMETHEUS_METRIC_VALID_HEARTBEAT_RECEIVED) {
					res, err := strconv.Atoi(string(bytes.Split(line, []byte(" "))[1])) // split at the space and convert to integer
					assert.NoError(t, err)
					if res > 0 {
						g.ready = true
						readyCounter++
						break
					}
				}
			}

			//logger.Info("node not ready yet", zap.Int("i", i))
		}
		time.Sleep(time.Second * 5)
	}
}

type testCase struct {
	msg *common.MessagePublication // a Wormhole message
	// number of Guardians who will initially observe this message through the mock watcher
	numGuardiansObserve int
	// if true, Guardians will not observe this message in the mock watcher, if they receive a reobservation request for it
	unavailableInReobservation bool
	// if true, the test environment will inject a reobservation request signed by Guardian 1,
	// as if that Guardian had made a manual reobservation request through an admin command
	performManualReobservationRequest bool
	// if true, assert that a VAA eventually exists for this message
	mustReachQuorum bool
	// if true, assert that no VAA exists for this message at the end of the test.
	// Note that it is not guaranteed that this message will never reach quorum because it may reach quorum some time after the test run finishes.
	mustNotReachQuorum bool
}

func randomTime() time.Time {
	return time.Unix(int64(math_rand.Uint32()%1700000000), 0) // nolint // convert time to unix and back to match what is done during serialization/de-serialization
}

var messageSequenceCounter uint64 = 0

func someMessage() *common.MessagePublication {
	messageSequenceCounter++
	return &common.MessagePublication{
		TxHash:           [32]byte{byte(messageSequenceCounter % 8), byte(messageSequenceCounter / 8), 3},
		Timestamp:        randomTime(),
		Nonce:            math_rand.Uint32(), //nolint
		Sequence:         messageSequenceCounter,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   [32]byte{1, 2, 3},
		Payload:          []byte{},
		Unreliable:       false,
	}
}

func makeObsDb(tc []testCase) mock.ObservationDb {
	db := make(map[eth_common.Hash]*common.MessagePublication)
	for _, t := range tc {
		if t.unavailableInReobservation {
			continue
		}
		db[t.msg.TxHash] = t.msg
	}
	return db
}

// #nosec G107 -- it's OK to make http requests with `statusAddr` because `statusAddr` is trusted.
func testStatusServer(ctx context.Context, logger *zap.Logger, statusAddr string) error {
	// Check /readyz
	for {
		url := statusAddr + "/readyz"
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			break
		}
		logger.Info("StatusServer error, waiting 100ms...", zap.String("url", url))
		time.Sleep(time.Millisecond * 100)
	}

	// Check /metrics (prometheus)
	for {
		url := statusAddr + "/metrics"
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			break
		}
		logger.Info("StatusServer error, waiting 100ms...", zap.String("url", url))
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func TestMain(m *testing.M) {
	readiness.NoPanic = true // otherwise we'd panic when running multiple guardians
	os.Exit(m.Run())
}

// TestInvalidWatcherConfig tries to instantiate a guardian with various invlid []watchers.WatcherConfig and asserts that it errors
func TestInvalidWatcherConfig(t *testing.T) {
	// TODO
}

// TestBasicConsensus tests that a set of guardians can form consensus on certain messages and reject certain other messages
func TestBasicConsensus(t *testing.T) {
	const numGuardians = 4 // Quorum will be 3 out of 4 guardians.

	msgZeroEmitter := someMessage()
	msgZeroEmitter.EmitterAddress = vaa.Address{}

	msgGovEmitter := someMessage()
	msgGovEmitter.EmitterAddress = vaa.GovernanceEmitter

	msgWrongEmitterChain := someMessage()
	msgWrongEmitterChain.EmitterChain = vaa.ChainIDEthereum

	// define the test cases to be executed
	testCases := []testCase{
		{ // one malicious Guardian makes an observation + sends a re-observation request; this should not reach quorum
			msg:                        someMessage(),
			numGuardiansObserve:        1,
			mustNotReachQuorum:         true,
			unavailableInReobservation: true,
		},
		{ // message with EmitterAddress == 0 should not reach quorum
			msg:                 msgZeroEmitter,
			numGuardiansObserve: numGuardians,
			mustNotReachQuorum:  true,
		},
		{ // message with Governance emitter should not reach quorum
			msg:                 msgGovEmitter,
			numGuardiansObserve: numGuardians,
			mustNotReachQuorum:  true,
		},
		{ // message with wrong EmitterChain should not reach quorum
			msg:                 msgWrongEmitterChain,
			numGuardiansObserve: numGuardians,
			mustNotReachQuorum:  true,
		},
		{ // vanilla case, where only a quorum of guardians gets the message
			msg:                 someMessage(),
			numGuardiansObserve: numGuardians*2/3 + 1,
			mustReachQuorum:     true,
		},
		{ // No Guardian makes the observation while watching, but we do a manual reobservation request.
			msg:                               someMessage(),
			numGuardiansObserve:               0,
			mustReachQuorum:                   true,
			performManualReobservationRequest: true,
		},
		// TODO add a testcase to test the automatic re-observation requests.
		// Need to refactor various usage of wall time to a mockable time first. E.g. using https://github.com/benbjohnson/clock
	}
	testConsensus(t, testCases, numGuardians)
}

// testConsensus spins up `numGuardians` guardians and runs & verifies the testCases
func testConsensus(t *testing.T, testCases []testCase, numGuardians int) {
	const testTimeout = time.Second * 60
	const guardianSetIndex = 5           // index of the active guardian set (can be anything, just needs to be set to something)
	const vaaCheckGuardianIndex uint = 0 // we will query this guardian's publicrpc for VAAs
	const adminRpcGuardianIndex uint = 0 // we will query this guardian's adminRpc

	// Test's main lifecycle context.
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), testTimeout)
	defer rootCtxCancel()

	zapLogger, zapObserver := setupLogsCapture()

	supervisor.New(rootCtx, zapLogger, func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)

		// create the Guardian Set
		gs := newMockGuardianSet(numGuardians)

		obsDb := makeObsDb(testCases)

		// run the guardians
		for i := 0; i < numGuardians; i++ {
			gRun := mockGuardianRunnable(gs, uint(i), obsDb)
			err := supervisor.Run(ctx, fmt.Sprintf("g-%d", i), gRun)
			assert.NoError(t, err)
		}
		logger.Info("All Guardians initiated.")
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		// Inform them of the Guardian Set
		commonGuardianSet := common.GuardianSet{
			Keys:  mockGuardianSetToGuardianAddrList(gs),
			Index: guardianSetIndex,
		}
		for i, g := range gs {
			logger.Info("Sending guardian set update", zap.Int("guardian_index", i))
			g.MockSetC <- &commonGuardianSet
		}

		// wait for the status server to come online and check that it works
		for i := range gs {
			err := testStatusServer(ctx, logger, fmt.Sprintf("http://127.0.0.1:%d/metrics", mockStatusPort(uint(i))))
			assert.NoError(t, err)
		}

		// Wait for them to connect each other and receive at least one heartbeat.
		// This is necessary because if they have not joined the p2p network yet, gossip messages may get dropped silently.
		assert.True(t, WAIT_FOR_LOGS || WAIT_FOR_METRICS)
		if WAIT_FOR_METRICS {
			waitForHeartbeatsInMetrics(t, ctx, gs)
		}
		if WAIT_FOR_LOGS {
			waitForHeartbeatsInLogs(t, zapObserver, gs)
		}
		logger.Info("All Guardians have received at least one heartbeat.")

		// have them make observations
		for _, testCase := range testCases {
			select {
			case <-ctx.Done():
				return nil
			default:
				// make the first testCase.numGuardiansObserve guardians observe it
				for guardianIndex, g := range gs {
					if guardianIndex >= testCase.numGuardiansObserve {
						break
					}
					msgCopy := *testCase.msg
					logger.Info("requesting mock observation for guardian", msgCopy.ZapFields(zap.Int("guardian_index", guardianIndex))...)
					g.MockObservationC <- &msgCopy
				}
			}
		}

		// Wait for adminrpc to come online
		for zapObserver.FilterMessage("admin server listening on").FilterField(zap.String("path", mockAdminStocket(adminRpcGuardianIndex))).Len() == 0 {
			logger.Info("admin server seems to be offline (according to logs). Waiting 100ms...")
			time.Sleep(time.Microsecond * 100)
		}

		// Send manual re-observation requests
		func() { // put this in own function to use defer
			s := fmt.Sprintf("unix:///%s", mockAdminStocket(vaaCheckGuardianIndex))
			conn, err := grpc.DialContext(ctx, s, grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			c := nodev1.NewNodePrivilegedServiceClient(conn)

			for i, testCase := range testCases {
				if testCase.performManualReobservationRequest {
					// timeout for grpc query
					logger.Info("injecting observation request through admin rpc", zap.Int("test_case", i))
					queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
					_, err = c.SendObservationRequest(queryCtx, &nodev1.SendObservationRequestRequest{
						ObservationRequest: &gossipv1.ObservationRequest{
							ChainId: uint32(testCase.msg.EmitterChain),
							TxHash:  testCase.msg.TxHash[:],
						},
					})
					queryCancel()
					assert.NoError(t, err)
				}
			}
		}()

		// Wait for publicrpc to come online
		for zapObserver.FilterMessage("publicrpc server listening").FilterField(zap.String("addr", mockPublicRpc(vaaCheckGuardianIndex))).Len() == 0 {
			logger.Info("publicrpc seems to be offline (according to logs). Waiting 100ms...")
			time.Sleep(time.Microsecond * 100)
		}

		// check that the VAAs were generated
		logger.Info("Connecting to publicrpc...")
		conn, err := grpc.DialContext(ctx, mockPublicRpc(vaaCheckGuardianIndex), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		defer conn.Close()
		c := publicrpcv1.NewPublicRPCServiceClient(conn)

		gsAddrList := mockGuardianSetToGuardianAddrList(gs)

		// ensure that all test cases have passed
		for i, testCase := range testCases {
			msg := testCase.msg

			logger.Info("Checking result of testcase", zap.Int("test_case", i))

			// poll the API until we get a response without error
			var r *publicrpcv1.GetSignedVAAResponse
			var err error
			for {
				select {
				case <-ctx.Done():
					assert.Fail(t, "timed out")
				default:
					// timeout for grpc query
					logger.Info("attempting to query for VAA", zap.Int("test_case", i))
					queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
					r, err = c.GetSignedVAA(queryCtx, &publicrpcv1.GetSignedVAARequest{
						MessageId: &publicrpcv1.MessageID{
							EmitterChain:   publicrpcv1.ChainID(msg.EmitterChain),
							EmitterAddress: msg.EmitterAddress.String(),
							Sequence:       msg.Sequence,
						},
					})
					queryCancel()
					if err != nil {
						logger.Info("error querying for VAA. Trying agin in 100ms.", zap.Int("test_case", i), zap.Error(err))
					}
				}
				if err == nil && r != nil {
					logger.Info("Received VAA from publicrpc", zap.Int("test_case", i), zap.Binary("vaa_bytes", r.VaaBytes))
					break
				}
				if testCase.mustNotReachQuorum {
					// no need to re-try because we're expecting an error. (and later we'll assert that's indeed an error)
					break
				}
				time.Sleep(time.Millisecond * 100)
			}

			assert.NotEqual(t, testCase.mustNotReachQuorum, testCase.mustReachQuorum) // either or
			if testCase.mustNotReachQuorum {
				assert.EqualError(t, err, "rpc error: code = NotFound desc = requested VAA not found in store")
			} else if testCase.mustReachQuorum {
				assert.NotNil(t, r)
				returnedVaa, err := vaa.Unmarshal(r.VaaBytes)
				assert.NoError(t, err)

				// Check signatures
				err = returnedVaa.Verify(gsAddrList)
				assert.NoError(t, err)

				// Match all the fields
				assert.Equal(t, returnedVaa.Version, uint8(1))
				assert.Equal(t, returnedVaa.GuardianSetIndex, uint32(guardianSetIndex))
				assert.Equal(t, returnedVaa.Timestamp, msg.Timestamp)
				assert.Equal(t, returnedVaa.Nonce, msg.Nonce)
				assert.Equal(t, returnedVaa.Sequence, msg.Sequence)
				assert.Equal(t, returnedVaa.ConsistencyLevel, msg.ConsistencyLevel)
				assert.Equal(t, returnedVaa.EmitterChain, msg.EmitterChain)
				assert.Equal(t, returnedVaa.EmitterAddress, msg.EmitterAddress)
				assert.Equal(t, returnedVaa.Payload, msg.Payload)
			}
		}

		// We're done!
		logger.Info("Tests completed.")

		supervisor.Signal(ctx, supervisor.SignalDone)

		rootCtxCancel()
		return nil
	},
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	assert.NotEqual(t, rootCtx.Err(), context.DeadlineExceeded)
	zapLogger.Info("Test root context cancelled, exiting...")
}
