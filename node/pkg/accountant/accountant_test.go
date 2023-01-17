package accountant

import (
	"context"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	enforceAccountant     = true
	dontEnforceAccountant = false
)

func newAccountantForTest(
	t *testing.T,
	ctx context.Context,
	accountantCheckEnabled bool,
	acctWriteC chan<- *common.MessagePublication,
) *Accountant {
	logger := zap.NewNop()
	var db db.MockAccountantDB

	gk := devnet.InsecureDeterministicEcdsaKeyByIndex(ethCrypto.S256(), uint64(0))

	gst := common.NewGuardianSetState(nil)
	gs := &common.GuardianSet{}
	gst.Set(gs)

	acct := NewAccountant(
		ctx,
		logger,
		&db,
		"0xdeadbeef", // accountantContract
		"none",       // accountantWS
		nil,          // wormchainConn
		accountantCheckEnabled,
		gk,
		gst,
		acctWriteC,
		GoTestMode,
	)

	err := acct.Start(ctx)
	require.NoError(t, err)
	return acct
}

// Converts a string into a go-ethereum Hash object used as test input.
func hashFromString(str string) ethCommon.Hash {
	if (len(str) > 2) && (str[0] == '0') && (str[1] == 'x') {
		str = str[2:]
	}

	return ethCommon.HexToHash(str)
}

// Note this method assumes 18 decimals for the amount.
func buildMockTransferPayloadBytes(
	t uint8,
	tokenChainID vaa.ChainID,
	tokenAddrStr string,
	toChainID vaa.ChainID,
	toAddrStr string,
	amtFloat float64,
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
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, ctx, enforceAccountant, acctChan)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	var payload = []byte{1, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, ctx, enforceAccountant, acctChan)
	require.NotNil(t, acct)

	emitterAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	var payload = []byte{2, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, ctx, dontEnforceAccountant, acctChan)
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
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	acct.publishTransfer(pe)
	assert.Equal(t, 0, len(acct.msgChan))
	assert.Equal(t, 0, len(acct.pendingTransfers))
}

func TestInterestingTransferShouldBeBlockedWhenEnforcingAccountant(t *testing.T) {
	ctx := context.Background()
	acctChan := make(chan *common.MessagePublication, 10)
	acct := newAccountantForTest(t, ctx, enforceAccountant, acctChan)
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
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	acct.publishTransfer(pe)
	assert.Equal(t, 1, len(acct.msgChan))
	assert.Equal(t, 0, len(acct.pendingTransfers))
}
