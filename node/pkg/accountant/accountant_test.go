package accountant

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethCommon "github.com/ethereum/go-ethereum/common"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	enforceAccountant     = true
	dontEnforceAccountant = false
)

type MockAccountantWormchainConn struct {
	BroadcastTxResponse string

	lock   sync.Mutex
	txResp *sdktx.BroadcastTxResponse
}

func (c *MockAccountantWormchainConn) Close() {
}

func (c *MockAccountantWormchainConn) SenderAddress() string {
	return "wormfakesigner"
}

func (c *MockAccountantWormchainConn) SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error) {
	return []byte{}, nil
}

func (c *MockAccountantWormchainConn) SignAndBroadcastTx(ctx context.Context, msg sdktypes.Msg) (*sdktx.BroadcastTxResponse, error) {
	for {
		c.lock.Lock()
		if c.txResp != nil {
			resp := c.txResp
			c.txResp = nil
			c.lock.Unlock()
			return resp, nil
		}
		c.lock.Unlock()
		time.Sleep(50 * time.Millisecond) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep
	}
}

func (c *MockAccountantWormchainConn) BroadcastTxResponseToString(txResp *sdktx.BroadcastTxResponse) string {
	return c.BroadcastTxResponse
}

func (c *MockAccountantWormchainConn) SetTxResp(txResp *sdktx.BroadcastTxResponse) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.txResp = txResp
}

func (c *MockAccountantWormchainConn) TxRespPending() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.txResp != nil
}

func (c *MockAccountantWormchainConn) WaitUntilTxRespConsumed() {
	for {
		stillPending := c.TxRespPending()
		time.Sleep(50 * time.Millisecond) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep
		if !stillPending {
			break
		}
	}
}

// AuditMockWormchainConn extends MockAccountantWormchainConn with configurable
// query responses for audit testing.
type AuditMockWormchainConn struct {
	MockAccountantWormchainConn

	allPendingTransfersResp []byte
	allPendingTransfersErr  error
	batchTransferStatusResp []byte
	batchTransferStatusErr  error

	queryLock         sync.Mutex
	queries           []string
	contractAddresses []string
}

func (c *AuditMockWormchainConn) SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error) {
	queryStr := string(query)
	c.queryLock.Lock()
	c.queries = append(c.queries, queryStr)
	c.contractAddresses = append(c.contractAddresses, contractAddress)
	c.queryLock.Unlock()

	if strings.Contains(queryStr, "all_pending_transfers") {
		return c.allPendingTransfersResp, c.allPendingTransfersErr
	}
	if strings.Contains(queryStr, "batch_transfer_status") {
		return c.batchTransferStatusResp, c.batchTransferStatusErr
	}
	return []byte{}, nil
}

func (c *AuditMockWormchainConn) QueryCount() int {
	c.queryLock.Lock()
	defer c.queryLock.Unlock()
	return len(c.queries)
}

func newAccountantForTest(
	t *testing.T,
	logger *zap.Logger,
	ctx context.Context,
	accountantCheckEnabled bool,
	obsvReqWriteC chan<- *gossipv1.ObservationRequest,
	acctWriteC chan<- *common.MessagePublication,
	wormchainConn *MockAccountantWormchainConn,
) *Accountant {
	var db guardianDB.MockAccountantDB

	pk := devnet.InsecureDeterministicEcdsaKeyByIndex(uint64(0))
	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(pk)
	require.NoError(t, err)

	gst := common.NewGuardianSetState(nil)
	gs := &common.GuardianSet{Keys: []ethCommon.Address{ethCommon.HexToAddress("0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")}}
	gst.Set(gs)

	env := common.GoTest
	if wormchainConn != nil {
		env = common.AccountantMock
	}

	acct := NewAccountant(
		ctx,
		logger,
		&db,
		obsvReqWriteC,
		"0xdeadbeef", // accountantContract
		"none",       // accountantWS
		wormchainConn,
		accountantCheckEnabled,
		"",
		nil,
		guardianSigner,
		gst,
		acctWriteC,
		DefaultSubmitObservationBatchSize,
		env,
	)

	err = acct.Start(ctx)
	require.NoError(t, err)
	return acct
}

