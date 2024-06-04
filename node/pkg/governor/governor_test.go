//nolint:unparam // we exclude the unparam linter because there are many cases here where parameters are static
package governor

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// This is so we can have consistent config data for unit tests.
func (gov *ChainGovernor) initConfigForTest(
	emitterChainID vaa.ChainID,
	emitterAddr vaa.Address,
	dailyLimit uint64,
	tokenChainID vaa.ChainID,
	tokenAddr vaa.Address,
	tokenSymbol string,
	tokenPrice float64,
	tokenDecimals int64,
) {
	gov.chains[emitterChainID] = &chainEntry{emitterChainId: emitterChainID, emitterAddr: emitterAddr, dailyLimit: dailyLimit}

	price := big.NewFloat(tokenPrice)
	decimalsFloat := big.NewFloat(math.Pow(10.0, float64(tokenDecimals)))
	decimals, _ := decimalsFloat.Int(nil)
	key := tokenKey{chain: tokenChainID, addr: tokenAddr}
	gov.tokens[key] = &tokenEntry{price: price, decimals: decimals, symbol: tokenSymbol, token: key}
}

func (gov *ChainGovernor) setDayLengthInMinutes(min int) {
	gov.dayLengthInMinutes = min
}

func (gov *ChainGovernor) setChainForTesting(
	emitterChainId vaa.ChainID,
	emitterAddrStr string,
	dailyLimit uint64,
	bigTransactionSize uint64,
) error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	emitterAddr, err := vaa.StringToAddress(emitterAddrStr)
	if err != nil {
		return err
	}

	ce := &chainEntry{
		emitterChainId:          emitterChainId,
		emitterAddr:             emitterAddr,
		dailyLimit:              dailyLimit,
		bigTransactionSize:      bigTransactionSize,
		checkForBigTransactions: bigTransactionSize != 0,
	}

	gov.chains[emitterChainId] = ce
	return nil
}

func (gov *ChainGovernor) setTokenForTesting(
	tokenChainID vaa.ChainID,
	tokenAddrStr string,
	symbol string,
	price float64,
) error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	tokenAddr, err := vaa.StringToAddress(tokenAddrStr)
	if err != nil {
		return err
	}

	bigPrice := big.NewFloat(price)
	decimalsFloat := big.NewFloat(math.Pow(10.0, float64(8)))
	decimals, _ := decimalsFloat.Int(nil)

	key := tokenKey{chain: tokenChainID, addr: tokenAddr}
	te := &tokenEntry{cfgPrice: bigPrice, price: bigPrice, decimals: decimals, symbol: symbol, coinGeckoId: symbol, token: key}
	gov.tokens[key] = te
	cge, cgExists := gov.tokensByCoinGeckoId[te.coinGeckoId]
	if !cgExists {
		gov.tokensByCoinGeckoId[te.coinGeckoId] = []*tokenEntry{te}
	} else {
		cge = append(cge, te)
		gov.tokensByCoinGeckoId[te.coinGeckoId] = cge
	}
	return nil
}

// getStatsForAllChains sums the number of transfers, value of all transfers, number of pending transfers,
// and the value of the pending transfers.
// Note that 'flow cancel transfers' are not included and therefore the values returned by this function may not
// match the Governor usage.
func (gov *ChainGovernor) getStatsForAllChains() (numTrans int, valueTrans uint64, numPending int, valuePending uint64) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		numTrans += len(ce.transfers)
		for _, te := range ce.transfers {
			valueTrans += te.dbTransfer.Value
		}

		numPending += len(ce.pending)
		for _, pe := range ce.pending {
			value, _ := computeValue(pe.amount, pe.token)
			valuePending += value
		}
	}

	return
}

func checkTargetOnReleasedIsSet(t *testing.T, toBePublished []*common.MessagePublication, targetChain vaa.ChainID, targetAddressStr string) {
	require.NotEqual(t, 0, len(toBePublished))
	toAddr, err := vaa.StringToAddress(targetAddressStr)
	require.NoError(t, err)
	for _, msg := range toBePublished {
		payload, err := vaa.DecodeTransferPayloadHdr(msg.Payload)
		require.NoError(t, err)
		assert.Equal(t, targetChain, payload.TargetChain)
		assert.Equal(t, toAddr, payload.TargetAddress)
	}
}

func TestTrimEmptyTransfers(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	var transfers []transfer
	sum, updatedTransfers, err := gov.TrimAndSumValue(transfers, now)
	require.NoError(t, err)
	assert.Equal(t, int64(0), sum)
	assert.Equal(t, 0, len(updatedTransfers))
}

