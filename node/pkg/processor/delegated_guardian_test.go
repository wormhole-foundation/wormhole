//go:build delegated_guardian_ci

package processor

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eth_crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	guardian0 = "befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe"
	guardian1 = "88d7d8b32a9105d228100e72dffe2fae0705d31c"
	guardian2 = "58076f561cc62a47087b567c86f986426dfcd000"
	ethDevnetRPC  = "http://eth-devnet:8545"
	ethDevnet2RPC = "http://eth-devnet2:8545"
	wormholeContractAddress        = "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
	delegatedGuardiansContractAddr = "0xfE82e8f24A51E670133f4268cDfc164c49FC3b37"
	anvilPrivateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
	numDevnetGuardians = 3
	delegatedGuardiansABI = `[{"inputs":[{"internalType":"bytes","name":"vaa","type":"bytes"}],"name":"submitConfig","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"nextConfigIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
)

var devnetGuardians = []string{
	guardian0,
	guardian1,
	guardian2,
}

func getNextConfigIndex(t *testing.T) uint64 {
	client, err := ethclient.Dial(ethDevnetRPC)
	require.NoError(t, err, "Failed to connect to Ethereum RPC")
	defer client.Close()

	contractAddr := eth_common.HexToAddress(delegatedGuardiansContractAddr)

	code, err := client.CodeAt(context.Background(), contractAddr, nil)
	require.NoError(t, err, "Failed to get contract code")
	require.NotEmpty(t, code, "DelegatedGuardians contract not deployed at %s", delegatedGuardiansContractAddr)

	parsedABI, err := abi.JSON(strings.NewReader(delegatedGuardiansABI))
	require.NoError(t, err, "Failed to parse ABI")

	data, err := parsedABI.Pack("nextConfigIndex")
	require.NoError(t, err, "Failed to pack nextConfigIndex call")

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}, nil)
	require.NoError(t, err, "Failed to call nextConfigIndex")
	require.NotEmpty(t, result, "Empty response from nextConfigIndex - contract may not be properly deployed")

	var configIndex *big.Int
	err = parsedABI.UnpackIntoInterface(&configIndex, "nextConfigIndex", result)
	require.NoError(t, err, "Failed to unpack nextConfigIndex result")

	t.Logf("Current nextConfigIndex: %d", configIndex.Uint64())
	return configIndex.Uint64()
}

func publishMessageToEthereum(t *testing.T, nonce uint32, payload []byte, consistencyLevel uint8) string {
	client, err := ethclient.Dial(ethDevnet2RPC)
	require.NoError(t, err, "Failed to connect to Ethereum RPC")
	defer client.Close()

	privateKey, err := eth_crypto.HexToECDSA(anvilPrivateKey)
	require.NoError(t, err, "Failed to parse private key")

	chainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "Failed to get chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	require.NoError(t, err, "Failed to create transactor")
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(100000)

	contractAddr := eth_common.HexToAddress(wormholeContractAddress)

	contract, err := ethabi.NewAbiTransactor(contractAddr, client)
	require.NoError(t, err, "Failed to create contract instance")

	tx, err := contract.PublishMessage(auth, nonce, payload, consistencyLevel)
	require.NoError(t, err, "Failed to send transaction")

	t.Logf("Transaction sent successfully: %s", tx.Hash().Hex())
	return tx.Hash().Hex()
}

func startGossipCollector(t *testing.T) *GossipCollector {
	bootstrapPeers := "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw" // guardian 0 peer
	networkID := "/wormhole/dev"
	port := uint(9999)

	collector, err := NewGossipCollector(bootstrapPeers, networkID, port)
	require.NoError(t, err, "Failed to create gossip collector")

	time.Sleep(2 * time.Second)
	t.Logf("Gossip collector started and connected")

	return collector
}

func stopGossipCollector(t *testing.T, collector *GossipCollector) *CapturedMessages {
	messages := collector.Capture()
	collector.Stop()

	t.Logf("Gossip collector stopped")
	t.Logf("Captured %d delegate observations", len(messages.DelegateObservations))
	t.Logf("Captured %d observation batches", len(messages.ObservationBatches))
	t.Logf("Captured %d VAAs", len(messages.VAAs))
	t.Logf("Captured %d observation requests", len(messages.ObservationRequests))

	return messages
}

type GossipCollector struct {
	batchObsvC         chan *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]
	signedIncomingVaaC chan *gossipv1.SignedVAAWithQuorum
	obsvReqC           chan *gossipv1.ObservationRequest
	delegateObsvC      chan *gossipv1.DelegateObservation
	signedGovCfgC      chan *gossipv1.SignedChainGovernorConfig
	signedGovStatusC   chan *gossipv1.SignedChainGovernorStatus

	delegateObservations []*gossipv1.DelegateObservation
	observationBatches   []*common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]
	vaas                 []*gossipv1.SignedVAAWithQuorum
	observationRequests  []*gossipv1.ObservationRequest

	cancel context.CancelFunc
	logger *zap.Logger
}

type CapturedMessages struct {
	DelegateObservations []*gossipv1.DelegateObservation
	ObservationBatches   []*common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]
	VAAs                 []*gossipv1.SignedVAAWithQuorum
	ObservationRequests  []*gossipv1.ObservationRequest
}

func NewGossipCollector(bootstrapPeers, networkID string, port uint) (*GossipCollector, error) {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	heartbeatC := make(chan *gossipv1.Heartbeat, 1024)
	gst := common.NewGuardianSetState(heartbeatC)

	// Initialize guardian set so messages aren't dropped
	guardianAddrs := make([]eth_common.Address, len(devnetGuardians))
	for i, addrHex := range devnetGuardians {
		addr, err := hex.DecodeString(addrHex)
		if err != nil {
			cancel()
			return nil, err
		}
		guardianAddrs[i] = eth_common.BytesToAddress(addr)
	}
	gs := common.NewGuardianSet(guardianAddrs, 0)
	gst.Set(gs)

	gc := &GossipCollector{
		batchObsvC:           make(chan *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch], 1024),
		signedIncomingVaaC:   make(chan *gossipv1.SignedVAAWithQuorum, 1024),
		obsvReqC:             make(chan *gossipv1.ObservationRequest, 1024),
		delegateObsvC:        make(chan *gossipv1.DelegateObservation, 1024),
		signedGovCfgC:        make(chan *gossipv1.SignedChainGovernorConfig, 1024),
		signedGovStatusC:     make(chan *gossipv1.SignedChainGovernorStatus, 1024),
		delegateObservations: make([]*gossipv1.DelegateObservation, 0),
		observationBatches:   make([]*common.MsgWithTimeStamp[gossipv1.SignedObservationBatch], 0),
		vaas:                 make([]*gossipv1.SignedVAAWithQuorum, 0),
		observationRequests:  make([]*gossipv1.ObservationRequest, 0),
		cancel:               cancel,
		logger:               logger,
	}

	components := p2p.DefaultComponents()
	components.Port = port

	params, err := p2p.NewRunParams(
		bootstrapPeers,
		networkID,
		priv,
		gst,
		cancel,
		p2p.WithSignedObservationBatchListener(gc.batchObsvC),
		p2p.WithSignedVAAListener(gc.signedIncomingVaaC),
		p2p.WithObservationRequestListener(gc.obsvReqC),
		p2p.WithDelegateObservationListener(gc.delegateObsvC),
		p2p.WithChainGovernorConfigListener(gc.signedGovCfgC),
		p2p.WithChainGovernorStatusListener(gc.signedGovStatusC),
		p2p.WithDisableHeartbeatVerify(true),
		p2p.WithComponents(components),
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Starting p2p network for gossip collection")

	_ = supervisor.New(ctx, logger, func(ctx context.Context) error {
		logger.Info("p2p network started, collecting messages...")
		return p2p.Run(params)(ctx)
	}, supervisor.WithPropagatePanic)

	go gc.collectMessages(ctx)

	return gc, nil
}

func (gc *GossipCollector) collectMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-gc.batchObsvC:
			gc.observationBatches = append(gc.observationBatches, msg)
			gc.logger.Debug("Collected SignedObservationBatch",
				zap.String("guardian_addr", hex.EncodeToString(msg.Msg.Addr)),
				zap.Int("num_observations", len(msg.Msg.Observations)),
			)

		case msg := <-gc.signedIncomingVaaC:
			gc.vaas = append(gc.vaas, msg)
			gc.logger.Debug("Collected SignedVAAWithQuorum")

		case msg := <-gc.obsvReqC:
			gc.observationRequests = append(gc.observationRequests, msg)
			gc.logger.Debug("Collected ObservationRequest",
				zap.Uint32("chain_id", msg.ChainId),
			)

		case msg := <-gc.delegateObsvC:
			gc.delegateObservations = append(gc.delegateObservations, msg)
			gc.logger.Info("Collected DelegateObservation",
				zap.Uint32("emitter_chain", msg.EmitterChain),
				zap.Uint64("sequence", msg.Sequence),
				zap.String("guardian_addr", hex.EncodeToString(msg.GuardianAddr)),
			)

		case <-gc.signedGovCfgC:
			gc.logger.Debug("Collected SignedChainGovernorConfig")

		case <-gc.signedGovStatusC:
			gc.logger.Debug("Collected SignedChainGovernorStatus")
		}
	}
}

func (gc *GossipCollector) Stop() {
	gc.cancel()
}

func (gc *GossipCollector) Capture() *CapturedMessages {
	return &CapturedMessages{
		DelegateObservations: append([]*gossipv1.DelegateObservation{}, gc.delegateObservations...),
		ObservationBatches:   append([]*common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]{}, gc.observationBatches...),
		VAAs:                 append([]*gossipv1.SignedVAAWithQuorum{}, gc.vaas...),
		ObservationRequests:  append([]*gossipv1.ObservationRequest{}, gc.observationRequests...),
	}
}

func (cm *CapturedMessages) FindDelegateObservationsByGuardian(guardianAddr string) []*gossipv1.DelegateObservation {
	result := make([]*gossipv1.DelegateObservation, 0)
	for _, obs := range cm.DelegateObservations {
		if hex.EncodeToString(obs.GuardianAddr) == guardianAddr {
			result = append(result, obs)
		}
	}
	return result
}

func (cm *CapturedMessages) FindDelegateObservationsByChain(chainID uint32) []*gossipv1.DelegateObservation {
	result := make([]*gossipv1.DelegateObservation, 0)
	for _, obs := range cm.DelegateObservations {
		if obs.EmitterChain == chainID {
			result = append(result, obs)
		}
	}
	return result
}

func (cm *CapturedMessages) FindObservationBatchByGuardian(guardianAddr string) *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch] {
	for _, batch := range cm.ObservationBatches {
		if hex.EncodeToString(batch.Msg.Addr) == guardianAddr {
			return batch
		}
	}
	return nil
}

// filterObservationsByChain filters observations to only include those from the specified chain ID
func filterObservationsByChain(observations []*gossipv1.Observation, chainID string) []*gossipv1.Observation {
	var filtered []*gossipv1.Observation
	for _, obs := range observations {
		if strings.HasPrefix(obs.MessageId, chainID+"/") {
			filtered = append(filtered, obs)
		}
	}
	return filtered
}

// sortObservations sorts a slice of observations into ascending order.
func sortObservations(observations []*gossipv1.Observation) {
	slices.SortFunc(observations, func(a, b *gossipv1.Observation) int {
		if a.MessageId < b.MessageId {
			return -1
		}
		if a.MessageId > b.MessageId {
			return 1
		}
		return 0
	})
}

// ensureEquivalentObservationBatches ensures all batches provided are equivalent.
func ensureEquivalentObservationBatches(t *testing.T, batches ...*gossipv1.SignedObservationBatch) {
	t.Helper()
	require.NotEmpty(t, batches)

	for _, b := range batches {
		sortObservations(b.Observations)
	}

	sentinel := batches[0]
	for _, b := range batches {
		require.Equal(t, len(sentinel.Observations), len(b.Observations))
		for idx := range sentinel.Observations {
			require.True(t, bytes.Equal(sentinel.Observations[idx].Hash, b.Observations[idx].Hash))
			require.True(t, bytes.Equal(sentinel.Observations[idx].TxHash, b.Observations[idx].TxHash))
			require.Equal(t, sentinel.Observations[idx].MessageId, b.Observations[idx].MessageId)
		}
	}
}

// ensureEquivalentDelegateObservations ensures all delegate observations provided are equivalent.
func ensureEquivalentDelegateObservations(t *testing.T, observations ...*gossipv1.DelegateObservation) {
	t.Helper()
	require.NotEmpty(t, observations)

	sentinel := observations[0]
	for _, ob := range observations[1:] {
		require.Equal(t, sentinel.Nonce, ob.Nonce)
		require.Equal(t, sentinel.ConsistencyLevel, ob.ConsistencyLevel)
		require.Equal(t, sentinel.EmitterChain, ob.EmitterChain)
		require.True(t, bytes.Equal(sentinel.EmitterAddress, ob.EmitterAddress))
		require.Equal(t, sentinel.Sequence, ob.Sequence)
		require.True(t, bytes.Equal(sentinel.Payload, ob.Payload))
		require.True(t, bytes.Equal(sentinel.TxHash, ob.TxHash))
	}
}

func TestDelegateChainUndelegated(t *testing.T) {
	config := map[vaa.ChainID]vaa.DelegatedGuardianConfig{
		4: {
			Threshold: 0,
			Keys:      []eth_common.Address{},
		},
	}

	createAndSubmitDelegatedGuardiansConfig(t, config)
	t.Log("Waiting for guardian to pick up configuration...")
	time.Sleep(20 * time.Second)

	collector := startGossipCollector(t)
	defer collector.Stop()

	publishMessageToEthereum(t, 0, []byte{0xde, 0xad, 0xbe, 0xef}, 200)

	time.Sleep(10 * time.Second)

	messages := stopGossipCollector(t, collector)

	// in this case we should not have delegated observations
	assert.Equal(t, 0, len(messages.DelegateObservations))


	t.Logf("\n=== All Delegate Observations ===")
	for i, obs := range messages.DelegateObservations {
		t.Logf("\nDelegate Observation #%d:", i+1)
		t.Logf("  Timestamp: %d", obs.Timestamp)
		t.Logf("  Nonce: %d", obs.Nonce)
		t.Logf("  EmitterChain: %d", obs.EmitterChain)
		t.Logf("  EmitterAddress: %s", hex.EncodeToString(obs.EmitterAddress))
		t.Logf("  Sequence: %d", obs.Sequence)
		t.Logf("  ConsistencyLevel: %d", obs.ConsistencyLevel)
		t.Logf("  Payload: %s", hex.EncodeToString(obs.Payload))
		t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
		t.Logf("  GuardianAddr: %s", hex.EncodeToString(obs.GuardianAddr))
	}

	// When delegated guardians are disabled (threshold=0), guardians should still observe messages
	ob0 := messages.FindObservationBatchByGuardian(guardian0)
	ob1 := messages.FindObservationBatchByGuardian(guardian1)
	ob2 := messages.FindObservationBatchByGuardian(guardian2)

	// Filter observations to only include chain 4 messages (test messages)
	// This avoids counting governance VAAs from chain 2 that may still be in flight
	if ob1 != nil {
		ob1.Msg.Observations = filterObservationsByChain(ob1.Msg.Observations, "4")
	}
	if ob2 != nil {
		ob2.Msg.Observations = filterObservationsByChain(ob2.Msg.Observations, "4")
	}

	if ob0 != nil {
		t.Logf("\n=== Observation Batch 0 (guardian0) ===")
		t.Logf("  Timestamp: %d", ob0.Timestamp.Unix())
		for i, obs := range ob0.Msg.Observations {
			t.Logf("\nObservation #%d:", i+1)
			t.Logf("  MessageId: %s", obs.MessageId)
			t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
			t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
		}
	} else {
		t.Log("No observation batch from guardian0")
	}

	if ob1 != nil {
		t.Logf("\n=== Observation Batch 1 (guardian1) ===")
		t.Logf("  Timestamp: %d", ob1.Timestamp.Unix())
		for i, obs := range ob1.Msg.Observations {
			t.Logf("\nObservation #%d:", i+1)
			t.Logf("  MessageId: %s", obs.MessageId)
			t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
			t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
		}
	} else {
		t.Log("No observation batch from guardian1")
	}

	if ob2 != nil {
		t.Logf("\n=== Observation Batch 2 (guardian2) ===")
		t.Logf("  Timestamp: %d", ob2.Timestamp.Unix())
		for i, obs := range ob2.Msg.Observations {
			t.Logf("\nObservation #%d:", i+1)
			t.Logf("  MessageId: %s", obs.MessageId)
			t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
			t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
		}
	} else {
		t.Log("No observation batch from guardian2")
	}

	// guardian-1 and guardian-2 watch chain 4, so they should have observations
	// guardian-0 only watches chain 2, so it won't have observations for chain 4
	require.NotNil(t, ob1, "Expected observation batch from guardian1")
	require.NotNil(t, ob2, "Expected observation batch from guardian2")

	ensureEquivalentObservationBatches(t, ob1.Msg, ob2.Msg)

	assert.Equal(t, 1, len(ob1.Msg.Observations), "Expected exactly 1 observation in guardian1 batch")
	assert.Equal(t, 1, len(ob2.Msg.Observations), "Expected exactly 1 observation in guardian2 batch")

	// chain id 4
	require.NotEmpty(t, ob1.Msg.Observations, "Guardian1 should have observations")
	messageID := ob1.Msg.Observations[0].MessageId
	assert.True(t, strings.HasPrefix(messageID, "4/"), "Message ID should be for chain 4, got: %s", messageID)
	t.Logf("Message ID: %s", messageID)

	// VAA is not produced because guardian-0 is still not listening to evm2
	// even if we undelegate chain id 4, so no quorum is reached
	require.Empty(t, messages.VAAs, "Expected no VAA to be produced")
	t.Logf("VAAs produced: %d", len(messages.VAAs))

	// we rollback configuration changes to devnet default config
	rollbackConfig := map[vaa.ChainID]vaa.DelegatedGuardianConfig{
		4: {
			Threshold: 2,
			Keys: []eth_common.Address{
				eth_common.HexToAddress("0x" + guardian1),
				eth_common.HexToAddress("0x" + guardian2),
			},
		},
	}

	createAndSubmitDelegatedGuardiansConfig(t, rollbackConfig)
	t.Log("Waiting for guardian to pick up configuration...")
	time.Sleep(20 * time.Second)
}

func TestDelegateObservationScenario(t *testing.T) {
	// Ensure delegated guardians are configured for chain 4
	config := map[vaa.ChainID]vaa.DelegatedGuardianConfig{
		4: {
			Threshold: 2,
			Keys: []eth_common.Address{
				eth_common.HexToAddress("0x" + guardian1),
				eth_common.HexToAddress("0x" + guardian2),
			},
		},
	}
	createAndSubmitDelegatedGuardiansConfig(t, config)
	t.Log("Waiting for guardian to pick up configuration...")
	time.Sleep(20 * time.Second)

	collector := startGossipCollector(t)
	defer collector.Stop()

	publishMessageToEthereum(t, 0, []byte{0xde, 0xad, 0xbe, 0xef}, 200)

	time.Sleep(10 * time.Second)

	messages := stopGossipCollector(t, collector)

	// ensure observation batches are as expected
	assert.Equal(t, 2, len(messages.DelegateObservations))
	ensureEquivalentDelegateObservations(t, messages.DelegateObservations...)

	message0 := messages.DelegateObservations[0]
	message1 := messages.DelegateObservations[1]

	// we are expecting guardian 1 and 2 only to send delegated observations
	actualGuardians := []string{
		hex.EncodeToString(message0.GuardianAddr),
		hex.EncodeToString(message1.GuardianAddr),
	}
	assert.ElementsMatch(t, devnetGuardians[1:3], actualGuardians)

	t.Logf("\n=== All Delegate Observations ===")
	for i, obs := range messages.DelegateObservations {
		t.Logf("\nDelegate Observation #%d:", i+1)
		t.Logf("  Timestamp: %d", obs.Timestamp)
		t.Logf("  Nonce: %d", obs.Nonce)
		t.Logf("  EmitterChain: %d", obs.EmitterChain)
		t.Logf("  EmitterAddress: %s", hex.EncodeToString(obs.EmitterAddress))
		t.Logf("  Sequence: %d", obs.Sequence)
		t.Logf("  ConsistencyLevel: %d", obs.ConsistencyLevel)
		t.Logf("  Payload: %s", hex.EncodeToString(obs.Payload))
		t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
		t.Logf("  GuardianAddr: %s", hex.EncodeToString(obs.GuardianAddr))
	}

	ob0 := messages.FindObservationBatchByGuardian(guardian0)
	ob1 := messages.FindObservationBatchByGuardian(guardian1)
	ob2 := messages.FindObservationBatchByGuardian(guardian2)

	// Filter observations to only include chain 4 messages (test messages)
	// This avoids counting governance VAAs from chain 2 that may still be in flight
	if ob0 != nil {
		ob0.Msg.Observations = filterObservationsByChain(ob0.Msg.Observations, "4")
	}
	if ob1 != nil {
		ob1.Msg.Observations = filterObservationsByChain(ob1.Msg.Observations, "4")
	}
	if ob2 != nil {
		ob2.Msg.Observations = filterObservationsByChain(ob2.Msg.Observations, "4")
	}

	t.Logf("\n=== Observation Batch 0 ===")
	t.Logf("  Timestamp: %d", ob0.Timestamp.Unix())
	for i, obs := range ob0.Msg.Observations {
		t.Logf("\nObservation #%d:", i+1)
		t.Logf("  MessageId: %s", obs.MessageId)
		t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
		t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
	}
	t.Logf("\n=== Observation Batch 1 ===")
	t.Logf("  Timestamp: %d", ob1.Timestamp.Unix())
	for i, obs := range ob1.Msg.Observations {
		t.Logf("\nObservation #%d:", i+1)
		t.Logf("  MessageId: %s", obs.MessageId)
		t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
		t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
	}
	t.Logf("\n=== Observation Batch 2 ===")
	t.Logf("  Timestamp: %d", ob2.Timestamp.Unix())
	for i, obs := range ob2.Msg.Observations {
		t.Logf("\nObservation #%d:", i+1)
		t.Logf("  MessageId: %s", obs.MessageId)
		t.Logf("  Hash: %s", hex.EncodeToString(obs.Hash))
		t.Logf("  TxHash: %s", hex.EncodeToString(obs.TxHash))
	}
	// ensure observation batches are as expected
	ensureEquivalentObservationBatches(t, ob0.Msg, ob1.Msg, ob2.Msg)
	assert.Equal(t, 1, len(ob0.Msg.Observations))

	// ensure canonical observations do not come before delegate quorum
	maxDelegateObservationsTimestamp := message0.Timestamp
	if message1.Timestamp > message0.Timestamp {
		maxDelegateObservationsTimestamp = message1.Timestamp
	}
	// NOTE: we cannot use assert.Less since seconds is not enough precision
	assert.LessOrEqual(t, int64(maxDelegateObservationsTimestamp), ob0.Timestamp.Unix())

	// ensure observation matches delegate observation
	mp, err := delegateObservationToMessagePublication(message0)
	require.NoError(t, err, "Failed to convert delegate observation to message publication")

	hash := mp.CreateDigest()
	assert.Equal(t, hash, hex.EncodeToString(ob0.Msg.Observations[0].Hash))
	assert.Equal(t, message0.TxHash, ob0.Msg.Observations[0].TxHash)
}

func createAndSubmitDelegatedGuardiansConfig(t *testing.T, config map[vaa.ChainID]vaa.DelegatedGuardianConfig) {
	configIndex := getNextConfigIndex(t)
	t.Logf("Creating delegated guardians config VAA with configIndex=%d", configIndex)

	body, err := vaa.BodyDelegatedGuardiansSetConfig{
		ConfigIndex: uint256.NewInt(configIndex),
		Config:      config,
	}.Serialize()
	require.NoError(t, err, "Failed to serialize governance body")

	timestamp := time.Now()
	nonce := uint32(configIndex)
	sequence := uint64(configIndex)
	guardianSetIndex := uint32(0)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)

	for i := 0; i < numDevnetGuardians; i++ {
		key := devnet.InsecureDeterministicEcdsaKeyByIndex(uint64(i))
		v.AddSignature(key, uint8(i))
	}

	vaaBytes, err := v.Marshal()
	require.NoError(t, err, "Failed to marshal VAA")

	t.Logf("Created governance VAA with %d signatures", len(v.Signatures))
	t.Logf("VAA hex: %s", hex.EncodeToString(vaaBytes))

	submitConfigToDelegatedGuardians(t, vaaBytes)

	t.Log("Governance VAA submitted successfully")
}

func submitConfigToDelegatedGuardians(t *testing.T, vaaBytes []byte) {
	client, err := ethclient.Dial(ethDevnetRPC)
	require.NoError(t, err, "Failed to connect to Ethereum RPC")
	defer client.Close()

	privateKey, err := eth_crypto.HexToECDSA(anvilPrivateKey)
	require.NoError(t, err, "Failed to parse private key")

	chainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "Failed to get chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	require.NoError(t, err, "Failed to create transactor")
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(500000)

	contractAddr := eth_common.HexToAddress(delegatedGuardiansContractAddr)

	parsedABI, err := abi.JSON(strings.NewReader(delegatedGuardiansABI))
	require.NoError(t, err, "Failed to parse ABI")

	data, err := parsedABI.Pack("submitConfig", vaaBytes)
	require.NoError(t, err, "Failed to pack submitConfig call")

	nonce, err := client.PendingNonceAt(context.Background(), auth.From)
	require.NoError(t, err, "Failed to get nonce")

	gasPrice, err := client.SuggestGasPrice(context.Background())
	require.NoError(t, err, "Failed to get gas price")

	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), auth.GasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	require.NoError(t, err, "Failed to sign transaction")

	err = client.SendTransaction(context.Background(), signedTx)
	require.NoError(t, err, "Failed to send transaction")

	t.Logf("Transaction sent: %s", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	require.NoError(t, err, "Failed to wait for transaction")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Transaction failed")

	t.Logf("Transaction mined in block %d", receipt.BlockNumber.Uint64())
}
