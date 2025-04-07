package node

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	math_rand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"sync/atomic"

	"github.com/certusone/wormhole/node/pkg/adminrpc"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/mock"
	eth_crypto "github.com/ethereum/go-ethereum/crypto"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2p_peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	eth_common "github.com/ethereum/go-ethereum/common"
)

const LOCAL_RPC_PORTRANGE_START = 10000
const LOCAL_P2P_PORTRANGE_START = 11000
const LOCAL_STATUS_PORTRANGE_START = 12000
const LOCAL_PUBLICWEB_PORTRANGE_START = 13000

var PROMETHEUS_METRIC_VALID_HEARTBEAT_RECEIVED = "wormhole_p2p_broadcast_messages_received_total{type=\"valid_heartbeat\"}"

const WAIT_FOR_LOGS = true
const WAIT_FOR_METRICS = false

// The level at which logs will be written to console; During testing, logs are produced and buffered at Info level, because some tests need to look for certain entries.
var CONSOLE_LOG_LEVEL = zap.InfoLevel

const guardianSetIndex = 5 // index of the active guardian set (can be anything, just needs to be set to something)

var TEST_ID_CTR atomic.Uint32

var logger *zap.Logger

func getTestId() uint {
	return uint(TEST_ID_CTR.Add(1))
}

type mockGuardian struct {
	p2pKey           libp2p_crypto.PrivKey
	MockObservationC chan *common.MessagePublication
	MockSetC         chan *common.GuardianSet
	guardianSigner   guardiansigner.GuardianSigner
	guardianAddr     eth_common.Address
	ready            bool
	config           *guardianConfig
	db               *db.Database
}

type guardianConfig struct {
	publicSocket string
	adminSocket  string
	publicRpc    string
	publicWeb    string
	statusPort   uint
	p2pPort      uint
}

func createGuardianConfig(t testing.TB, testId uint, mockGuardianIndex uint) *guardianConfig {
	t.Helper()
	return &guardianConfig{
		publicSocket: fmt.Sprintf("/tmp/test_guardian_%d_public.socket", mockGuardianIndex+testId*20),
		adminSocket:  fmt.Sprintf("/tmp/test_guardian_%d_admin.socket", mockGuardianIndex+testId*20), // TODO consider using os.CreateTemp("/tmp", "test_guardian_adminXXXXX.socket"),
		publicRpc:    fmt.Sprintf("127.0.0.1:%d", mockGuardianIndex+LOCAL_RPC_PORTRANGE_START+testId*20),
		publicWeb:    fmt.Sprintf("127.0.0.1:%d", mockGuardianIndex+LOCAL_PUBLICWEB_PORTRANGE_START+testId*20),
		statusPort:   mockGuardianIndex + LOCAL_STATUS_PORTRANGE_START + testId*20,
		p2pPort:      mockGuardianIndex + LOCAL_P2P_PORTRANGE_START + testId*20,
	}
}

func newMockGuardianSet(t testing.TB, testId uint, n int) []*mockGuardian {
	t.Helper()
	gs := make([]*mockGuardian, n)

	for i := 0; i < n; i++ {
		// generate guardian signer
		guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
		if err != nil {
			panic(err)
		}

		gs[i] = &mockGuardian{
			p2pKey:           devnet.DeterministicP2PPrivKeyByIndex(int64(i)),
			MockObservationC: make(chan *common.MessagePublication),
			MockSetC:         make(chan *common.GuardianSet),
			guardianSigner:   guardianSigner,
			guardianAddr:     eth_crypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background())),
			config:           createGuardianConfig(t, testId, uint(i)), // #nosec G115 -- Guardian set will never be that large
		}
	}

	return gs
}

func mockGuardianSetToGuardianAddrList(t testing.TB, gs []*mockGuardian) []eth_common.Address {
	t.Helper()
	result := make([]eth_common.Address, len(gs))
	for i, g := range gs {
		result[i] = g.guardianAddr
	}
	return result
}