func TestSumAllFromToday(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	var transfers []transfer
	transferTime, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 11:00am (CST)")
	require.NoError(t, err)
	dbTransfer := &db.Transfer{Value: 125000, Timestamp: transferTime}
	transfer, err := newTransferFromDbTransfer(dbTransfer)
	require.NoError(t, err)
	transfers = append(transfers, transfer)
	sum, updatedTransfers, err := gov.TrimAndSumValue(transfers, now.Add(-time.Hour*24))
	require.NoError(t, err)
	assert.Equal(t, uint64(125000), uint64(sum))
	assert.Equal(t, 1, len(updatedTransfers))
}

func TestSumWithFlowCancelling(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Choose a hard-coded value from the Flow Cancel Token List
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var originChain vaa.ChainID = 2
	var originAddress vaa.Address
	originAddress, err = vaa.StringToAddress("000000000000000000000000bcca60bb61934080951369a648fb03df4f96263c")
	require.NoError(t, err)

	// Ensure asset is registered in the governor and can flow cancel
	key := tokenKey{originChain, originAddress}
	assert.True(t, gov.tokens[key].flowCancels)

	now, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	var chainEntryTransfers []transfer
	transferTime, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	// Set up values and governor limit
	emitterTransferValue := uint64(125000)
	flowCancelValue := uint64(100000)

	emitterLimit := emitterTransferValue * 2 // make sure the limit always exceeds the transfer value
	emitterChainId := 1

	// Setup transfers
	// - Transfer from emitter: we only care about Value
	// - Transfer that flow cancels: Transfer must be a valid entry from FlowCancelTokenList()  (based on origin chain and origin address)
	//				 and the destination chain must be the same as the emitter chain
	outgoingDbTransfer := &db.Transfer{Value: emitterTransferValue, Timestamp: transferTime}
	outgoingTransfer, err := newTransferFromDbTransfer(outgoingDbTransfer)
	require.NoError(t, err)

	// Flow cancelling transfer
	incomingDbTransfer := &db.Transfer{
		OriginChain:   originChain,
		OriginAddress: originAddress,
		TargetChain:   vaa.ChainID(emitterChainId), // emitter
		Value:         flowCancelValue,
		Timestamp:     transferTime,
	}

	chainEntryTransfers = append(chainEntryTransfers, outgoingTransfer)

	// Populate chainEntry and ChainGovernor
	emitter := &chainEntry{
		transfers:      chainEntryTransfers,
		emitterChainId: vaa.ChainID(emitterChainId),
		dailyLimit:     emitterLimit,
	}

	err = emitter.addFlowCancelTransferFromDbTransfer(incomingDbTransfer)
	require.NoError(t, err)

	gov.chains[emitter.emitterChainId] = emitter

	// Sanity check: ensure that there are transfers in the chainEntry
	expectedNumTransfers := 2
	_, transfers, err := gov.TrimAndSumValue(emitter.transfers, now)
	require.NoError(t, err)
	assert.Equal(t, expectedNumTransfers, len(transfers))

	// Calculate Governor Usage for emitter, including flow cancelling.
	usage, err := gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.NoError(t, err)
	difference := uint64(25000) // emitterTransferValue - flowCancelTransferValue
	assert.Equal(t, difference, usage)
}

// Flow cancelling transfers are subtracted from the overall sum of all transfers from a given
// emitter chain. Since we are working with uint64 values, ensure that there is no underflow.
// When the sum of all flow cancelling transfers is greater than emitted transfers for a chain,
// the expected result is that the resulting Governor Usage equals 0 (and not a negative number
// or a very large underflow result).
// Also, the function should not return an error in this case.
func TestFlowCancelCannotUnderflow(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Set-up asset to be used in the test
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var originChain vaa.ChainID = 2
	var originAddress vaa.Address
	originAddress, err = vaa.StringToAddress("000000000000000000000000bcca60bb61934080951369a648fb03df4f96263c")
	require.NoError(t, err)

	// Ensure asset is registered in the governor and can flow cancel
	key := tokenKey{originChain, originAddress}
	assert.True(t, gov.tokens[key].flowCancels)

	now, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	var transfers_from_emitter []transfer
	transferTime, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	// Set up values and governor limit
	emitterTransferValue := uint64(100000)
	flowCancelValue := emitterTransferValue + 25000 // make sure this value is higher than `emitterTransferValue`

	emitterLimit := emitterTransferValue * 2 // make sure the limit always exceeds the transfer value
	emitterChainId := 1

	// Setup transfers
	// - Transfer from emitter: we only care about Value
	// - Transfer that flow cancels: Transfer must be a valid entry from FlowCancelTokenList()  (based on origin chain and origin address)
	//				 and the destination chain must be the same as the emitter chain
	emitterDbTransfer := &db.Transfer{Value: emitterTransferValue, Timestamp: transferTime}
	emitterTransfer, err := newTransferFromDbTransfer(emitterDbTransfer)
	require.NoError(t, err)
	transfers_from_emitter = append(transfers_from_emitter, emitterTransfer)

	flowCancelDbTransfer := &db.Transfer{
		OriginChain:   originChain,
		OriginAddress: originAddress,
		TargetChain:   vaa.ChainID(emitterChainId), // emitter
		Value:         flowCancelValue,
		Timestamp:     transferTime,
	}

	// Populate chainEntrys and ChainGovernor
	emitter := &chainEntry{
		transfers:      transfers_from_emitter,
		emitterChainId: vaa.ChainID(emitterChainId),
		dailyLimit:     emitterLimit,
	}
	err = emitter.addFlowCancelTransferFromDbTransfer(flowCancelDbTransfer)
	require.NoError(t, err)

	gov.chains[emitter.emitterChainId] = emitter

	expectedNumTransfers := 2
	_, transfers, err := gov.TrimAndSumValue(emitter.transfers, now)
	require.NoError(t, err)
	assert.Equal(t, expectedNumTransfers, len(transfers))

	// Calculate Governor Usage for emitter, including flow cancelling
	// Should be zero when flow cancel transfer values exceed emitted transfer values.
	usage, err := gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.NoError(t, err)
	assert.Zero(t, usage)
}