// Converts a TxHash string into a byte array to be used as a TxID.
func hashToTxID(str string) []byte {
	if (len(str) > 2) && (str[0] == '0') && (str[1] == 'x') {
		str = str[2:]
	}

	return ethCommon.HexToHash(str).Bytes()
}

// Note this method assumes 18 decimals for the amount.
func buildMockTransferPayloadBytes(
	t uint8, //nolint:unparam
	tokenChainID vaa.ChainID, //nolint:unparam
	tokenAddrStr string, //nolint:unparam
	toChainID vaa.ChainID, //nolint:unparam
	toAddrStr string, //nolint:unparam
	amtFloat float64, //nolint:unparam
) []byte {
	bytes := make([]byte, 101)
	bytes[0] = t

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

func TestVaaFromUninterestingEmitter(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	var payload = []byte{1, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payload,
	}

	shouldPublish, err := acct.SubmitObservation(&msg)
	require.NoError(t, err)
	assert.Equal(t, true, shouldPublish)
	assert.Equal(t, 0, len(acct.pendingTransfers))
}

func TestVaaForUninterestingPayloadType(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	var payload = []byte{2, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payload,
	}

	shouldPublish, err := acct.SubmitObservation(&msg)
	require.NoError(t, err)
	assert.Equal(t, true, shouldPublish)
	assert.Equal(t, 0, len(acct.pendingTransfers))
}

func TestInterestingTransferShouldNotBeBlockedWhenNotEnforcingAccountant(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, dontEnforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		vaa.ChainIDPolygon,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		1.25,
	)

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	shouldPublish, err := acct.SubmitObservation(&msg)
	require.NoError(t, err)

	// The transfer should not be blocked, but it should be in the pending map.
	assert.Equal(t, true, shouldPublish)
	pe, exists := acct.pendingTransfers[msg.MessageIDString()]
	require.Equal(t, true, exists)
	require.NotNil(t, pe)

	// PublishTransfer should not publish to the channel but it should remove it from the map.
	acct.publishTransferAlreadyLocked(pe)
	assert.Equal(t, 0, len(acct.msgChan))
	assert.Equal(t, 0, len(acct.pendingTransfers))
}

func TestInterestingTransferShouldBeBlockedWhenEnforcingAccountant(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		vaa.ChainIDPolygon,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		1.25,
	)

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	shouldPublish, err := acct.SubmitObservation(&msg)
	require.NoError(t, err)
	assert.Equal(t, false, shouldPublish)
	assert.Equal(t, 1, len(acct.pendingTransfers))
	assert.Equal(t, 0, len(acct.msgChan))

	// The same message a second time should still be blocked, but the pending map should not change.
	msg2 := msg
	shouldPublish, err = acct.SubmitObservation(&msg2)
	require.NoError(t, err)
	assert.Equal(t, false, shouldPublish)
	assert.Equal(t, 0, len(acct.msgChan))
	pe, exists := acct.pendingTransfers[msg.MessageIDString()]
	require.Equal(t, true, exists)
	require.NotNil(t, pe)

	// PublishTransfer should publish to the channel and remove it from the map.
	acct.publishTransferAlreadyLocked(pe)
	assert.Equal(t, 1, len(acct.msgChan))
	assert.Equal(t, 0, len(acct.pendingTransfers))
}

func TestForDeadlock(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)
	wormchainConn := MockAccountantWormchainConn{}
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, &wormchainConn)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")

	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		vaa.ChainIDPolygon,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		1.25,
	)

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1683136244),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	var txResp sdktx.BroadcastTxResponse
	err := json.Unmarshal(createTxRespForCommitted(), &txResp)
	require.NoError(t, err)
	wormchainConn.SetTxResp(&txResp)

	shouldPublish, err := acct.SubmitObservation(&msg)
	require.NoError(t, err)
	assert.Equal(t, false, shouldPublish)
	assert.Equal(t, 1, len(acct.pendingTransfers))
	assert.Equal(t, 0, len(acct.msgChan))

	// Wait until the response gets received from the contract.
	wormchainConn.WaitUntilTxRespConsumed()

	assert.Equal(t, 1, len(acct.msgChan))

	msg2 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1683136244),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	var txResp2 sdktx.BroadcastTxResponse
	err = json.Unmarshal(createTxRespForCommitted(), &txResp2)
	require.NoError(t, err)
	wormchainConn.SetTxResp(&txResp2)

	shouldPublish, _ = acct.SubmitObservation(&msg2)
	require.NoError(t, err)
	assert.Equal(t, false, shouldPublish)
	assert.Equal(t, 1, len(acct.pendingTransfers))
	assert.Equal(t, 1, len(acct.msgChan))

	wormchainConn.WaitUntilTxRespConsumed()

	assert.Equal(t, 2, len(acct.msgChan))

	resultMsg := <-acctChan
	assert.Equal(t, msg, *resultMsg)

	resultMsg2 := <-acctChan
	assert.Equal(t, msg2, *resultMsg2)
}