// mockGuardianRunnable returns a runnable that first sets up a mock guardian an then runs it.
func mockGuardianRunnable(t testing.TB, gs []*mockGuardian, mockGuardianIndex uint, obsDb mock.ObservationDb) supervisor.Runnable {
	t.Helper()
	return func(ctx context.Context) error {
		// Create a sub-context with cancel function that we can pass to G.run.
		ctx, ctxCancel := context.WithCancel(ctx)
		defer ctxCancel()

		// setup db
		db := db.OpenDb(nil, nil)
		defer db.Close()
		gs[mockGuardianIndex].db = db

		// set environment
		env := common.GoTest

		// setup a mock watcher
		var watcherConfigs = []watchers.WatcherConfig{
			&mock.WatcherConfig{
				NetworkID:        "mock",
				ChainID:          vaa.ChainIDSolana,
				MockObservationC: gs[mockGuardianIndex].MockObservationC,
				MockSetC:         gs[mockGuardianIndex].MockSetC,
				ObservationDb:    obsDb,
			},
		}

		// configure p2p
		nodeName := fmt.Sprintf("g-%d", mockGuardianIndex)
		networkID := "/wormhole/localdev"
		zeroPeerId, err := libp2p_peer.IDFromPublicKey(gs[0].p2pKey.GetPublic())
		if err != nil {
			return err
		}
		bootstrapPeers := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic/p2p/%s", gs[0].config.p2pPort, zeroPeerId.String())

		// configure adminservice
		rpcMap := make(map[string]string)

		// We set this to None because we don't want to count these logs when counting the amount of logs generated per message
		publicRpcLogDetail := common.GrpcLogDetailNone

		cfg := gs[mockGuardianIndex].config

		// assemble all the options
		guardianOptions := []*GuardianOption{
			GuardianOptionDatabase(db),
			GuardianOptionWatchers(watcherConfigs, nil),
			GuardianOptionNoAccountant(), // disable accountant
			GuardianOptionGovernor(true, false, ""),
			GuardianOptionGatewayRelayer("", nil), // disable gateway relayer
			GuardianOptionP2P(gs[mockGuardianIndex].p2pKey, networkID, bootstrapPeers, nodeName, false, false, cfg.p2pPort, "", 0, "", "", func() string { return "" }, []string{}, []string{}, []string{}),
			GuardianOptionPublicRpcSocket(cfg.publicSocket, publicRpcLogDetail),
			GuardianOptionPublicrpcTcpService(cfg.publicRpc, publicRpcLogDetail),
			GuardianOptionPublicWeb(cfg.publicWeb, cfg.publicSocket, "", false, ""),
			GuardianOptionAdminService(cfg.adminSocket, nil, nil, rpcMap),
			GuardianOptionStatusServer(fmt.Sprintf("[::]:%d", cfg.statusPort)),
			GuardianOptionProcessor(networkID),
		}

		guardianNode := NewGuardianNode(
			env,
			gs[mockGuardianIndex].guardianSigner,
		)

		if err = supervisor.Run(ctx, "g", guardianNode.Run(ctxCancel, guardianOptions...)); err != nil {
			panic(err)
		}

		<-ctx.Done()
		time.Sleep(time.Second * 1) // Wait 1s for all sorts of things to complete.
		db.Close()                  // close BadgerDb

		return nil
	}
}

// setupLogsCapture is a helper function for making a zap logger/observer combination for testing that certain logs have been made
func setupLogsCapture(t testing.TB, options ...zap.Option) (*zap.Logger, *observer.ObservedLogs, *LogSizeCounter) {
	t.Helper()
	observedCore, observedLogs := observer.New(zap.InfoLevel)
	consoleLogger := zaptest.NewLogger(t, zaptest.Level(CONSOLE_LOG_LEVEL))
	lc := NewLogSizeCounter(zap.InfoLevel)
	parentLogger := zap.New(zapcore.NewTee(observedCore, consoleLogger.Core(), lc.Core()), options...)
	return parentLogger, observedLogs, lc
}

func waitForHeartbeatsInLogs(t testing.TB, zapObserver *observer.ObservedLogs, gs []*mockGuardian) {
	t.Helper()
	// example log entry that we're looking for:
	// 		INFO	root.g-2.g.p2p	p2p/p2p.go:465	valid signed heartbeat received	{"value": "node_name:\"g-0\"  timestamp:1685677055425243683  version:\"development\"  guardian_addr:\"0xeF2a03eAec928DD0EEAf35aD31e34d2b53152c07\"  boot_timestamp:1685677040424855922  p2p_node_id:\"\\x00$\\x08\\x01\\x12 \\x97\\xf3\\xbd\\x87\\x13\\x15(\\x1e\\x8b\\x83\\xedǩ\\xfd\\x05A\\x06aTD\\x90p\\xcc\\xdb<\\xddB\\xcfi\\xccވ\"", "from": "12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"}
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
		time.Sleep(time.Millisecond)
	}
}

// waitForPromMetricGte waits until prometheus metric `metric` >= `min` on all guardians in `gs`.
// WARNING: Currently, there is only a global registry for all prometheus metrics, leading to all guardian nodes writing to the same one.
//
//	As long as this is the case, you probably don't want to use this function.
func waitForPromMetricGte(t testing.TB, ctx context.Context, gs []*mockGuardian, metric string, minimum int) {
	t.Helper()
	metricBytes := []byte(metric)
	requests := make([]*http.Request, len(gs))
	readyFlags := make([]bool, len(gs))

	// create the prom api clients
	for i := range gs {
		url := fmt.Sprintf("http://localhost:%d/metrics", gs[i].config.statusPort)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		assert.NoError(t, err)
		requests[i] = req
	}

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// query them
	for readyCounter := 0; readyCounter < len(gs); {
		for i := range gs {
			if readyFlags[i] {
				continue
			}

			ready := func() bool { // use anonymous function to have proper scope for the defer
				resp, err := httpClient.Do(requests[i])
				if err != nil {
					return false
				}
				defer io.Copy(io.Discard, resp.Body) //nolint:errcheck //we don't care about the error
				defer resp.Body.Close()

				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := scanner.Bytes()
					if bytes.HasPrefix(line, metricBytes) {
						res, err := strconv.Atoi(string(bytes.Split(line, []byte(" "))[1])) // split at the space and convert to integer
						assert.NoError(t, err)
						if res >= minimum {
							return true
						}
					}
				}
				return false
			}()

			if ready {
				readyFlags[i] = true
				readyCounter++
			}
		}
		time.Sleep(time.Second * 5) // TODO
	}
}

// waitForVaa polls the publicRpc service every 5ms until there is a response.
func waitForVaa(t testing.TB, ctx context.Context, c publicrpcv1.PublicRPCServiceClient, msgId *publicrpcv1.MessageID, mustNotReachQuorum bool) (*publicrpcv1.GetSignedVAAResponse, error) {
	t.Helper()
	var r *publicrpcv1.GetSignedVAAResponse
	var err error

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("context canceled")
		default:
			queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
			r, err = c.GetSignedVAA(queryCtx, &publicrpcv1.GetSignedVAARequest{MessageId: msgId})
			queryCancel()
		}
		if err == nil && r != nil {
			// success
			return r, err
		}
		if mustNotReachQuorum {
			// no need to re-try because we're expecting an error.
			return r, err
		}
		time.Sleep(time.Millisecond * 10)
	}
}