// Simulate a case where the total sum of transfers for a chain in a 24 hour period exceeds
// the configured Governor limit. This should never happen, so we make sure that an error
// is returned if the system is in this state
func TestInvariantGovernorLimit(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	var transfers_from_emitter []transfer
	transferTime, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	emitterTransferValue := uint64(125000)

	emitterLimit := emitterTransferValue * 20
	emitterChainId := 1

	// Create a lot of transfers. Their total value should exceed `emitterLimit`
	for i := 0; i < 25; i++ {
		transfer, err := newTransferFromDbTransfer(&db.Transfer{Value: emitterTransferValue, Timestamp: transferTime})
		require.NoError(t, err)
		transfers_from_emitter = append(
			transfers_from_emitter,
			transfer,
		)
	}

	// Populate chainEntry and ChainGovernor
	emitter := &chainEntry{
		transfers:      transfers_from_emitter,
		emitterChainId: vaa.ChainID(emitterChainId),
		dailyLimit:     emitterLimit,
	}
	gov.chains[emitter.emitterChainId] = emitter

	// XXX: sanity check
	expectedNumTransfers := 25
	sum, transfers, err := gov.TrimAndSumValue(emitter.transfers, now)
	require.NoError(t, err)
	assert.Equal(t, expectedNumTransfers, len(transfers))
	assert.NotZero(t, sum)

	// Make sure we trigger the Invariant
	usage, err := gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.ErrorContains(t, err, "invariant violation: calculated sum")
	assert.Zero(t, usage)
}

func TestTrimOneOfTwoTransfers(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	var transfers []transfer

	// The first transfer should be expired.
	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:59am (CST)")
	require.NoError(t, err)
	dbTransfer := &db.Transfer{Value: 125000, Timestamp: transferTime1}
	transfer, err := newTransferFromDbTransfer(dbTransfer)
	require.NoError(t, err)
	transfers = append(transfers, transfer)

	// But the second should not.
	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 1:00pm (CST)")
	require.NoError(t, err)
	dbTransfer = &db.Transfer{Value: 225000, Timestamp: transferTime2}
	transfer2, err := newTransferFromDbTransfer(dbTransfer)
	require.NoError(t, err)
	transfers = append(transfers, transfer2)
	assert.Equal(t, 2, len(transfers))

	sum, updatedTransfers, err := gov.TrimAndSumValue(transfers, now.Add(-time.Hour*24))
	require.NoError(t, err)
	assert.Equal(t, 1, len(updatedTransfers))
	assert.Equal(t, uint64(225000), uint64(sum))
}

func TestTrimSeveralTransfers(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	var transfers []transfer

	// The first two transfers should be expired.
	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 10:00am (CST)")
	require.NoError(t, err)
	dbTransfer1 := &db.Transfer{Value: 125000, Timestamp: transferTime1}
	transfer1, err := newTransferFromDbTransfer(dbTransfer1)
	require.NoError(t, err)
	transfers = append(transfers, transfer1)

	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:00am (CST)")
	require.NoError(t, err)
	dbTransfer2 := &db.Transfer{Value: 135000, Timestamp: transferTime2}
	transfer2, err := newTransferFromDbTransfer(dbTransfer2)
	require.NoError(t, err)
	transfers = append(transfers, transfer2)

	// But the next three should not.
	transferTime3, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 1:00pm (CST)")
	require.NoError(t, err)
	dbTransfer3 := &db.Transfer{Value: 145000, Timestamp: transferTime3}
	transfer3, err := newTransferFromDbTransfer(dbTransfer3)
	require.NoError(t, err)
	transfers = append(transfers, transfer3)

	transferTime4, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 2:00pm (CST)")
	require.NoError(t, err)
	dbTransfer4 := &db.Transfer{Value: 155000, Timestamp: transferTime4}
	transfer4, err := newTransferFromDbTransfer(dbTransfer4)
	require.NoError(t, err)
	transfers = append(transfers, transfer4)

	transferTime5, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 2:00pm (CST)")
	require.NoError(t, err)
	dbTransfer5 := &db.Transfer{Value: 165000, Timestamp: transferTime5}
	transfer5, err := newTransferFromDbTransfer(dbTransfer5)
	require.NoError(t, err)
	transfers = append(transfers, transfer5)

	assert.Equal(t, 5, len(transfers))

	sum, updatedTransfers, err := gov.TrimAndSumValue(transfers, now.Add(-time.Hour*24))
	require.NoError(t, err)
	assert.Equal(t, 3, len(updatedTransfers))
	assert.Equal(t, uint64(465000), uint64(sum))
}