func TestNumPendingEntries(t *testing.T) {
	pe1 := &pendingEntry{msgId: "1"}
	pe2 := &pendingEntry{msgId: "2"}
	pe3 := &pendingEntry{msgId: "3"}
	pe4 := &pendingEntry{msgId: "4"}

	tmpMap := map[string][]*pendingEntry{
		"2-aabbcc": {pe1, pe2, pe3},
		"4-ddeeff": {pe4},
	}

	assert.Equal(t, 4, numPendingEntries(tmpMap))
}

func TestNumPendingEntriesEmpty(t *testing.T) {
	tmpMap := map[string][]*pendingEntry{}
	assert.Equal(t, 0, numPendingEntries(tmpMap))
}

func TestNumPendingEntriesNilMap(t *testing.T) {
	var tmpMap map[string][]*pendingEntry
	assert.Equal(t, 0, numPendingEntries(tmpMap))
}

func TestNumPendingEntriesEmptySlices(t *testing.T) {
	tmpMap := map[string][]*pendingEntry{
		"1-aa": {},
		"2-bb": {},
		"3-cc": {},
		"4-dd": {},
		"5-ee": {},
	}
	assert.Equal(t, 0, numPendingEntries(tmpMap))
}

func TestNumPendingEntriesAllNilValues(t *testing.T) {
	tmpMap := map[string][]*pendingEntry{
		"2-aabbcc": {nil, nil, nil},
	}
	assert.Equal(t, 0, numPendingEntries(tmpMap))
}

func TestNumPendingEntriesMixedNils(t *testing.T) {
	pe1 := &pendingEntry{msgId: "1"}
	pe2 := &pendingEntry{msgId: "2"}

	tmpMap := map[string][]*pendingEntry{
		"2-aabbcc": {pe1, nil, pe2, nil},
		"4-ddeeff": {nil},
	}
	assert.Equal(t, 2, numPendingEntries(tmpMap))
}

func TestCreateAuditMapMultipleTransfersSameTxHash(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		vaa.ChainIDPolygon,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		1.25,
	)

	// Two transfers from the same transaction (same TxID and EmitterChain) but different sequences.
	txID := hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063")

	msg1 := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	msg2 := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	// Manually add both transfers to the pending map (they have different msgIds because of different sequences).
	pe1 := &pendingEntry{msg: msg1, msgId: msg1.MessageIDString(), digest: "digest1"}
	pe2 := &pendingEntry{msg: msg2, msgId: msg2.MessageIDString(), digest: "digest2"}
	pe1.setUpdTime()
	pe2.setUpdTime()
	acct.pendingTransfers[pe1.msgId] = pe1
	acct.pendingTransfers[pe2.msgId] = pe2

	// Verify the two messages have different msgIds but the same audit key.
	require.NotEqual(t, pe1.msgId, pe2.msgId)
	require.Equal(t, pe1.makeAuditKey(), pe2.makeAuditKey())

	tmpMap := acct.createAuditMap(false)

	// There should be one key in the map (the shared audit key) with two entries.
	assert.Equal(t, 1, len(tmpMap))
	assert.Equal(t, 2, numPendingEntries(tmpMap))

	key := pe1.makeAuditKey()
	entries, exists := tmpMap[key]
	require.Equal(t, true, exists)
	assert.Equal(t, 2, len(entries))
}