type testCase struct {
	msg    *common.MessagePublication // a Wormhole message
	govMsg *nodev1.GovernanceMessage  // protobuf representation of msg as governance message, if applicable.
	// number of Guardians who will initially observe this message through the mock watcher
	numGuardiansObserve int
	// number of Guardians where the governance message will be injected through the adminrpc
	numGuardiansInjectGov int
	// if true, Guardians will not observe this message in the mock watcher, if they receive a reobservation request for it
	unavailableInReobservation bool
	// if true, the test environment will inject a reobservation request signed by Guardian 1,
	// as if that Guardian had made a manual reobservation request through an admin command
	performManualReobservationRequest bool
	// if true, we will put the VAA into each guardian's DB
	prePopulateVAA bool
	// if true, assert that a VAA eventually exists for this message
	mustReachQuorum bool
	// if true, assert that no VAA exists for this message at the end of the test.
	// Note that it is not guaranteed that this message will never reach quorum because it may reach quorum some time after the test run finishes.
	mustNotReachQuorum bool
}

func randomTime() time.Time {
	return time.Unix(int64(math_rand.Uint32()%1700000000), 0) // nolint // convert time to unix and back to match what is done during serialization/de-serialization
}

var someMsgSequenceCounter uint64 = 0
var someMsgEmitter vaa.Address = [32]byte{1, 2, 3}
var someMsgEmitterChain vaa.ChainID = vaa.ChainIDSolana

func someMessage() *common.MessagePublication {
	someMsgSequenceCounter++
	txID := [32]byte{byte(someMsgSequenceCounter % 8), byte(someMsgSequenceCounter / 8), 3}
	return &common.MessagePublication{
		TxID:             txID[:],
		Timestamp:        randomTime(),
		Nonce:            math_rand.Uint32(), //nolint
		Sequence:         someMsgSequenceCounter,
		ConsistencyLevel: 1,
		EmitterChain:     someMsgEmitterChain,
		EmitterAddress:   someMsgEmitter,
		Payload:          []byte{},
		Unreliable:       false,
	}
}

var tokenBridgeSequenceCounter uint64 = 0

// governedMsg creates a token bridge message that will be in-scope for the governor module.
// The transfer is of wrapped-SOL from Solana to Ethereum.
// If shouldBeDelayed == true, then the amount will be set to 1_000_000_000_000 wSOL which should exceed the governor limit.
func governedMsg(shouldBeDelayed bool) *common.MessagePublication {

	// buildMockTransferPayloadBytes is copied from governor_test.go.
	buildMockTransferPayloadBytes := func(
		tokenChainID vaa.ChainID,
		tokenAddrStr string,
		toChainID vaa.ChainID,
		toAddrStr string,
		amtFloat float64,
	) []byte {
		bytes := make([]byte, 101)
		bytes[0] = 1 // tb payload type

		amtBigFloat := big.NewFloat(amtFloat)
		amtBigFloat = amtBigFloat.Mul(amtBigFloat, big.NewFloat(100000000))
		amount, _ := amtBigFloat.Int(nil)
		amtBytes := amount.Bytes()
		if len(amtBytes) > 32 {
			panic("amount will not fit in 32 bytes!")
		}
		copy(bytes[33-len(amtBytes):33], amtBytes)

		tokenAddr, _ := vaa.StringToAddress(tokenAddrStr)
		copy(bytes[33:65], tokenAddr.Bytes())
		binary.BigEndian.PutUint16(bytes[65:67], uint16(tokenChainID))
		toAddr, _ := vaa.StringToAddress(toAddrStr)
		copy(bytes[67:99], toAddr.Bytes())
		binary.BigEndian.PutUint16(bytes[99:101], uint16(toChainID))
		return bytes
	}

	var amount float64 = 0.0001
	if shouldBeDelayed {
		amount = 1_000_000_000_000
	}

	tokenAddrStr := "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001" // #nosec G101 -- Address for testing
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"                          // whatever
	payloadBytes := buildMockTransferPayloadBytes(
		vaa.ChainIDSolana,
		tokenAddrStr,
		vaa.ChainIDEthereum,
		toAddrStr,
		amount, // very large number to exceed governor limit
	)

	tokenBridgeSequenceCounter++
	txID := [32]byte{byte(tokenBridgeSequenceCounter % 8), byte(tokenBridgeSequenceCounter / 8), 3, 1, 10, 76}
	return &common.MessagePublication{
		TxID:             txID[:],
		Timestamp:        randomTime(),
		Nonce:            math_rand.Uint32(), //nolint
		Sequence:         tokenBridgeSequenceCounter,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   vaa.Address(sdk.KnownTokenbridgeEmitters[vaa.ChainIDSolana]),
		Payload:          payloadBytes,
		Unreliable:       false,
	}
}

func makeObsDb(tc []testCase) mock.ObservationDb {
	db := make(map[eth_common.Hash]*common.MessagePublication)
	for _, t := range tc {
		if t.unavailableInReobservation {
			continue
		}
		db[eth_common.BytesToHash(t.msg.TxID)] = t.msg
	}
	return db
}