func TestTrimmingAllTransfersShouldReturnZero(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	var transfers []transfer

	transferTime1, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:00am (CST)")
	require.NoError(t, err)
	dbTransfer1 := &db.Transfer{Value: 125000, Timestamp: transferTime1}
	transfer1, err := newTransferFromDbTransfer(dbTransfer1)
	require.NoError(t, err)
	transfers = append(transfers, transfer1)

	transferTime2, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "May 31, 2022 at 11:45am (CST)")
	require.NoError(t, err)
	dbTransfer2 := &db.Transfer{Value: 125000, Timestamp: transferTime2}
	transfer2, err := newTransferFromDbTransfer(dbTransfer2)
	require.NoError(t, err)
	transfers = append(transfers, transfer2)

	assert.Equal(t, 2, len(transfers))

	sum, updatedTransfers, err := gov.TrimAndSumValue(transfers, now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(updatedTransfers))
	assert.Equal(t, int64(0), sum)
}

func newChainGovernorForTest(ctx context.Context) (*ChainGovernor, error) {
	return newChainGovernorForTestWithLogger(ctx, zap.NewNop())
}

func newChainGovernorForTestWithLogger(ctx context.Context, logger *zap.Logger) (*ChainGovernor, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx is nil")
	}

	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, common.GoTest)

	err := gov.Run(ctx)
	if err != nil {
		return gov, nil
	}

	emitterAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	if err != nil {
		return gov, nil
	}

	tokenAddr, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	if err != nil {
		return gov, nil
	}

	gov.initConfigForTest(
		vaa.ChainIDEthereum,
		emitterAddr,
		1000000,
		vaa.ChainIDEthereum,
		tokenAddr,
		"WETH",
		1774.62,
		8,
	)

	return gov, nil
}

// Converts a string into a go-ethereum Hash object used as test input.
func hashFromString(str string) eth_common.Hash {
	if (len(str) > 2) && (str[0] == '0') && (str[1] == 'x') {
		str = str[2:]
	}

	return eth_common.HexToHash(str)
}

func TestVaaForUninterestingEmitterChain(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	payload := []byte{1, 97, 97, 97, 97, 97}

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

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
}

func TestVaaForUninterestingEmitterAddress(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	payload := []byte{1, 97, 97, 97, 97, 97}

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

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 0, len(gov.msgsSeen))
}

func TestVaaForUninterestingPayloadType(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	payload := []byte{2, 97, 97, 97, 97, 97}

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

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 0, len(gov.msgsSeen))
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

func TestBuidMockTransferPayload(t *testing.T) {
	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	payload, err := vaa.DecodeTransferPayloadHdr(payloadBytes)
	require.NoError(t, err)

	expectedTokenAddr, err := vaa.StringToAddress(tokenAddrStr)
	require.NoError(t, err)

	expectedToAddr, err := vaa.StringToAddress(toAddrStr)
	require.NoError(t, err)

	expected := &vaa.TransferPayloadHdr{
		Type:          1,
		Amount:        big.NewInt(125000000),
		OriginAddress: expectedTokenAddr,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: expectedToAddr,
		TargetChain:   vaa.ChainIDPolygon,
	}

	assert.Equal(t, expected, payload)
}

func TestVaaForUninterestingToken(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	uninterestingTokenAddrStr := "0x42"
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	payloadBytes := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		uninterestingTokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	tokenBridgeAddr, _ := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes,
	}

	canPost, err := gov.ProcessMsgForTime(&msg, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 0, len(gov.msgsSeen))
}

func TestTransfersUpToAndOverTheLimit(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 0)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	// The first two transfers should be accepted.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	canPost, err := gov.ProcessMsgForTime(&msg1, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(2218), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	canPost, err = gov.ProcessMsgForTime(&msg2, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(4436), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))

	// But the third one should be queued up.
	payloadBytes2 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1250,
	)

	msg3 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(3),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes2,
	}

	canPost, err = gov.ProcessMsgForTime(&msg3, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(4436), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(2218274), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))

	// But a small one should still go through.
	msg4 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(4),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	canPost, err = gov.ProcessMsgForTime(&msg4, time.Now())
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 3, numTrans)
	assert.Equal(t, uint64(4436+2218), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(2218274), valuePending)
	assert.Equal(t, 4, len(gov.msgsSeen))
}