func TestCreateAuditMapDifferentTxHashes(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	payloadBytes := buildMockTransferPayloadBytes(3,
		vaa.ChainIDBSC,
		"0x0290fb167208af455bb137780163b7b7a9a10c16",
		vaa.ChainIDEthereum,
		"0x0290fb167208af455bb137780163b7b7a9a10c16",
		2.50,
	)

	msg1 := &common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	msg2 := &common.MessagePublication{
		TxID:             hashToTxID("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	pe1 := &pendingEntry{msg: msg1, msgId: msg1.MessageIDString(), digest: "digest1"}
	pe2 := &pendingEntry{msg: msg2, msgId: msg2.MessageIDString(), digest: "digest2"}
	pe1.setUpdTime()
	pe2.setUpdTime()
	acct.pendingTransfers[pe1.msgId] = pe1
	acct.pendingTransfers[pe2.msgId] = pe2

	// Verify the two messages have different audit keys.
	require.NotEqual(t, pe1.makeAuditKey(), pe2.makeAuditKey())

	tmpMap := acct.createAuditMap(false)

	// There should be two keys in the map, each with one entry.
	assert.Equal(t, 2, len(tmpMap))
	assert.Equal(t, 2, numPendingEntries(tmpMap))

	entries1, exists := tmpMap[pe1.makeAuditKey()]
	require.Equal(t, true, exists)
	assert.Equal(t, 1, len(entries1))

	entries2, exists := tmpMap[pe2.makeAuditKey()]
	require.Equal(t, true, exists)
	assert.Equal(t, 1, len(entries2))
}

func TestCreateAuditMapFiltersNTT(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, logger, ctx, enforceAccountant, obsvReqWriteC, acctChan, nil)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		vaa.ChainIDPolygon,
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8",
		1.25,
	)

	// Two transfers with the same TxID but different NTT flags.
	txID := hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063")

	msg1 := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	msg2 := &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	pe1 := &pendingEntry{msg: msg1, msgId: msg1.MessageIDString(), digest: "digest1", isNTT: false}
	pe2 := &pendingEntry{msg: msg2, msgId: msg2.MessageIDString(), digest: "digest2", isNTT: true}
	pe1.setUpdTime()
	pe2.setUpdTime()
	acct.pendingTransfers[pe1.msgId] = pe1
	acct.pendingTransfers[pe2.msgId] = pe2

	// createAuditMap(false) should only contain the non-NTT transfer.
	baseMap := acct.createAuditMap(false)
	assert.Equal(t, 1, numPendingEntries(baseMap))
	key := pe1.makeAuditKey()
	entries, exists := baseMap[key]
	require.Equal(t, true, exists)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, pe1.msgId, entries[0].msgId)

	// createAuditMap(true) should only contain the NTT transfer.
	nttMap := acct.createAuditMap(true)
	assert.Equal(t, 1, numPendingEntries(nttMap))
	entries, exists = nttMap[key]
	require.Equal(t, true, exists)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, pe2.msgId, entries[0].msgId)
}

// createTestPendingEntry creates a pendingEntry for testing with semi realistic values.
// The digest is computed from the message using CreateDigest().
func createTestPendingEntry(
	emitterChain vaa.ChainID, //nolint:unparam
	emitterAddr vaa.Address,
	sequence uint64,
	txHash []byte,
	payload []byte,
) *pendingEntry {
	msg := &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        time.Unix(1654543099, 0),
		Nonce:            1,
		Sequence:         sequence,
		EmitterChain:     emitterChain,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: 32,
		Payload:          payload,
	}
	return &pendingEntry{
		msg:         msg,
		msgId:       msg.MessageIDString(),
		digest:      msg.CreateDigest(),
		enforceFlag: true,
	}
}

// drainObsvReqChannel drains the observation request channel
func drainObsvReqChannel(ch chan *gossipv1.ObservationRequest) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// drainMsgChannel drains a message publication channel
func drainMsgChannel(ch chan *common.MessagePublication) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// newAccountantForAuditTest creates an accountant configured for audit testing
// with an AuditMockWormchainConn.
func newAccountantForAuditTest(
	t *testing.T,
	logger *zap.Logger,
	ctx context.Context,
	obsvReqWriteC chan *gossipv1.ObservationRequest,
	acctWriteC chan *common.MessagePublication,
	wormchainConn *AuditMockWormchainConn,
) *Accountant {
	return newAccountantForAuditModeTest(t, logger, ctx, obsvReqWriteC, acctWriteC, "0xdeadbeef", wormchainConn, "", nil)
}