// waitForStatusServer queries the /readyz and /metrics endpoints at `statusAddr` every 100ms until they are online.
// #nosec G107 -- it's OK to make http requests with `statusAddr` because `statusAddr` is trusted.
func waitForStatusServer(ctx context.Context, logger *zap.Logger, statusAddr string) error {
	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// Check /readyz
	for {
		url := statusAddr + "/readyz"
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Info("StatusServer error, waiting 100ms...", zap.String("url", url))
			time.Sleep(time.Millisecond * 100)
			continue // try again
		}
		// success, we're done
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
		break
	}

	// Check /metrics (prometheus)
	for {
		url := statusAddr + "/metrics"
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Info("StatusServer error, waiting 100ms...", zap.String("url", url))
			time.Sleep(time.Millisecond * 100)
			continue // try again
		}
		// success, we're done
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
		break
	}
	return nil
}

func TestMain(m *testing.M) {
	logger, _ = zap.NewDevelopment()
	readiness.NoPanic = true // otherwise we'd panic when running multiple guardians
	retVal := m.Run()
	logger.Info("test exiting", zap.Int("retVal", retVal))
	os.Exit(retVal)
}

func createGovernanceMsgAndVaa(t testing.TB) (*common.MessagePublication, *nodev1.GovernanceMessage) {
	t.Helper()
	msgGov := someMessage()
	msgGov.EmitterAddress = vaa.GovernanceEmitter
	msgGov.EmitterChain = vaa.GovernanceChain

	govMsg := &nodev1.GovernanceMessage{
		Sequence: msgGov.Sequence,
		Nonce:    msgGov.Nonce,
		Payload: &nodev1.GovernanceMessage_GuardianSet{
			GuardianSet: &nodev1.GuardianSetUpdate{
				Guardians: []*nodev1.GuardianSetUpdate_Guardian{
					{
						Pubkey: "0x187727CdD17C8142FE9b29A066F577548423aF0e",
						Name:   "P2P Validator",
					},
				},
			},
		},
	}
	govVaa, err := adminrpc.GovMsgToVaa(govMsg, guardianSetIndex, msgGov.Timestamp)
	require.NoError(t, err)
	msgGov.Payload = govVaa.Payload
	msgGov.ConsistencyLevel = govVaa.ConsistencyLevel

	return msgGov, govMsg
}