func TestPendingTransferBeingReleased(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 0)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// The first VAA should be accepted.
	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		270,
	)

	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(479147), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// And so should the second.
	payloadBytes2 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		275,
	)

	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes2,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 6:00pm (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))

	// But the third one should be queued up.
	payloadBytes3 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		280,
	)

	msg3 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes3,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 2:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg3, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(496893), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))

	// And so should the fourth one.
	payloadBytes4 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		300,
	)

	msg4 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes4,
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 8:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg4, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(496893+532385), valuePending)
	assert.Equal(t, 4, len(gov.msgsSeen))

	// If we check pending before noon, nothing should happen.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 9:00am (CST)")
	toBePublished, err := gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(496893+532385), valuePending)
	assert.Equal(t, 4, len(gov.msgsSeen))

	// But at 3pm, the first one should drop off and the first queued one should get posted.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 3:00pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(toBePublished))
	checkTargetOnReleasedIsSet(t, toBePublished, vaa.ChainIDPolygon, toAddrStr)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(488020+496893), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(532385), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))
}

func TestSmallerPendingTransfersAfterBigOneShouldGetReleased(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 0)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// The first VAA should be accepted.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			270,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(479147), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// And so should the second.
	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			275,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 6:00pm (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))

	// But the third, big one should be queued up.
	msg3 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			500,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 2:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg3, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(887309), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))

	// A fourth, smaller, but still too big one, should get enqueued.
	msg4 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			100,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 8:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg4, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(887309+177461), valuePending)
	assert.Equal(t, 4, len(gov.msgsSeen))

	// A fifth, smaller, but still too big one, should also get enqueued.
	msg5 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			101,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 8:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg5, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 3, numPending)
	assert.Equal(t, uint64(887309+177461+179236), valuePending)
	assert.Equal(t, 5, len(gov.msgsSeen))

	// A sixth, big one should also get enqueued.
	msg6 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			501,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 2:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg6, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 4, numPending)
	assert.Equal(t, uint64(887309+177461+179236+889084), valuePending)
	assert.Equal(t, 6, len(gov.msgsSeen))

	// If we check pending before noon, nothing should happen.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 9:00am (CST)")
	toBePublished, err := gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(479147+488020), valueTrans)
	assert.Equal(t, 4, numPending)
	assert.Equal(t, uint64(887309+177461+179236+889084), valuePending)
	assert.Equal(t, 6, len(gov.msgsSeen))

	// But at 3pm, the first one should drop off. This should result in the second and third, smaller pending ones being posted, but not the two big ones.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 3:00pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 2, len(toBePublished))
	checkTargetOnReleasedIsSet(t, toBePublished, vaa.ChainIDPolygon, toAddrStr)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 3, numTrans)
	assert.Equal(t, uint64(488020+177461+179236), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(887309+889084), valuePending)
	assert.Equal(t, 5, len(gov.msgsSeen))
}

func TestMainnetConfigIsValid(t *testing.T) {
	logger := zap.NewNop()
	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, common.GoTest)

	gov.env = common.TestNet
	err := gov.initConfig()
	require.NoError(t, err)
}

func TestTestnetConfigIsValid(t *testing.T) {
	logger := zap.NewNop()
	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, common.GoTest)

	gov.env = common.TestNet
	err := gov.initConfig()
	require.NoError(t, err)
}

func TestLargeTransactionGetsEnqueuedAndReleasedWhenTheTimerExpires(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// The first small transfer should be accepted.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			50,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(88730), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// And so should the second.
	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			50,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 6:00pm (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(88730+88730), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))

	// But the third big one should get enqueued.
	msg3 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(3),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			100,
		),
	}

	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 2:00am (CST)")
	canPost, err = gov.ProcessMsgForTime(&msg3, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(88730+88730), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))

	// If we check pending before noon, nothing should happen.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 9:00am (CST)")
	toBePublished, err := gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(88730+88730), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)
	assert.Equal(t, 3, len(gov.msgsSeen))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(88730+88730), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)

	// But just after noon, the first one should drop off. The big pending one should not be affected.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 12:01pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(88730), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))

	// And Just after 6pm, the second one should drop off. The big pending one should still not be affected.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 6:01pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)

	// 23 hours after the big transaction is enqueued, it should still be there.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 3, 2022 at 1:01am (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// But then the operator resets the release time.
	_, err = gov.resetReleaseTimerForTime(msg3.MessageIDString(), now)
	require.NoError(t, err)

	// So now, 12 hours later the big transaction is enqueued, it still won't get released.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 3, 2022 at 1:00pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(177461), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// But finally, a full 24hrs, it should get released.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 4, 2022 at 1:01am (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(toBePublished))
	checkTargetOnReleasedIsSet(t, toBePublished, vaa.ChainIDPolygon, toAddrStr)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	require.NoError(t, err)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 0, len(gov.msgsSeen))

	// But the big transaction should not affect the daily notional.
	ce, exists := gov.chains[vaa.ChainIDEthereum]
	require.Equal(t, true, exists)
	valueTrans, err = sumValue(ce.transfers, now)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), valueTrans)
}