func newAccountantForAuditModeTest(
	t *testing.T,
	logger *zap.Logger,
	ctx context.Context,
	obsvReqWriteC chan *gossipv1.ObservationRequest,
	acctWriteC chan *common.MessagePublication,
	contract string,
	wormchainConn *AuditMockWormchainConn,
	nttContract string,
	nttWormchainConn *AuditMockWormchainConn,
) *Accountant {
	var db guardianDB.MockAccountantDB

	pk := devnet.InsecureDeterministicEcdsaKeyByIndex(uint64(0))
	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(pk)
	require.NoError(t, err)

	gst := common.NewGuardianSetState(nil)
	// Guardian set with index 0, our guardian at index 0
	gs := &common.GuardianSet{
		Index: 0,
		Keys:  []ethCommon.Address{ethCommon.HexToAddress("0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")},
	}
	gst.Set(gs)

	acct := NewAccountant(
		ctx,
		logger,
		&db,
		obsvReqWriteC,
		contract,
		"none",
		wormchainConn,
		true, // enforceFlag
		nttContract,
		nttWormchainConn,
		guardianSigner,
		gst,
		acctWriteC,
		DefaultSubmitObservationBatchSize,
		common.GoTest, // Use GoTest to avoid starting worker goroutines that consume from subChan
	)

	err = acct.Start(ctx)
	require.NoError(t, err)
	return acct
}

// Standard test values for audit tests
var (
	testEmitterAddr, _ = vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	testEmitterAddrStr = "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
	testSequence       = uint64(1674568234)
	testTxHash         = []byte{0x06, 0xf5, 0x41, 0xf5, 0xec, 0xfc, 0x43, 0x40, 0x7c, 0x31, 0x58, 0x7a, 0xa6, 0xac, 0x3a, 0x68, 0x9e, 0x89, 0x60, 0xf3, 0x6d, 0xc2, 0x3c, 0x33, 0x2d, 0xb5, 0x51, 0x0d, 0xfc, 0x6a, 0x40, 0x64}
)

func TestRunAuditBaseOnlySkipsNttAudit(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)
	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: []byte(`{"pending":[]}`),
	}

	acct := newAccountantForAuditModeTest(t, logger, ctx, obsvReqWriteC, acctChan, "base-contract", wormchainConn, "", nil)

	require.NotPanics(t, func() {
		acct.runAudit(ctx)
	})
	assert.Equal(t, 1, wormchainConn.QueryCount(), "expected base audit query")
	assert.Equal(t, []string{"base-contract"}, wormchainConn.contractAddresses)
}

func TestRunAuditNttOnlySkipsBaseAudit(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)
	nttWormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: []byte(`{"pending":[]}`),
	}

	acct := newAccountantForAuditModeTest(t, logger, ctx, obsvReqWriteC, acctChan, "", nil, "ntt-contract", nttWormchainConn)

	require.NotPanics(t, func() {
		acct.runAudit(ctx)
	})
	assert.Equal(t, 1, nttWormchainConn.QueryCount(), "expected NTT audit query")
	assert.Equal(t, []string{"ntt-contract"}, nttWormchainConn.contractAddresses)
}