// TestConsensus tests that a set of guardians can form consensus on certain messages and reject certain other messages
func TestConsensus(t *testing.T) {
	// adjust processor time intervals to make tests pass faster
	processor.FirstRetryMinWait = time.Second * 3
	processor.CleanupInterval = time.Second * 1

	const numGuardians = 4 // Quorum will be 3 out of 4 guardians.

	msgZeroEmitter := someMessage()
	msgZeroEmitter.EmitterAddress = vaa.Address{}

	msgGovEmitter := someMessage()
	msgGovEmitter.EmitterAddress = vaa.GovernanceEmitter

	msgGov, msgGovProto := createGovernanceMsgAndVaa(t)

	msgWrongEmitterChain := someMessage()
	msgWrongEmitterChain.EmitterChain = vaa.ChainIDEthereum

	// define the test cases to be executed
	// The ones with mustNotReachQuorum=true should be defined first to give them more time to execute.
	testCases := []testCase{
		{
			// Only two Guardian gets the message, but one already has it in the local database.
			// Hence the first Guardian (index 0) should not make an automatic re-observation request
			// We currently don't explicitly verify the non-existence of the re-observation request, but can see it through the code coverage
			msg:                 someMessage(),
			numGuardiansObserve: 2,
			mustReachQuorum:     true,
			prePopulateVAA:      true,
		},
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
		{ // Message covered by Governor that should be delayed 24h and hence not reach quorum within this test
			msg:                 governedMsg(true),
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
		{ // Only one Guardian makes the observation while watching and needs to do an automatic re-observation request.
			msg:                 someMessage(),
			numGuardiansObserve: 1,
			mustReachQuorum:     true,
		},
		{ // Message covered by Governor that should pass immediately
			msg:                 governedMsg(false),
			numGuardiansObserve: numGuardians,
			mustReachQuorum:     true,
		},
		{ // Injected governance message
			msg:                   msgGov,
			govMsg:                msgGovProto,
			numGuardiansObserve:   0,
			numGuardiansInjectGov: numGuardians,
			mustReachQuorum:       true,
		},
		// TODO add a testcase to test the automatic re-observation requests.
		// Need to refactor various usage of wall time to a mockable time first. E.g. using https://github.com/benbjohnson/clock
	}
	runConsensusTests(t, testCases, numGuardians)
}

// runConsensusTests spins up `numGuardians` guardians and runs & verifies the testCases
func runConsensusTests(t *testing.T, testCases []testCase, numGuardians int) {
	const testTimeout = time.Second * 30
	const vaaCheckGuardianIndex uint = 0 // we will query this guardian's publicrpc for VAAs
	const adminRpcGuardianIndex uint = 0 // we will query this guardian's adminRpc
	testId := getTestId()

	// Test's main lifecycle context.
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), testTimeout)
	defer rootCtxCancel()

	zapLogger, zapObserver, _ := setupLogsCapture(t)

	supervisor.New(rootCtx, zapLogger, func(ctx context.Context) error {
		// create the Guardian Set
		gs := newMockGuardianSet(t, testId, numGuardians)

		obsDb := makeObsDb(testCases)

		// run the guardians
		for i := 0; i < numGuardians; i++ {
			gRun := mockGuardianRunnable(t, gs, uint(i), obsDb) // #nosec G115 -- Guardian set will never be that large
			err := supervisor.Run(ctx, fmt.Sprintf("g-%d", i), gRun)
			if i == 0 && numGuardians > 1 {
				time.Sleep(time.Second) // give the bootstrap guardian some time to start up
			}
			assert.NoError(t, err)
		}
		logger.Info("All Guardians initiated.")
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		// Inform them of the Guardian Set
		commonGuardianSet := *common.NewGuardianSet(
			mockGuardianSetToGuardianAddrList(t, gs),
			guardianSetIndex,
		)
		for i, g := range gs {
			logger.Info("Sending guardian set update", zap.Int("guardian_index", i))
			g.MockSetC <- &commonGuardianSet
		}

		// wait for the status server to come online and check that it works
		for _, g := range gs {
			err := waitForStatusServer(ctx, logger, fmt.Sprintf("http://127.0.0.1:%d/metrics", g.config.statusPort))
			assert.NoError(t, err)
		}

		// pre-populate VAAs
		for _, testCase := range testCases {
			if testCase.prePopulateVAA {
				v := testCase.msg.CreateVAA(guardianSetIndex)
				v.Signatures = []*vaa.Signature{{Index: 0}}
				err := gs[0].db.StoreSignedVAA(v)
				assert.NoError(t, err)
			}
		}

		// Wait for them to connect each other and receive at least one heartbeat.
		// This is necessary because if they have not joined the p2p network yet, gossip messages may get dropped silently.
		assert.True(t, WAIT_FOR_LOGS || WAIT_FOR_METRICS)
		assert.False(t, WAIT_FOR_LOGS && WAIT_FOR_METRICS) // can't do both, because they both write to gs[].ready
		if WAIT_FOR_METRICS {
			waitForPromMetricGte(t, ctx, gs, PROMETHEUS_METRIC_VALID_HEARTBEAT_RECEIVED, 1)
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

		// Do adminrpc stuff: Send manual re-observation requests and perform governance msg injections
		func() { // put this in own function to use defer
			// Wait for adminrpc to come online
			adminCs := make([]nodev1.NodePrivilegedServiceClient, numGuardians)
			for i := 0; i < numGuardians; i++ {
				for zapObserver.FilterMessage("admin server listening on").FilterField(zap.String("path", gs[i].config.adminSocket)).Len() == 0 {
					logger.Info("admin server seems to be offline (according to logs). Waiting 100ms...")
					time.Sleep(time.Microsecond * 100)
				}

				s := fmt.Sprintf("unix:///%s", gs[i].config.adminSocket)
				conn, err := grpc.DialContext(ctx, s, grpc.WithTransportCredentials(insecure.NewCredentials()))
				require.NoError(t, err)
				defer conn.Close()
				adminCs[i] = nodev1.NewNodePrivilegedServiceClient(conn)
			}

			for i, testCase := range testCases {
				if testCase.performManualReobservationRequest {
					logger.Info("injecting observation request through admin rpc", zap.Int("test_case", i))
					queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
					_, err := adminCs[adminRpcGuardianIndex].SendObservationRequest(queryCtx, &nodev1.SendObservationRequestRequest{
						ObservationRequest: &gossipv1.ObservationRequest{
							ChainId: uint32(testCase.msg.EmitterChain),
							TxHash:  testCase.msg.TxID,
						},
					})
					queryCancel()
					assert.NoError(t, err)
				}

				for j := 0; j < testCase.numGuardiansInjectGov; j++ {
					require.NotNil(t, testCase.govMsg)
					logger.Info("injecting message through admin rpc", zap.Int("test_case", i), zap.Int("guardian", j))
					queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
					_, err := adminCs[j].InjectGovernanceVAA(queryCtx, &nodev1.InjectGovernanceVAARequest{
						CurrentSetIndex: guardianSetIndex,
						Messages:        []*nodev1.GovernanceMessage{testCase.govMsg},
						Timestamp:       uint32(testCase.msg.Timestamp.Unix()), // #nosec G115 -- This conversion is safe until year 2106
					})
					queryCancel()
					assert.NoError(t, err)
				}
			}
		}()

		// Wait for publicrpc to come online
		for zapObserver.FilterMessage("publicrpc server listening").FilterField(zap.String("addr", gs[vaaCheckGuardianIndex].config.publicRpc)).Len() == 0 {
			logger.Info("publicrpc seems to be offline (according to logs). Waiting 100ms...")
			time.Sleep(time.Microsecond * 100)
		}

		// check that the VAAs were generated
		logger.Info("Connecting to publicrpc...")
		conn, err := grpc.DialContext(ctx, gs[vaaCheckGuardianIndex].config.publicRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		defer conn.Close()
		c := publicrpcv1.NewPublicRPCServiceClient(conn)

		gsAddrList := mockGuardianSetToGuardianAddrList(t, gs)

		// ensure that all test cases have passed
		for i, testCase := range testCases {
			msg := testCase.msg

			logger.Info("Checking result of testcase", zap.Int("test_case", i))

			// poll the API until we get a response without error
			msgId := &publicrpcv1.MessageID{
				EmitterChain:   publicrpcv1.ChainID(msg.EmitterChain),
				EmitterAddress: msg.EmitterAddress.String(),
				Sequence:       msg.Sequence,
			}
			r, err := waitForVaa(t, ctx, c, msgId, testCase.mustNotReachQuorum)

			assert.NotEqual(t, testCase.mustNotReachQuorum, testCase.mustReachQuorum) // either or
			if testCase.mustNotReachQuorum {
				assert.EqualError(t, err, "rpc error: code = NotFound desc = requested VAA not found in store")
			} else if testCase.mustReachQuorum {
				require.NotNil(t, r)
				returnedVaa, err := vaa.Unmarshal(r.VaaBytes)
				assert.NoError(t, err)

				// Check signatures
				if !testCase.prePopulateVAA { // if the VAA is pre-populated with a dummy, then this is expected to fail
					err = returnedVaa.Verify(gsAddrList)
					assert.NoError(t, err)
				}

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

	// There are some things that happen outside of the supervisor context and are a bit racey when the root context
	// is shutdown. Namely some pkg/db bits, metrics sinks, and p2p logging. This gives them time to shutdown so the
	// tests are happy.
	zapLogger.Info("Test root context cancelled, waiting for everything to shut down properly...")
	time.Sleep(time.Millisecond * 50)
}

type testCaseGuardianConfig struct {
	name string
	opts []*GuardianOption
	err  string
}

// TestWatcherConfigs tries to instantiate a guardian with various invalid []watchers.WatcherConfig and asserts that it errors
func TestWatcherConfigs(t *testing.T) {
	tc := []testCaseGuardianConfig{
		{
			name: "no error",
			opts: []*GuardianOption{
				GuardianOptionWatchers([]watchers.WatcherConfig{
					&mock.WatcherConfig{
						NetworkID: "mock1",
						ChainID:   vaa.ChainIDSolana,
					},
					&mock.WatcherConfig{
						NetworkID:           "mock2",
						ChainID:             vaa.ChainIDEthereum,
						L1FinalizerRequired: "mock1",
					},
				}, nil),
			},
			err: "",
		},
		{
			name: "watcher-NetworkID-collision",
			opts: []*GuardianOption{
				GuardianOptionWatchers([]watchers.WatcherConfig{
					&mock.WatcherConfig{
						NetworkID: "mock",
						ChainID:   vaa.ChainIDSolana,
					},
					&mock.WatcherConfig{
						NetworkID: "mock",
						ChainID:   vaa.ChainIDSolana,
					},
				}, nil),
			},
			err: "NetworkID already configured: mock",
		},
		{
			name: "watcher-noL1",
			opts: []*GuardianOption{
				GuardianOptionWatchers([]watchers.WatcherConfig{
					&mock.WatcherConfig{
						NetworkID:           "mock",
						ChainID:             vaa.ChainIDSolana,
						L1FinalizerRequired: "something-that-does-not-exist",
					},
				}, nil),
			},
			err: "L1finalizer does not exist. Please check the order of the watcher configurations in watcherConfigs.",
		},
	}
	runGuardianConfigTests(t, tc)
}

func TestGuardianConfigs(t *testing.T) {
	tc := []testCaseGuardianConfig{
		{
			name: "unfulfilled-dependency",
			opts: []*GuardianOption{
				GuardianOptionAccountant(
					"",    // websocket
					"",    // contract
					false, // enforcing
					nil,   // wormchainConn
					"",    // nttContract
					nil,   // nttWormchainConn
				),
			},
			err: "Check the order of your options.",
		},
		{
			name: "double-configuration",
			opts: []*GuardianOption{
				GuardianOptionDatabase(nil),
				GuardianOptionDatabase(nil),
			},
			err: "Component db is already configured and cannot be configured a second time",
		},
	}
	runGuardianConfigTests(t, tc)
}

func runGuardianConfigTests(t *testing.T, testCases []testCaseGuardianConfig) {
	for _, tc := range testCases {
		// because we're only instantiating the guardians and kill them right after they started running, 2s should be plenty of time
		const testTimeout = time.Second * 2

		func() {
			// Test's main lifecycle context.
			rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), testTimeout)
			defer rootCtxCancel()

			// we need to catch a zap.Logger.Fatal() here.
			// By default zap.Logger.Fatal() will os.Exit(1), which we can't catch.
			// We modify zap's behavior to instead assert that the error is the one we're looking for and then panic
			// The panic will be subsequently caught by the supervisor
			fatalHook := make(fatalHook)
			defer close(fatalHook)
			zapLogger, zapObserver, _ := setupLogsCapture(t, zap.WithFatalHook(fatalHook))

			supervisor.New(rootCtx, zapLogger, func(ctx context.Context) error {
				// Create a sub-context with cancel function that we can pass to G.run.
				ctx, ctxCancel := context.WithCancel(ctx)
				defer ctxCancel()

				if err := supervisor.Run(ctx, tc.name, NewGuardianNode(common.GoTest, nil).Run(ctxCancel, tc.opts...)); err != nil {
					panic(err)
				}

				supervisor.Signal(ctx, supervisor.SignalHealthy)

				// wait for all options to get applied
				// If we were expecting an error, we should never get past this point.
				for len(zapObserver.FilterMessage("GuardianNode initialization done.").All()) == 0 {
					time.Sleep(time.Millisecond * 10)
				}

				// Test done.
				logger.Info("Test done.")
				supervisor.Signal(ctx, supervisor.SignalDone)
				rootCtxCancel()

				return nil
			})

			select {
			case r := <-fatalHook:
				if tc.err == "" {
					assert.Equal(t, tc.err, r)
				}
				assert.Contains(t, r, tc.err)
				rootCtxCancel()
			case <-rootCtx.Done():
				assert.NotEqual(t, rootCtx.Err(), context.DeadlineExceeded)
				assert.Equal(t, tc.err, "") // we only want to end up here if we did not expect an error.
			}

			// There is a race condition where the logger can get destroyed before the supervisor finishes logging on exit. Wait for the last log message from the supervisor.
			count := 0
			for len(zapObserver.FilterMessage("supervisor exited").All()) == 0 {
				time.Sleep(time.Millisecond * 10)
				count++
				assert.Greater(t, 100, count)
			}
		}()
	}
}

// fatalHook catches zap.Logger.Fatal(), sends them to triggerC, and then panics.
type fatalHook chan string

func (c fatalHook) OnWrite(ce *zapcore.CheckedEntry, fields []zapcore.Field) {
	// construct message, which will be the main log message, followed by all errors
	var sb strings.Builder

	sb.WriteString(ce.Message)

	for _, f := range fields {
		err, ok := f.Interface.(error)
		if ok {
			sb.WriteString(", error:")
			sb.WriteString(err.Error())
		}
	}

	c <- sb.String()
	panic(ce.Message)
}

func signingMsgs(n int) [][]byte {
	msgs := make([][]byte, n)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = eth_crypto.Keccak256Hash([]byte{byte(i)}).Bytes()
	}
	return msgs
}

func signMsgsP2p(pk libp2p_crypto.PrivKey, msgs [][]byte) [][]byte {
	n := len(msgs)
	signatures := make([][]byte, n)
	// Ed25519.Sign
	for i := 0; i < n; i++ {
		sig, err := pk.Sign(msgs[i])
		if err != nil {
			panic(err)
		}
		signatures[i] = sig
	}
	return signatures
}

func signMsgsEth(pk *ecdsa.PrivateKey, msgs [][]byte) [][]byte {
	n := len(msgs)
	signatures := make([][]byte, n)
	// Ed25519.Sign
	for i := 0; i < n; i++ {
		sig, err := eth_crypto.Sign(msgs[i], pk)
		if err != nil {
			panic(err)
		}
		signatures[i] = sig
	}
	return signatures
}

func BenchmarkCrypto(b *testing.B) {
	b.Run("libp2p (Ed25519)", func(b *testing.B) {

		p2pKey := devnet.DeterministicP2PPrivKeyByIndex(1)

		b.Run("sign", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			b.ResetTimer()
			signMsgsP2p(p2pKey, msgs)
		})

		b.Run("verify", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			signatures := signMsgsP2p(p2pKey, msgs)
			b.ResetTimer()

			// Ed25519.Verify
			for i := 0; i < b.N; i++ {
				ok, err := p2pKey.GetPublic().Verify(msgs[i], signatures[i])
				assert.NoError(b, err)
				assert.True(b, ok)
			}
		})
	})

	/*
		RSA is an option for libp2p.
		In an optimized RSA implementation, signature verification in RSA can be faster than with elliptic curves, while signature generation is always slower.
		Since libp2p is verification-heavy, this might overall still be a faster option.
		This benchmarks show that the libp2p RSA sigverify seems to be unoptimized and is actually slower than ED25519, as of go-libp2p v0.29.2:
			libp2p_(Ed25519)/sign-64		36178 ns/op
			libp2p_(Ed25519)/verify-64		85326 ns/op
			libp2p_(RSA)/sign-64		  2226550 ns/op
			libp2p_(RSA)/verify-64		   327945 ns/op
	*/
	b.Run("libp2p (RSA)", func(b *testing.B) {

		r := math_rand.New(math_rand.NewSource(0)) //#nosec G404 testnet / devnet keys are public knowledge
		p2pKey, _, err := libp2p_crypto.GenerateKeyPairWithReader(libp2p_crypto.RSA, 2048, r)
		if err != nil {
			panic(err)
		}

		b.Run("sign", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			b.ResetTimer()
			signMsgsP2p(p2pKey, msgs)
		})

		b.Run("verify", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			signatures := signMsgsP2p(p2pKey, msgs)
			b.ResetTimer()

			// RSA.Verify
			for i := 0; i < b.N; i++ {
				ok, err := p2pKey.GetPublic().Verify(msgs[i], signatures[i])
				assert.NoError(b, err)
				assert.True(b, ok)
			}
		})
	})

	b.Run("eth_crypto (secp256k1)", func(b *testing.B) {

		gk := devnet.InsecureDeterministicEcdsaKeyByIndex(eth_crypto.S256(), 0)

		b.Run("sign", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			b.ResetTimer()
			signMsgsEth(gk, msgs)
		})

		b.Run("verify", func(b *testing.B) {
			msgs := signingMsgs(b.N)
			signatures := signMsgsEth(gk, msgs)
			b.ResetTimer()

			// Ed25519.Verify
			for i := 0; i < b.N; i++ {
				_, err := eth_crypto.Ecrecover(msgs[i], signatures[i])
				assert.NoError(b, err)
			}
		})
	})
}