func TestSmallTransactionsGetReleasedWhenTheTimerExpires(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)

	// This configuration does not make sense for real, but allows for this test.
	// We are setting the big transfer size smaller than the daily limit, so we can
	// easily enqueue a transfer that is not considered big and confirm that it eventually
	// gets released after the release time passes.

	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 10000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// Submit a small transfer that will get enqueued due to the low daily limit.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			50,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(88730), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// If we check 23hrs later, nothing should happen.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 11:00am (CST)")
	toBePublished, err := gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 0, len(toBePublished))

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(88730), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// But after 24hrs, it should get released.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 12:01pm (CST)")
	toBePublished, err = gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(toBePublished))
	checkTargetOnReleasedIsSet(t, toBePublished, vaa.ChainIDPolygon, toAddrStr)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 0, len(gov.msgsSeen))
}

func TestIsBigTransfer(t *testing.T) {
	emitterAddr := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	bigTransactionSize := uint64(5_000_000)

	ce := chainEntry{
		emitterChainId:          vaa.ChainIDEthereum,
		emitterAddr:             emitterAddr,
		dailyLimit:              uint64(50_000_000),
		bigTransactionSize:      bigTransactionSize,
		checkForBigTransactions: bigTransactionSize != 0,
	}

	assert.Equal(t, false, ce.isBigTransfer(uint64(4_999_999)))
	assert.Equal(t, true, ce.isBigTransfer(uint64(5_000_000)))
	assert.Equal(t, true, ce.isBigTransfer(uint64(5_000_001)))
}

func TestTransferPayloadTooShort(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 0)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	payloadBytes1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	payloadBytes1 = payloadBytes1[0 : len(payloadBytes1)-1]

	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	// The low level method should return an error.
	_, err = gov.ProcessMsgForTime(&msg, time.Now())
	assert.EqualError(t, err, "buffer too short")
	assert.Equal(t, 0, len(gov.msgsSeen))

	// The higher level method should return false, saying we should not publish.
	canPost := gov.ProcessMsg(&msg)
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, len(gov.msgsSeen))
}

func TestDontReloadDuplicates(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	emitterAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	emitterAddr, err := vaa.StringToAddress(emitterAddrStr)
	require.NoError(t, err)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	tokenAddr, err := vaa.StringToAddress(tokenAddrStr)
	require.NoError(t, err)
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"

	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, emitterAddrStr, 1000000, 0)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, emitterAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 12:01pm (CST)")
	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))

	var xfers []*db.Transfer

	xfer1 := &db.Transfer{
		Timestamp:      startTime.Add(time.Minute * 5),
		Value:          uint64(1000),
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: emitterAddr,
		MsgID:          "2/" + emitterAddrStr + "/125",
		Hash:           "Hash1",
	}
	xfers = append(xfers, xfer1)

	xfer2 := &db.Transfer{
		Timestamp:      startTime.Add(time.Minute * 5),
		Value:          uint64(2000),
		OriginChain:    vaa.ChainIDEthereum,
		OriginAddress:  tokenAddr,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: emitterAddr,
		MsgID:          "2/" + emitterAddrStr + "/126",
		Hash:           "Hash2",
	}
	xfers = append(xfers, xfer2)

	// Add a duplicate of each transfer
	xfers = append(xfers, xfer1)
	xfers = append(xfers, xfer2)
	assert.Equal(t, 4, len(xfers))

	payload1 := buildMockTransferPayloadBytes(1,
		vaa.ChainIDEthereum,
		tokenAddrStr,
		vaa.ChainIDPolygon,
		toAddrStr,
		1.25,
	)

	var pendings []*db.PendingTransfer
	pending1 := &db.PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 24),
		Msg: common.MessagePublication{
			TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        time.Unix(int64(1654543099), 0),
			Nonce:            uint32(1),
			Sequence:         uint64(200),
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   emitterAddr,
			ConsistencyLevel: uint8(32),
			Payload:          payload1,
		},
	}
	pendings = append(pendings, pending1)

	pending2 := &db.PendingTransfer{
		ReleaseTime: now.Add(time.Hour * 24),
		Msg: common.MessagePublication{
			TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
			Timestamp:        time.Unix(int64(1654543099), 0),
			Nonce:            uint32(1),
			Sequence:         uint64(201),
			EmitterChain:     vaa.ChainIDEthereum,
			EmitterAddress:   emitterAddr,
			ConsistencyLevel: uint8(32),
			Payload:          payload1,
		},
	}
	pendings = append(pendings, pending2)

	// Add a duplicate of each pending transfer
	pendings = append(pendings, pending1)
	pendings = append(pendings, pending2)
	assert.Equal(t, 4, len(pendings))

	for _, p := range xfers {
		err := gov.reloadTransfer(p)
		require.NoError(t, err)
	}

	for _, p := range pendings {
		gov.reloadPendingTransfer(p)
	}

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(3000), valueTrans)
	assert.Equal(t, 2, numPending)
	assert.Equal(t, uint64(4436), valuePending)
}