func TestPerformAuditResubmitsUnsignedTransfer(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)

	// Create pending entry
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)

	// Decode the digest from hex to bytes for the mock response
	digestBytes, err := hex.DecodeString(pe.digest)
	require.NoError(t, err)

	// Mock returns pending transfer where guardian 0 hasn't signed (signatures="0")
	allPendingResp := createPendingTransfersForTestWithTxHash(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		testTxHash,
		digestBytes,
		0,   // guardian set index
		"0", // signatures - guardian 0 has NOT signed
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: allPendingResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	// Add the pending entry to the accountant's map
	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	// Create tmpMap for audit (simulating what createAuditMap does)
	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: entry should have been submitted to subChan
	assert.Equal(t, 1, len(acct.subChan), "expected 1 message in subChan")

	// Verify: entry should have been removed from tmpMap because phase 1 found it
	// in the contract's pending list and we hadn't signed it, so it was deleted
	// from tmpMap after being resubmitted.
	assert.Equal(t, 0, len(tmpMap), "expected tmpMap to be empty")

	// Verify: submitPending flag should be set
	assert.True(t, pe.submitPending(), "expected submitPending to be true")

	// Drain channels
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditRequestsReobservation(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	// Mock returns pending transfer where guardian 0 hasn't signed
	// but we DON'T have this transfer locally
	allPendingResp := createPendingTransfersForTestWithTxHash(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		testTxHash,
		[]byte("somedigest"),
		0,   // guardian set index
		"0", // signatures - guardian 0 has NOT signed
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: allPendingResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	// Empty local map - we don't have this transfer
	tmpMap := map[string][]*pendingEntry{}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: reobservation request should have been sent
	assert.Equal(t, 1, len(obsvReqWriteC), "expected 1 reobservation request")

	// Verify the request has correct values
	req := <-obsvReqWriteC
	assert.Equal(t, uint32(vaa.ChainIDEthereum), req.ChainId)
	assert.Equal(t, testTxHash, req.TxHash)

	// Verify: tmpMap remains empty because we started with an empty local map.
	// The contract's pending transfer was handled via reobservation request, not
	// via tmpMap lookup, so tmpMap was never modified.
	assert.Equal(t, 0, len(tmpMap), "expected tmpMap to remain empty")

	// Drain channels
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditSkipsAlreadySigned(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)
	digestBytes, err := hex.DecodeString(pe.digest)
	require.NoError(t, err)

	// Mock returns pending transfer where guardian 0 HAS signed (signatures="1")
	allPendingResp := createPendingTransfersForTestWithTxHash(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		testTxHash,
		digestBytes,
		0,   // guardian set index
		"1", // signatures - guardian 0 HAS signed (bit 0 set)
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: allPendingResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: nothing should have been submitted (we already signed)
	assert.Equal(t, 0, len(acct.subChan), "expected no messages in subChan")
	assert.Equal(t, 0, len(obsvReqWriteC), "expected no reobservation requests")

	// Verify: tmpMap still contains the entry because phase 1 skipped it
	// (guardian already signed), so delete(tmpMap) was never called.
	// Phase 2 iterates remaining tmpMap entries but never removes them from tmpMap.
	assert.Equal(t, 1, len(tmpMap), "expected tmpMap to still contain the entry")

	// Drain channels
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditPhase2Committed(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)

	// Decode the digest from hex to bytes for the mock response
	digestBytes, err := hex.DecodeString(pe.digest)
	require.NoError(t, err)

	// Mock: all_pending_transfers returns empty (transfer not in pending list)
	// Mock: batch_transfer_status returns committed with matching digest
	batchStatusResp := createBatchTransferStatusResponse(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		"committed",
		digestBytes,
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: []byte(`{"pending":[]}`),
		batchTransferStatusResp: batchStatusResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: transfer should have been published to msgChan
	assert.Equal(t, 1, len(acctChan), "expected 1 message in msgChan")

	// Verify: entry should have been removed from pendingTransfers
	acct.pendingTransfersLock.Lock()
	_, exists := acct.pendingTransfers[pe.msgId]
	acct.pendingTransfersLock.Unlock()
	assert.False(t, exists, "expected entry to be removed from pendingTransfers")

	// Verify: tmpMap still contains the entry because phase 2 never deletes from tmpMap.
	// The entry was not in the contract's pending list (phase 1 had nothing to process),
	// and phase 2 only reads tmpMap to query batch status — it publishes the committed
	// transfer but does not remove it from tmpMap.
	assert.Equal(t, 1, len(tmpMap), "expected tmpMap to still contain the entry")

	// Drain channels
	drainMsgChannel(acctChan)
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditPhase2Unknown(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)

	// Mock: all_pending_transfers returns empty
	// Mock: batch_transfer_status returns null (contract doesn't know about it)
	batchStatusResp := createBatchTransferStatusResponse(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		"null",
		nil,
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: []byte(`{"pending":[]}`),
		batchTransferStatusResp: batchStatusResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: entry should have been resubmitted to subChan
	assert.Equal(t, 1, len(acct.subChan), "expected 1 message in subChan")

	// Verify: submitPending flag should be set
	assert.True(t, pe.submitPending(), "expected submitPending to be true")

	// Verify: tmpMap still contains the entry because phase 2 never deletes from tmpMap.
	// The contract returned null status (unknown), so the entry was resubmitted, but
	// tmpMap itself is only modified by delete() in phase 1.
	assert.Equal(t, 1, len(tmpMap), "expected tmpMap to still contain the entry")

	// Drain channels
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditPhase2DigestMismatch(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)

	// Use a DIFFERENT digest than what the pending entry has
	wrongDigestBytes := []byte("this_is_a_completely_different_digest_value_32b")

	// Mock: all_pending_transfers returns empty
	// Mock: batch_transfer_status returns committed with DIFFERENT digest
	batchStatusResp := createBatchTransferStatusResponse(
		uint16(vaa.ChainIDEthereum),
		testEmitterAddrStr,
		testSequence,
		"committed",
		wrongDigestBytes,
	)

	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersResp: []byte(`{"pending":[]}`),
		batchTransferStatusResp: batchStatusResp,
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: nothing should be published to msgChan (digest mismatch)
	assert.Equal(t, 0, len(acctChan), "expected no messages in msgChan")

	// Verify: entry should have been removed from pendingTransfers (dropped)
	acct.pendingTransfersLock.Lock()
	_, exists := acct.pendingTransfers[pe.msgId]
	acct.pendingTransfersLock.Unlock()
	assert.False(t, exists, "expected entry to be removed from pendingTransfers")

	// Verify: tmpMap still contains the entry because phase 2 never deletes from tmpMap.
	// The transfer was dropped from pendingTransfers due to digest mismatch, but
	// tmpMap itself is only modified by delete() in phase 1.
	assert.Equal(t, 1, len(tmpMap), "expected tmpMap to still contain the entry")

	// Drain channels
	drainMsgChannel(acctChan)
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}

func TestPerformAuditQueryError(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	obsvReqWriteC := make(chan *gossipv1.ObservationRequest, 10)
	acctChan := make(chan *common.MessagePublication, MsgChannelCapacity)

	payload := buildMockTransferPayloadBytes(1, vaa.ChainIDEthereum, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", vaa.ChainIDPolygon, "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", 1.25)
	pe := createTestPendingEntry(vaa.ChainIDEthereum, testEmitterAddr, testSequence, testTxHash, payload)

	// Mock: all_pending_transfers returns error
	wormchainConn := &AuditMockWormchainConn{
		allPendingTransfersErr: errors.New("query failed"),
	}

	acct := newAccountantForAuditTest(t, logger, ctx, obsvReqWriteC, acctChan, wormchainConn)

	acct.pendingTransfersLock.Lock()
	acct.pendingTransfers[pe.msgId] = pe
	acct.pendingTransfersLock.Unlock()

	tmpMap := map[string][]*pendingEntry{
		pe.makeAuditKey(): {pe},
	}

	// Run the audit - should not panic
	acct.performAudit(ctx, tmpMap, wormchainConn, "test-contract")

	// Verify: entry should still be in pendingTransfers (not removed due to error)
	acct.pendingTransfersLock.Lock()
	_, exists := acct.pendingTransfers[pe.msgId]
	acct.pendingTransfersLock.Unlock()
	assert.True(t, exists, "expected entry to remain in pendingTransfers")

	// Verify: nothing should have been sent to channels
	assert.Equal(t, 0, len(acct.subChan), "expected no messages in subChan")
	assert.Equal(t, 0, len(obsvReqWriteC), "expected no reobservation requests")

	// Verify: tmpMap still contains the entry because the query failed and
	// performAudit returned early, so neither phase 1 nor phase 2 processed it.
	assert.Equal(t, 1, len(tmpMap), "expected tmpMap to still contain the entry")

	// Drain channels
	drainMsgChannel(acct.subChan)
	drainObsvReqChannel(obsvReqWriteC)
}