// How to run:
//
//	go test -v -bench ^BenchmarkConsensus -benchtime=1x -count 1 -run ^$ > bench.log; tail bench.log
func BenchmarkConsensus(b *testing.B) {
	require.Equal(b, b.N, 1)
	//CONSOLE_LOG_LEVEL = zap.DebugLevel
	//CONSOLE_LOG_LEVEL = zap.InfoLevel
	CONSOLE_LOG_LEVEL = zap.WarnLevel
	runConsensusBenchmark(b, "1", 19, 1000, 50) // ~7.5s
	//runConsensusBenchmark(b, "1", 19, 1000, 5) // ~10s
	//runConsensusBenchmark(b, "1", 19, 1000, 1) // ~13s
}

func runConsensusBenchmark(t *testing.B, name string, numGuardians int, numMessages uint64, maxPendingObs int) {
	const vaaCheckGuardianIndex = 1 // we will query this Guardian for VAAs.

	t.Run(name, func(t *testing.B) {
		require.Equal(t, t.N, 1)
		testId := getTestId()
		msgSeqStart := someMsgSequenceCounter

		const testTimeout = time.Minute * 2
		const guardianSetIndex = 5 // index of the active guardian set (can be anything, just needs to be set to something)

		// Test's main lifecycle context.
		rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), testTimeout)
		defer rootCtxCancel()

		zapLogger, zapObserver, setupLogsCapture := setupLogsCapture(t)

		supervisor.New(rootCtx, zapLogger, func(ctx context.Context) error {
			// create the Guardian Set
			gs := newMockGuardianSet(t, testId, numGuardians)

			var obsDb mock.ObservationDb = nil // TODO

			// run the guardians
			for i := 0; i < numGuardians; i++ {
				gRun := mockGuardianRunnable(t, gs, uint(i), obsDb) // #nosec G115 -- This conversion is safe based on the constant used above
				err := supervisor.Run(ctx, fmt.Sprintf("g-%d", i), gRun)
				if i == 0 && numGuardians > 1 {
					time.Sleep(time.Second) // give the bootstrap guardian some time to start up
				}
				assert.NoError(t, err)
			}
			logger.Info("All Guardians initiated.")
			supervisor.Signal(ctx, supervisor.SignalHealthy)

			// Inform them of the Guardian Set
			commonGuardianSet := *common.NewGuardianSet(
				mockGuardianSetToGuardianAddrList(t, gs),
				guardianSetIndex,
			)
			for i, g := range gs {
				logger.Info("Sending guardian set update", zap.Int("guardian_index", i))
				g.MockSetC <- &commonGuardianSet
			}

			// wait for the status server to come online and check that it works
			for _, g := range gs {
				err := waitForStatusServer(ctx, logger, fmt.Sprintf("http://127.0.0.1:%d/metrics", g.config.statusPort))
				assert.NoError(t, err)
			}

			// Wait for them to connect each other and receive at least one heartbeat.
			// This is necessary because if they have not joined the p2p network yet, gossip messages may get dropped silently.
			assert.True(t, WAIT_FOR_LOGS || WAIT_FOR_METRICS)
			if WAIT_FOR_METRICS {
				waitForPromMetricGte(t, ctx, gs, PROMETHEUS_METRIC_VALID_HEARTBEAT_RECEIVED, 1)
			}
			if WAIT_FOR_LOGS {
				waitForHeartbeatsInLogs(t, zapObserver, gs)
			}
			logger.Info("All Guardians have received at least one heartbeat.")

			// Wait for publicrpc to come online.
			for zapObserver.FilterMessage("publicrpc server listening").FilterField(zap.String("addr", gs[vaaCheckGuardianIndex].config.publicRpc)).Len() == 0 {
				logger.Info("publicrpc seems to be offline (according to logs). Waiting 100ms...")
				time.Sleep(time.Microsecond * 100)
			}
			// now that it's online, connect to publicrpc of guardian-0
			conn, err := grpc.DialContext(ctx, gs[vaaCheckGuardianIndex].config.publicRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()
			c := publicrpcv1.NewPublicRPCServiceClient(conn)

			logger.Info("-----------Beginning benchmark-----------")
			setupLogsCapture.Reset()
			t.ResetTimer()

			// nextObsReadyC ensures that there are not more than `maxPendingObs` observations pending at any given point in time.
			nextObsReadyC := make(chan struct{}, maxPendingObs)
			for j := 0; j < maxPendingObs; j++ {
				nextObsReadyC <- struct{}{}
			}

			go func() {
				// feed observations to nodes
				for i := uint64(0); i < numMessages; i++ {
					select {
					case <-ctx.Done():
						return
					case <-nextObsReadyC:
						msg := someMessage()
						for _, g := range gs {
							msgCopy := *msg
							g.MockObservationC <- &msgCopy
						}
					}
				}
			}()

			// check that the VAAs were generated
			for i := uint64(0); i < numMessages; i++ {
				msgId := &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(someMsgEmitterChain),
					EmitterAddress: someMsgEmitter.String(),
					Sequence:       msgSeqStart + uint64(i+1),
				}
				// a VAA should not take longer than 10s to be produced, no matter what.
				waitCtx, cancelFunc := context.WithTimeout(ctx, time.Second*10)
				_, err := waitForVaa(t, waitCtx, c, msgId, false)
				cancelFunc()
				assert.NoError(t, err)
				if err != nil {
					// early cancel the benchmark
					rootCtxCancel()
				}
				nextObsReadyC <- struct{}{}
			}

			// We're done!
			logger.Info("Tests completed.")
			t.StopTimer()
			logsize := setupLogsCapture.Reset()
			// normalize
			logsize = logsize / uint64(numMessages) / uint64(numGuardians) // #nosec G115 -- These conversions are safe based on the constants used above
			logger.Warn("benchmarkConsensus: logsize report", zap.Uint64("logbytes_per_msg", logsize))
			supervisor.Signal(ctx, supervisor.SignalDone)
			rootCtxCancel()
			return nil
		},
			supervisor.WithPropagatePanic)

		<-rootCtx.Done()
		assert.NotEqual(t, rootCtx.Err(), context.DeadlineExceeded)
		zapLogger.Info("Test root context cancelled, exiting...")

		// wait for everything to shut down gracefully
		//time.Sleep(time.Second * 11) // 11s is needed to gracefully shutdown libp2p, but since switching to dedicated ports per `testId`, this is no longer necessary
		time.Sleep(time.Second * 1) // 1s is needed to gracefully shutdown BadgerDB
	})
}