func TestReobservationOfPublishedMsg(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// The first transfer should be accepted.
	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			50,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:10pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(88730), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// A reobservation of the same message should get published but should not affect the notional value.
	canPost, err = gov.ProcessMsgForTime(&msg, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(88730), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))
}

func TestReobservationOfEnqueued(t *testing.T) {
	// The duplicate should not get published and not get enqueued again.
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// A big transfer should get enqueued.
	msg := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			5000,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:10pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(8_873_099), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// A reobservation of the same message should not get published and should not get enqueued again.
	canPost, err = gov.ProcessMsgForTime(&msg, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, false, canPost)
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, uint64(0), valueTrans)
	assert.Equal(t, 1, numPending)
	assert.Equal(t, uint64(8_873_099), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))
}

func TestReusedMsgIdWithDifferentPayloadGetsProcessed(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// The first transfer should be accepted.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			50,
		),
	}

	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:10pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 1, numTrans)
	assert.Equal(t, uint64(88730), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// A second message with the same msgId but a different payload should also get published and apply to the notional value.
	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			5,
		),
	}

	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChains()
	assert.Equal(t, true, canPost)
	assert.Equal(t, 2, numTrans)
	assert.Equal(t, uint64(97603), valueTrans)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 2, len(gov.msgsSeen))
}

func getIdsFromCoinGeckoQuery(t *testing.T, query string) []string {
	unescaped, err := url.QueryUnescape(query)
	require.NoError(t, err)

	fields := strings.Split(unescaped, "?")
	require.Equal(t, 2, len(fields))

	u, err := url.ParseQuery(fields[1])
	require.NoError(t, err)

	idField, exists := u["ids"]
	require.Equal(t, true, exists)
	require.Equal(t, 1, len(idField))

	return strings.Split(idField[0], ",")
}

func TestCoinGeckoQueries(t *testing.T) {
	type testCase struct {
		desc            string
		numIds          int
		chunkSize       int
		expectedQueries int
	}

	tests := []testCase{
		{numIds: 0, chunkSize: 100, expectedQueries: 0, desc: "Zero queries"},
		{numIds: 42, chunkSize: 100, expectedQueries: 1, desc: "Easily fits in one"},
		{numIds: 100, chunkSize: 100, expectedQueries: 1, desc: "Exactly fits in one"},
		{numIds: 242, chunkSize: 207, expectedQueries: 2, desc: "Easily fits in two"},
		{numIds: 414, chunkSize: 207, expectedQueries: 2, desc: "Exactly fits in two"},
		{numIds: 5001, chunkSize: 207, expectedQueries: 25, desc: "A bunch of queries"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ids := make([]string, tc.numIds)
			for idx := 0; idx < tc.numIds; idx++ {
				ids[idx] = fmt.Sprintf("id%d", idx)
			}

			queries := createCoinGeckoQueries(ids, tc.chunkSize)
			require.Equal(t, tc.expectedQueries, len(queries))

			results := make(map[string]string)
			for _, query := range queries {
				idsInQuery := getIdsFromCoinGeckoQuery(t, query)
				require.GreaterOrEqual(t, tc.chunkSize, len(idsInQuery))
				for _, id := range idsInQuery {
					results[id] = id
				}
			}

			require.Equal(t, tc.numIds, len(results))

			for _, id := range ids {
				if _, exists := results[id]; !exists {
					assert.Equal(t, "id not found in query", id)
				}
				delete(results, id)
			}
			if len(results) != 0 {
				for id := range results {
					assert.Equal(t, "bogus id created by query", id)
				}
			}
		})
	}
}

// setupLogsCapture is a helper function for making a zap logger/observer combination for testing that certain logs have been made
func setupLogsCapture(t testing.TB, options ...zap.Option) (*zap.Logger, *observer.ObservedLogs) {
	t.Helper()
	observedCore, observedLogs := observer.New(zap.InfoLevel)
	consoleLogger := zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel))
	parentLogger := zap.New(zapcore.NewTee(observedCore, consoleLogger.Core()), options...)
	return parentLogger, observedLogs
}

func TestPendingTransferWithBadPayloadGetsDroppedNotReleased(t *testing.T) {
	ctx := context.Background()
	zapLogger, zapObserver := setupLogsCapture(t)
	gov, err := newChainGovernorForTestWithLogger(ctx, zapLogger)
	require.NoError(t, err)
	require.NotNil(t, gov)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)

	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 10000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62)
	require.NoError(t, err)

	// Create two big transactions.
	msg1 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			5000,
		),
	}

	msg2 := common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum,
			tokenAddrStr,
			vaa.ChainIDPolygon,
			toAddrStr,
			5000,
		),
	}

	// Post the two big transfers and and verify they get enqueued.
	now, _ := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	canPost, err := gov.ProcessMsgForTime(&msg1, now)
	require.NoError(t, err)
	assert.Equal(t, false, canPost)

	canPost, err = gov.ProcessMsgForTime(&msg2, now)
	require.NoError(t, err)
	assert.Equal(t, false, canPost)

	numTrans, _, numPending, _ := gov.getStatsForAllChains()
	assert.Equal(t, 2, len(gov.msgsSeen))
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, 2, numPending)

	// Corrupt the payload of msg2 so that when we try to release it, it will get dropped.
	gov.mutex.Lock()
	ce, exists := gov.chains[vaa.ChainIDEthereum]
	require.True(t, exists)
	require.Equal(t, 2, len(ce.pending))
	require.Equal(t, "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/2", ce.pending[1].dbData.Msg.MessageIDString())
	ce.pending[1].dbData.Msg.Payload = nil
	gov.mutex.Unlock()

	// After 24hrs, msg1 should get released but msg2 should get dropped.
	now, _ = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 12:01pm (CST)")
	toBePublished, err := gov.CheckPendingForTime(now)
	require.NoError(t, err)
	assert.Equal(t, 1, len(toBePublished))
	checkTargetOnReleasedIsSet(t, toBePublished, vaa.ChainIDPolygon, toAddrStr)
	assert.Equal(t, "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1", toBePublished[0].MessageIDString())

	// Verify that we got the expected error in the logs.
	loggedEntries := zapObserver.FilterMessage("failed to decode payload for pending VAA, dropping it").All()
	require.Equal(t, 1, len(loggedEntries))

	foundIt := false
	for _, f := range loggedEntries[0].Context {
		if f.Key == "msgID" && f.String == "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/2" {
			foundIt = true
		}
	}
	assert.True(t, foundIt)

	// Verify that the message is no longer pending.
	gov.mutex.Lock()
	ce, exists = gov.chains[vaa.ChainIDEthereum]
	require.True(t, exists)
	assert.Equal(t, 0, len(ce.pending))
	gov.mutex.Unlock()

	// Neither one should be in the map of messages seen.
	_, exists = gov.msgsSeen[gov.HashFromMsg(&msg1)]
	assert.False(t, exists)
	_, exists = gov.msgsSeen[gov.HashFromMsg(&msg2)]
	assert.False(t, exists)
}

func TestCheckedAddUint64HappyPath(t *testing.T) {
	// Both non-zero
	x := uint64(1000)
	y := uint64(337)
	sum, err := CheckedAddUint64(x, y)
	require.NoError(t, err)
	assert.Equal(t, uint64(1337), sum)

	// x is zero
	x = 0
	y = 2000
	sum, err = CheckedAddUint64(x, y)
	require.NoError(t, err)
	assert.Equal(t, uint64(2000), sum)

	// y is zero
	x = 3000
	y = 0
	sum, err = CheckedAddUint64(x, y)
	require.NoError(t, err)
	assert.Equal(t, uint64(3000), sum)
}

func TestCheckedAddInt64HappyPath(t *testing.T) {
	// Two positive numbers
	x := int64(1000)
	y := int64(337)
	sum, err := CheckedAddInt64(x, y)
	require.NoError(t, err)
	assert.Equal(t, int64(1337), sum)

	// One positive, one negative
	x = 100
	y = -1000
	sum, err = CheckedAddInt64(x, y)
	require.NoError(t, err)
	assert.Equal(t, int64(-900), sum)

	// Both negative
	x = -100
	y = -1000
	sum, err = CheckedAddInt64(x, y)
	require.NoError(t, err)
	assert.Equal(t, int64(-1100), sum)

	// x is zero
	x = 0
	y = 2000
	sum, err = CheckedAddInt64(x, y)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), sum)

	// y is zero
	x = 3000
	y = 0
	sum, err = CheckedAddInt64(x, y)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), sum)
}

func TestCheckedAddUint64ReturnsErrorOnOverflow(t *testing.T) {
	// Return error on overflow
	sum, err := CheckedAddUint64(math.MaxUint64, 1)
	require.Error(t, err)
	assert.Equal(t, uint64(0), sum)
}

func TestCheckedAddInt64ReturnsErrorOnOverflow(t *testing.T) {
	// Return error on overflow
	sum, err := CheckedAddInt64(math.MaxInt64, 1)
	require.Error(t, err)
	assert.Equal(t, int64(0), sum)
}

func TestCheckedAddInt64ReturnsErrorOnUnderflow(t *testing.T) {
	// Return error on underflow
	sum, err := CheckedAddInt64(math.MinInt64, -1)
	require.Error(t, err)
	assert.Equal(t, int64(0), sum)
}
