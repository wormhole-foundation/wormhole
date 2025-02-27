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

func (gov *ChainGovernor) setDayLengthInMinutes(minimum int) {
	gov.dayLengthInMinutes = minimum
}

// Utility method: adds a new `chainEntry` to `gov`
// Supplying a bigTransactionSize of 0 will skip checks for big transactions.
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

// Utility method: adds a new `tokenEntry` to `gov`
func (gov *ChainGovernor) setTokenForTesting(
	tokenChainID vaa.ChainID,
	tokenAddrStr string,
	symbol string,
	price float64,
	flowCancels bool,
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
	te := &tokenEntry{cfgPrice: bigPrice, price: bigPrice, decimals: decimals, symbol: symbol, coinGeckoId: symbol, token: key, flowCancels: flowCancels}
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

// getStatsForAllChains but includes flow cancelling in its statistics. This results in different values for valueTrans
// TODO these functions can probably be merged together and a boolean can be passed if we want flow cancel results.
func (gov *ChainGovernor) getStatsForAllChainsCancelFlow() (numTrans int, valueTrans int64, numPending int, valuePending uint64) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		numTrans += len(ce.transfers)
		for _, te := range ce.transfers {
			valueTrans += te.value // Needs to be .value and not .dbTransfer.value because we want the SIGNED version of this.
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

// Make sure that the code doesn't panic if called with a nil chainEntry
func TestTrimAndSumValueForChainReturnsErrorForNilChainEntry(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 1, 2022 at 12:00pm (CST)")
	require.NoError(t, err)

	sum, err := gov.TrimAndSumValueForChain(nil, now)
	require.Error(t, err)
	assert.Equal(t, uint64(0), sum)
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
	assert.Equal(t, uint64(125000), uint64(sum)) // #nosec G115 -- If this overflowed the test would fail anyway
	assert.Equal(t, 1, len(updatedTransfers))
}

// Checks sum calculation for the flow cancel mechanism
func TestSumWithFlowCancelling(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Choose a hard-coded value from the Flow Cancel Token List
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var originChain vaa.ChainID = 1
	var originAddress vaa.Address
	originAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
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

func TestFlowCancelFeatureFlag(t *testing.T) {

	ctx := context.Background()
	var db db.MockGovernorDB
	gov := NewChainGovernor(zap.NewNop(), &db, common.GoTest, true, "")

	// Trigger the evaluation of the flow cancelling config
	err := gov.Run(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Test private bool
	assert.True(t, gov.flowCancelEnabled)
	// Test public getter
	assert.True(t, gov.IsFlowCancelEnabled())
	numFlowCancelling := 0
	for _, tokenEntry := range gov.tokens {
		if tokenEntry.flowCancels == true {
			numFlowCancelling++
		}
	}
	assert.NotZero(t, numFlowCancelling)

	// Disable flow cancelling
	gov = NewChainGovernor(zap.NewNop(), &db, common.GoTest, false, "")

	// Trigger the evaluation of the flow cancelling config
	err = gov.Run(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Test private bool
	assert.False(t, gov.flowCancelEnabled)
	// Test public getter
	assert.False(t, gov.IsFlowCancelEnabled())
	numFlowCancelling = 0
	for _, tokenEntry := range gov.tokens {
		if tokenEntry.flowCancels == true {
			numFlowCancelling++
		}
	}
	assert.Zero(t, numFlowCancelling)

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
	var originChain vaa.ChainID = 1
	var originAddress vaa.Address
	originAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
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

// We never expect this to occur when flow-cancelling is disabled. If flow-cancelling is enabled, there
// are some cases where the outgoing value exceeds the daily limit. Example: a large, incoming transfer
// of a flow-cancelling asset increases the Governor capacity beyond the daily limit. After 24h, that
// transfer is trimmed. This reduces the daily limit back to normal, but by this time more outgoing
// transfers have been emitted, causing the sum to exceed the daily limit.
func TestChainEntrySumExceedsDailyLimit(t *testing.T) {
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

	usage, err := gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.NoError(t, err)
	assert.Equal(t, emitterTransferValue*uint64(expectedNumTransfers), usage) // #nosec G115 -- If this overflowed the test would fail anyway
}

func TestTrimAndSumValueOverflowErrors(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)

	now, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	var transfers_from_emitter []transfer
	transferTime, err := time.Parse("2006-Jan-02", "2024-Feb-19")
	require.NoError(t, err)

	emitterChainId := vaa.ChainIDSolana

	transfer, err := newTransferFromDbTransfer(&db.Transfer{Value: math.MaxInt64, Timestamp: transferTime})
	require.NoError(t, err)
	transfer2, err := newTransferFromDbTransfer(&db.Transfer{Value: 1, Timestamp: transferTime})
	require.NoError(t, err)
	transfers_from_emitter = append(transfers_from_emitter, transfer, transfer2)

	// Populate chainEntry and ChainGovernor
	emitter := &chainEntry{
		transfers:      transfers_from_emitter,
		emitterChainId: vaa.ChainID(emitterChainId),
		dailyLimit:     10000,
	}
	gov.chains[emitter.emitterChainId] = emitter

	sum, _, err := gov.TrimAndSumValue(emitter.transfers, now.Add(-time.Hour*24))
	require.ErrorContains(t, err, "integer overflow")
	assert.Zero(t, sum)
	usage, err := gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.ErrorContains(t, err, "integer overflow")
	assert.Equal(t, uint64(10000), usage)

	// overwrite emitter (discard transfer added above)
	emitter = &chainEntry{
		emitterChainId: vaa.ChainID(emitterChainId),
		dailyLimit:     10000,
	}
	gov.chains[emitter.emitterChainId] = emitter

	// Now test underflow
	transfer3 := &db.Transfer{Value: math.MaxInt64, Timestamp: transferTime, TargetChain: vaa.ChainIDSolana}

	ce := gov.chains[emitter.emitterChainId]
	err = ce.addFlowCancelTransferFromDbTransfer(transfer3)
	require.NoError(t, err)
	err = ce.addFlowCancelTransferFromDbTransfer(transfer3)
	require.NoError(t, err)

	sum, _, err = gov.TrimAndSumValue(emitter.transfers, now.Add(-time.Hour*24))
	require.ErrorContains(t, err, "integer underflow")
	assert.Zero(t, sum)
	usage, err = gov.TrimAndSumValueForChain(emitter, now.Add(-time.Hour*24))
	require.ErrorContains(t, err, "integer underflow")
	assert.Equal(t, uint64(10000), usage)
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
	assert.Equal(t, uint64(225000), uint64(sum)) // #nosec G115 -- If this overflowed the test would fail anyway
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
	assert.Equal(t, uint64(465000), uint64(sum)) // #nosec G115 -- If this overflowed the test would fail anyway
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
	gov := NewChainGovernor(logger, &db, common.GoTest, true, "")

	err := gov.Run(ctx)
	if err != nil {
		return gov, err
	}

	emitterAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	if err != nil {
		return gov, err
	}

	tokenAddr, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E")
	if err != nil {
		return gov, err
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

// Converts a TxHash string into a byte array to be used as a TxID.
func hashToTxID(str string) []byte {
	if (len(str) > 2) && (str[0] == '0') && (str[1] == 'x') {
		str = str[2:]
	}

	return eth_common.HexToHash(str).Bytes()
}

func TestVaaForUninterestingEmitterChain(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	emitterAddr, _ := vaa.StringToAddress("0x00")
	payload := []byte{1, 97, 97, 97, 97, 97}

	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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

// Test the flow cancel mechanism at the resolution of the ProcessMsgForTime (VAA parsing)
// This test simulates a transaction of a flow-cancelling asset from one chain to another and back.
// After this operation, we verify that the net flow across these chains is zero but that the
// transfers have indeed been processed.
// Finally a regular (non flow-cancelling) transfer is added just to ensure we aren't testing some empty/nil/0 case.
// The flow cancelling asset has an origin chain that is different from the emitter chain to demonstrate
// that these values don't have to match.
func TestFlowCancelProcessMsgForTimeFullCancel(t *testing.T) {

	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Set-up time
	gov.setDayLengthInMinutes(24 * 60)
	transferTime := time.Unix(int64(1654543099), 0)

	// Solana USDC used as the flow cancelling asset. This ensures that the flow cancel mechanism works
	// when the Origin chain of the asset does not match the emitter chain
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var flowCancelTokenOriginAddress vaa.Address
	flowCancelTokenOriginAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
	require.NoError(t, err)

	var notFlowCancelTokenOriginAddress vaa.Address
	notFlowCancelTokenOriginAddress, err = vaa.StringToAddress("77777af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f7777")
	require.NoError(t, err)

	// Data for Ethereum
	tokenBridgeAddrStrEthereum := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddrEthereum, err := vaa.StringToAddress(tokenBridgeAddrStrEthereum)
	require.NoError(t, err)
	recipientEthereum := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8" //nolint:gosec

	// Data for Sui
	tokenBridgeAddrStrSui := "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9" //nolint:gosec
	tokenBridgeAddrSui, err := vaa.StringToAddress(tokenBridgeAddrStrSui)
	require.NoError(t, err)
	recipientSui := "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31"

	// Data for Solana. Only used to represent the flow cancel asset.
	// "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
	tokenBridgeAddrStrSolana := "0x0e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4" //nolint:gosec

	// Add chain entries to `gov`
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStrEthereum, 10000, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSui, tokenBridgeAddrStrSui, 10000, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSolana, tokenBridgeAddrStrSolana, 10000, 0)
	require.NoError(t, err)

	// Add flow cancel asset and non-flow cancelable asset to the token entry for `gov`
	err = gov.setTokenForTesting(vaa.ChainIDSolana, flowCancelTokenOriginAddress.String(), "USDC", 1.0, true)
	require.NoError(t, err)
	assert.NotNil(t, gov.tokens[tokenKey{chain: vaa.ChainIDSolana, addr: flowCancelTokenOriginAddress}])
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, notFlowCancelTokenOriginAddress.String(), "NOTCANCELABLE", 1.0, false)
	require.NoError(t, err)

	// Transfer from Ethereum to Sui via the token bridge
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        transferTime,
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDSolana, // The origin asset for the token being transferred
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui, // destination chain of the transfer
			recipientSui,
			5000,
		),
	}

	// Transfer from Sui to Ethereum via the token bridge
	msg2 := common.MessagePublication{
		TxID:             hashToTxID("0xabc123f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
		Timestamp:        transferTime,
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   tokenBridgeAddrSui,
		ConsistencyLevel: uint8(0), // Sui has a consistency level of 0 (instant)
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDSolana, // Asset is owned by Solana chain. That's all we care about here.
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDEthereum, // destination chain
			recipientEthereum,
			1000,
		),
	}

	// msg and asset that are NOT flow cancelable
	msg3 := common.MessagePublication{
		TxID:             hashToTxID("0x888888f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a8888"),
		Timestamp:        time.Unix(int64(transferTime.Unix()+1), 0),
		Nonce:            uint32(3),
		Sequence:         uint64(3),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(0), // Sui has a consistency level of 0 (instant)
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum, // Asset is owned by Ethereum chain. That's all we care about here.
			notFlowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui,
			recipientSui,
			1500,
		),
	}

	// Stage 0: No transfers sent
	chainEntryEthereum, exists := gov.chains[vaa.ChainIDEthereum]
	assert.True(t, exists)
	assert.NotNil(t, chainEntryEthereum)
	chainEntrySui, exists := gov.chains[vaa.ChainIDSui]
	assert.True(t, exists)
	assert.NotNil(t, chainEntrySui)
	sumEth, ethTransfers, err := gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, len(ethTransfers))
	assert.Zero(t, sumEth)
	require.NoError(t, err)
	sumSui, suiTransfers, err := gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(1654543099), 0))
	assert.Zero(t, len(suiTransfers))
	assert.Zero(t, sumSui)
	require.NoError(t, err)

	// Perform a FIRST transfer (Ethereum --> Sui)
	result, err := gov.ProcessMsgForTime(&msg1, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, numTrans)          // One for the positive and one for the negative
	assert.Equal(t, int64(0), valueTrans) // Zero! Cancel flow token!
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(1), len(chainEntryEthereum.transfers))
	assert.Equal(t, int(1), len(chainEntrySui.transfers))
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(5000), sumEth) // Outbound on Ethereum
	assert.Equal(t, int(1), len(ethTransfers))
	require.NoError(t, err)

	// Outbound check:
	// - ensure that the sum of the transfers is equal to the value of the inverse transfer
	// - ensure the actual governor usage is Zero (any negative value is converted to zero by TrimAndSumValueForChain)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, 1, len(suiTransfers)) // A single NEGATIVE transfer
	assert.Equal(t, int64(-5000), sumSui) // Ensure the inverse (negative) transfer is in the Sui chain Entry
	require.NoError(t, err)
	suiGovernorUsage, err := gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)

	// Perform a SECOND transfer (Sui --> Ethereum)
	result, err = gov.ProcessMsgForTime(&msg2, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	// Stage 2: Transfer sent from Sui to Ethereum.
	// This transfer should result in some flow cancelling on Ethereum so we assert that its sum has decreased
	// compared to the previous step.
	// Check the governor stats both with respect to flow cancelling and to the actual value that has moved.
	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, len(gov.msgsSeen)) // Two messages observed
	assert.Equal(t, 4, numTrans)          // Two messages, but four transfers because inverses are added.
	assert.Equal(t, int64(0), valueTrans) // The two transfers and their inverses cancel each other out.
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	// Verify the stats that are non flow-cancelling.
	// In practice this is the sum of the absolute value of all the transfers.
	// 5000 * 2 + 1000 * 2 = 12000
	_, absValueTrans, _, _ := gov.getStatsForAllChains()
	assert.Equal(t, uint64(12000), absValueTrans)

	// Check the state of the governor.
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(2), len(chainEntryEthereum.transfers)) // One for inbound refund and another for outbound
	assert.Equal(t, int(2), len(chainEntrySui.transfers))      // One for inbound refund and another for outbound
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(4000), sumEth)       // Out was 5000 then the cancellation makes this 4000.
	assert.Equal(t, int(2), len(ethTransfers)) // Two transfers: outbound 5000 and inverse -1000 transfer
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(2), len(suiTransfers))
	assert.Equal(t, int64(-4000), sumSui) // -5000 from Ethereum inverse added to 1000 from sending to Ethereum
	require.NoError(t, err)
	suiGovernorUsage, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)

	// Message for a non-flow cancellable token (Ethereum --> Sui)
	result, err = gov.ProcessMsgForTime(&msg3, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	// Stage 3: Asset withoout flow cancelling has also been sent
	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 3, len(gov.msgsSeen))
	assert.Equal(t, 5, numTrans)             // Only a single new transfer for the positive change
	assert.Equal(t, int64(1500), valueTrans) // Consume 1500 capacity on Ethereum
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	// Verify the stats that are non flow-cancelling.
	// In practice this is the sum of the absolute value of all the transfers.
	// 5000 * 2 + 1000 * 2 + 1500 = 13500
	_, absValueTrans, _, _ = gov.getStatsForAllChains()
	assert.Equal(t, uint64(13500), absValueTrans) // The net actual flow of assets is 4000 (after cancelling) plus 1500

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(3), len(chainEntryEthereum.transfers)) // One for inbound refund and another for outbound
	assert.Equal(t, int(2), len(chainEntrySui.transfers))      // One for inbound refund and another for outbound
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(5500), sumEth)       // The value of the non-cancelled transfer
	assert.Equal(t, int(3), len(ethTransfers)) // Two transfers cancel each other out
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(2), len(suiTransfers))
	assert.Equal(t, int64(-4000), sumSui) // Sui's limit should not change
	require.NoError(t, err)
	suiGovernorUsage, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)
}

// Test the flow cancel mechanism at the resolution of the ProcessMsgForTime (VAA parsing)
// This test checks a flow cancel scenario where the amounts don't completely cancel each other
// out.
// It also highlights the differences between the following values:
// - Governor stats for chains: the sum of the absolute values of all transfers
// - Governor stats for chains, flow cancelling: the sum of transfer values, including 'inverse' transfers
// - The sum of transfers in a chain entry: The sum of outbound transfers and inbound flow cancelling transfers for a chain
// - The Governor usage for a chain: Same as above but saturates to 0 as a lower bound
func TestFlowCancelProcessMsgForTimePartialCancel(t *testing.T) {

	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Set-up time
	gov.setDayLengthInMinutes(24 * 60)
	transferTime := time.Unix(int64(1654543099), 0)

	// Solana USDC used as the flow cancelling asset. This ensures that the flow cancel mechanism works
	// when the Origin chain of the asset does not match the emitter chain
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var flowCancelTokenOriginAddress vaa.Address
	flowCancelTokenOriginAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
	require.NoError(t, err)

	var notFlowCancelTokenOriginAddress vaa.Address
	notFlowCancelTokenOriginAddress, err = vaa.StringToAddress("77777af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f7777")
	require.NoError(t, err)

	// Data for Ethereum
	tokenBridgeAddrStrEthereum := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddrEthereum, err := vaa.StringToAddress(tokenBridgeAddrStrEthereum)
	require.NoError(t, err)
	recipientEthereum := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8" //nolint:gosec

	// Data for Sui
	tokenBridgeAddrStrSui := "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9" //nolint:gosec
	tokenBridgeAddrSui, err := vaa.StringToAddress(tokenBridgeAddrStrSui)
	require.NoError(t, err)
	recipientSui := "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31" //nolint:gosec

	// Add chain entries to `gov`
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStrEthereum, 10000, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSui, tokenBridgeAddrStrSui, 10000, 0)
	require.NoError(t, err)

	// Add flow cancel asset and non-flow cancelable asset to the token entry for `gov`
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, flowCancelTokenOriginAddress.String(), "USDC", 1.0, true)
	require.NoError(t, err)
	assert.NotNil(t, gov.tokens[tokenKey{chain: vaa.ChainIDEthereum, addr: flowCancelTokenOriginAddress}])
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, notFlowCancelTokenOriginAddress.String(), "NOTCANCELABLE", 2.5, false)
	require.NoError(t, err)

	// Transfer from Ethereum to Sui via the token bridge
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        transferTime,
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum, // The origin asset for the token being transferred
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui, // destination chain of the transfer
			recipientSui,
			5000,
		),
	}

	// Transfer from Sui to Ethereum via the token bridge
	msg2 := common.MessagePublication{
		TxID:             hashToTxID("0xabc123f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4064"),
		Timestamp:        transferTime,
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   tokenBridgeAddrSui,
		ConsistencyLevel: uint8(0), // Sui has a consistency level of 0 (instant)
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum, // Asset is owned by Ethereum chain. That's all we care about here.
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDEthereum, // destination chain
			recipientEthereum,
			5000,
		),
	}

	// msg and asset that are NOT flow cancelable
	msg3 := common.MessagePublication{
		TxID:             hashToTxID("0x888888f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a8888"),
		Timestamp:        time.Unix(int64(transferTime.Unix()+1), 0),
		Nonce:            uint32(3),
		Sequence:         uint64(3),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(0), // Sui has a consistency level of 0 (instant)
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDEthereum, // Asset is owned by Ethereum chain. That's all we care about here.
			notFlowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui,
			recipientSui,
			1000, // Note that this asset is worth 2.5 USD, so the notional value is 2500
		),
	}

	// Stage 0: No transfers sent
	chainEntryEthereum, exists := gov.chains[vaa.ChainIDEthereum]
	assert.True(t, exists)
	assert.NotNil(t, chainEntryEthereum)
	chainEntrySui, exists := gov.chains[vaa.ChainIDSui]
	assert.True(t, exists)
	assert.NotNil(t, chainEntrySui)
	sumEth, ethTransfers, err := gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, len(ethTransfers))
	assert.Zero(t, sumEth)
	require.NoError(t, err)
	sumSui, suiTransfers, err := gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(1654543099), 0))
	assert.Zero(t, len(suiTransfers))
	assert.Zero(t, sumSui)
	require.NoError(t, err)

	result, err := gov.ProcessMsgForTime(&msg1, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	numTrans, valueTrans, numPending, valuePending := gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, numTrans)          // One for the positive and one for the negative
	assert.Equal(t, int64(0), valueTrans) // Zero! Cancel flow token!
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(1), len(chainEntryEthereum.transfers))
	assert.Equal(t, int(1), len(chainEntrySui.transfers))
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(5000), sumEth) // Outbound on Ethereum
	assert.Equal(t, int(1), len(ethTransfers))
	require.NoError(t, err)

	// Outbound check:
	// - ensure that the sum of the transfers is equal to the value of the inverse transfer
	// - ensure the actual governor usage is Zero (any negative value is converted to zero by TrimAndSumValueForChain)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, 1, len(suiTransfers)) // A single NEGATIVE transfer
	assert.Equal(t, int64(-5000), sumSui) // Ensure the inverse (negative) transfer is in the Sui chain Entry
	require.NoError(t, err)
	suiGovernorUsage, err := gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)

	// Perform a SECOND transfer (Sui --> Ethereum)
	result, err = gov.ProcessMsgForTime(&msg2, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	// Stage 2: Transfer sent from Sui to Ethereum.
	// This transfer should result in flow cancelling on Ethereum so we assert that its sum has decreased
	// compared to the previous step.
	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, len(gov.msgsSeen)) // Two messages observed
	assert.Equal(t, 4, numTrans)          // Two messages, but four transfers because inverses are added.
	assert.Equal(t, int64(0), valueTrans) // New flow is zero! Cancel flow token!
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	// Check the state of the governor. Confirm that both chains have two transfers but have cancelled
	// each other out in terms of the summed values.
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(2), len(chainEntryEthereum.transfers)) // One for inbound refund and another for outbound
	assert.Equal(t, int(2), len(chainEntrySui.transfers))      // One for inbound refund and another for outbound
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(0), sumEth)          // Out was 4000 then the cancellation makes this zero.
	assert.Equal(t, int(2), len(ethTransfers)) // Two transfers cancel each other out
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(2), len(suiTransfers))
	assert.Equal(t, int64(0), sumSui)
	require.NoError(t, err)

	// Message for a non-flow cancellable token (Ethereum --> Sui)
	result, err = gov.ProcessMsgForTime(&msg3, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	// Stage 3: Asset withoout flow cancelling has also been sent
	numTrans, valueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 3, len(gov.msgsSeen))
	assert.Equal(t, 5, numTrans)             // Only a single new transfer for the positive change
	assert.Equal(t, int64(2500), valueTrans) // Change in value from the transfer: 1000 tokens worth $2.5 USD
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(3), len(chainEntryEthereum.transfers)) // One for inbound refund and another for outbound
	assert.Equal(t, int(2), len(chainEntrySui.transfers))      // One for inbound refund and another for outbound
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(2500), sumEth)       // The value of the non-cancelled transfer
	assert.Equal(t, int(3), len(ethTransfers)) // Two transfers cancel each other out
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(2), len(suiTransfers))
	assert.Equal(t, int64(0), sumSui) // Sui's limit is still zero
	require.NoError(t, err)
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddr,
		ConsistencyLevel: uint8(32),
		Payload:          payloadBytes1,
	}

	msg2 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	now, err = time.Parse("Jan 2, 2006 at 3:04pm (MST)", "Jun 2, 2022 at 9:00am (CST)")
	require.NoError(t, err)
	assert.NotNil(t, now)
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

func TestPopulateChainIds(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)
	require.NoError(t, err)
	assert.NotNil(t, gov)
	// Sanity check
	assert.NotZero(t, len(gov.chainIds))

	// Ensure that the chainIds slice match the entries in the chains map
	assert.Equal(t, len(gov.chains), len(gov.chainIds))
	lowest := 0
	for _, chainId := range gov.chainIds {
		chainEntry, ok := gov.chains[chainId]
		assert.NotNil(t, chainEntry)
		assert.True(t, ok)
		assert.Equal(t, chainEntry.emitterChainId, chainId)
		// Check that the chainIds are in ascending order. The point of this slice is that it provides
		// deterministic ordering over chainIds.
		assert.Greater(t, int(chainId), lowest)
		lowest = int(chainId)
	}
}

// Test that, when a small transfer (under the 'big tx limit') of a flow-cancelling asset is queued and
// later released, it causes a reduction in the Governor usage for the destination chain.
func TestPendingTransferFlowCancelsWhenReleased(t *testing.T) {

	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Set-up time
	gov.setDayLengthInMinutes(24 * 60)
	transferTime := time.Unix(int64(1654543099), 0)

	// Solana USDC used as the flow cancelling asset. This ensures that the flow cancel mechanism works
	// when the Origin chain of the asset does not match the emitter chain
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var flowCancelTokenOriginAddress vaa.Address
	flowCancelTokenOriginAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
	require.NoError(t, err)

	require.NoError(t, err)

	// Data for Ethereum
	tokenBridgeAddrStrEthereum := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddrEthereum, err := vaa.StringToAddress(tokenBridgeAddrStrEthereum)
	require.NoError(t, err)
	recipientEthereum := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8" //nolint:gosec

	// Data for Sui
	tokenBridgeAddrStrSui := "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9" //nolint:gosec
	tokenBridgeAddrSui, err := vaa.StringToAddress(tokenBridgeAddrStrSui)
	require.NoError(t, err)
	recipientSui := "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31"

	// Data for Solana. Only used to represent the flow cancel asset.
	// "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
	tokenBridgeAddrStrSolana := "0x0e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4" //nolint:gosec

	// Add chain entries to `gov`
	dailyLimit := uint64(10000)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStrEthereum, dailyLimit, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSui, tokenBridgeAddrStrSui, dailyLimit, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSolana, tokenBridgeAddrStrSolana, dailyLimit, 0)
	require.NoError(t, err)

	// Add flow cancel asset and non-flow cancelable asset to the token entry for `gov`
	err = gov.setTokenForTesting(vaa.ChainIDSolana, flowCancelTokenOriginAddress.String(), "USDC", 1.0, true)
	require.NoError(t, err)
	assert.NotNil(t, gov.tokens[tokenKey{chain: vaa.ChainIDSolana, addr: flowCancelTokenOriginAddress}])

	// First message: consume most of the dailyLimit for the emitter chain
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x888888f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a8888"),
		Timestamp:        time.Unix(int64(transferTime.Unix()+1), 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDSolana, // The origin asset for the token being transferred
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui,
			recipientSui,
			10000,
		),
	}

	// Second message: This transfer gets queued because the limit is exhausted
	msg2 := common.MessagePublication{
		TxID:             hashToTxID("0x888888f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a8888"),
		Timestamp:        time.Unix(int64(transferTime.Unix()+2), 0),
		Nonce:            uint32(2),
		Sequence:         uint64(2),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddrEthereum,
		ConsistencyLevel: uint8(32),
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDSolana,
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDSui,
			recipientSui,
			500,
		),
	}

	// Third message: Incoming flow cancelling transfer to the emitter chain for the previous messages. This
	// reduces the Governor usage for that chain.
	msg3 := common.MessagePublication{
		TxID:             hashToTxID("0x888888f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a8888"),
		Timestamp:        time.Unix(int64(transferTime.Unix()+3), 0),
		Nonce:            uint32(3),
		Sequence:         uint64(3),
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   tokenBridgeAddrSui,
		ConsistencyLevel: uint8(0), // Sui has a consistency level of 0 (instant)
		Payload: buildMockTransferPayloadBytes(1,
			vaa.ChainIDSolana,
			flowCancelTokenOriginAddress.String(),
			vaa.ChainIDEthereum,
			recipientEthereum,
			1000,
		),
	}

	// Stage 0: No transfers sent
	chainEntryEthereum, exists := gov.chains[vaa.ChainIDEthereum]
	assert.True(t, exists)
	assert.NotNil(t, chainEntryEthereum)
	chainEntrySui, exists := gov.chains[vaa.ChainIDSui]
	assert.True(t, exists)
	assert.NotNil(t, chainEntrySui)
	sumEth, ethTransfers, err := gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, len(ethTransfers))
	assert.Zero(t, len(chainEntryEthereum.pending))
	assert.Zero(t, sumEth)
	require.NoError(t, err)
	sumSui, suiTransfers, err := gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(1654543099), 0))
	assert.Zero(t, len(suiTransfers))
	assert.Zero(t, sumSui)
	require.NoError(t, err)

	// Perform a FIRST transfer (Ethereum --> Sui)
	result, err := gov.ProcessMsgForTime(&msg1, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	numTrans, netValueTrans, numPending, valuePending := gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, numTrans)             // One for the positive and one for the negative
	assert.Equal(t, int64(0), netValueTrans) // Zero, because the asset flow cancels
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
	assert.Equal(t, 1, len(gov.msgsSeen))

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(1), len(chainEntryEthereum.transfers))
	assert.Equal(t, int(0), len(chainEntryEthereum.pending)) // One for inbound refund and another for outbound
	assert.Equal(t, int(1), len(chainEntrySui.transfers))
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(10000), sumEth) // Equal to total dailyLimit
	assert.Equal(t, int(1), len(ethTransfers))
	require.NoError(t, err)

	// Outbound check:
	// - ensure that the sum of the transfers is equal to the value of the inverse transfer
	// - ensure the actual governor usage is Zero (any negative value is converted to zero by TrimAndSumValueForChain)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, 1, len(suiTransfers))  // A single NEGATIVE transfer
	assert.Equal(t, int64(-10000), sumSui) // Ensure the inverse (negative) transfer is in the Sui chain Entry
	require.NoError(t, err)
	suiGovernorUsage, err := gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)

	// Perform a SECOND transfer (Ethereum --> Sui again)
	// When a transfer is queued, ProcessMsgForTime should return false.
	result, err = gov.ProcessMsgForTime(&msg2, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.False(t, result)
	require.NoError(t, err)

	// Stage 2: Transfer sent from Ethereum to Sui gets queued
	numTrans, netValueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, len(gov.msgsSeen))    // Two messages observed
	assert.Equal(t, 2, numTrans)             // Two transfers (same as previous step)
	assert.Equal(t, int64(0), netValueTrans) // The two transfers and their inverses cancel each other out.
	assert.Equal(t, 1, numPending)           // Second transfer is queued because the limit is exhausted
	assert.Equal(t, uint64(500), valuePending)

	// Check the state of the governor.
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(1), len(chainEntryEthereum.transfers)) // One from previous step
	assert.Equal(t, int(1), len(chainEntryEthereum.pending))   // One for inbound refund and another for outbound
	assert.Equal(t, int(1), len(chainEntrySui.transfers))      // One inverse transfer. Inverse from pending not added yet
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(10000), sumEth)      // Same as before: full dailyLimit
	assert.Equal(t, int(1), len(ethTransfers)) // Same as before
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(1), len(suiTransfers)) // just the inverse from before
	assert.Equal(t, int64(-10000), sumSui)     // Unchanged.
	require.NoError(t, err)
	suiGovernorUsage, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, suiGovernorUsage) // Actual governor usage must not be negative.
	require.NoError(t, err)

	// Stage 3: Message that reduces Governor usage for Ethereum (Sui --> Ethereum)
	result, err = gov.ProcessMsgForTime(&msg3, time.Now())
	assert.True(t, result)
	require.NoError(t, err)

	// Stage 3: Governor usage reduced on Ethereum due to incoming from Sui
	numTrans, netValueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 3, len(gov.msgsSeen))
	assert.Equal(t, 4, numTrans)             // Two transfers and their inverses
	assert.Equal(t, int64(0), netValueTrans) // Still zero because everything flow cancels
	assert.Equal(t, 1, numPending)           // Not released yet
	assert.Equal(t, uint64(500), valuePending)

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(2), len(chainEntryEthereum.transfers))
	assert.Equal(t, int(1), len(chainEntryEthereum.pending)) // We have not yet released the pending transfer
	assert.Equal(t, int(2), len(chainEntrySui.transfers))
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(9000), sumEth)       // We freed up room because of Sui incoming
	assert.Equal(t, int(2), len(ethTransfers)) // Two transfers cancel each other out
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(2), len(suiTransfers))
	assert.Equal(t, int64(-9000), sumSui) // We consumed some outbound capacity
	require.NoError(t, err)
	suiGovernorUsage, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, uint64(0), suiGovernorUsage) // Still zero because it's still negative
	require.NoError(t, err)

	// Stage 4: Release the pending transfer. We deliberately do not advance the time here because we are relying
	// on the pending transfer being released as a result of flow-cancelling and not because 24 hours have passed.
	// NOTE that even though the function says "Checked..." it modifies `gov` as a side-effect when a pending
	// transfer is ready to be released
	toBePublished, err := gov.CheckPendingForTime(time.Unix(int64(transferTime.Unix()-1000), 0))
	require.NoError(t, err)
	assert.Equal(t, 1, len(toBePublished))

	// Stage 4: Pending transfer released. This increases the Ethereum Governor usage again and reduces Sui.
	numTrans, netValueTrans, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 3, len(gov.msgsSeen))
	assert.Equal(t, 6, numTrans)             // Two new transfers created from previous pending transfer
	assert.Equal(t, int64(0), netValueTrans) // Still zero because everything flow cancels
	assert.Equal(t, 0, numPending)           // Pending transfer has been released
	assert.Equal(t, uint64(0), valuePending)

	// Verify the stats that are non flow-cancelling.
	// In practice this is the sum of the absolute value of all the transfers, including the inverses.
	// 2 * (10000 + 1000 + 500) = 23000
	_, absValueTrans, _, _ := gov.getStatsForAllChains()
	assert.Equal(t, uint64(23000), absValueTrans)

	// Check the state of the governor
	chainEntryEthereum = gov.chains[vaa.ChainIDEthereum]
	chainEntrySui = gov.chains[vaa.ChainIDSui]
	assert.Equal(t, int(3), len(chainEntryEthereum.transfers)) // Two outbound, one inverse from Sui
	assert.Equal(t, int(0), len(chainEntryEthereum.pending))   // Released
	assert.Equal(t, int(3), len(chainEntrySui.transfers))      // One outbound, two inverses from Ethereum
	sumEth, ethTransfers, err = gov.TrimAndSumValue(chainEntryEthereum.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(9500), sumEth)
	assert.Equal(t, int(3), len(ethTransfers))
	require.NoError(t, err)
	sumSui, suiTransfers, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int(3), len(suiTransfers)) // New inverse transfer added after pending transfer was released
	assert.Equal(t, int64(-9500), sumSui)      // Flow-cancelling inverse transfer added to Sui after released
	require.NoError(t, err)
	suiGovernorUsage, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, uint64(0), suiGovernorUsage) // Still zero
	require.NoError(t, err)
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// The first VAA should be accepted.
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	gov := NewChainGovernor(logger, &db, common.GoTest, true, "")

	gov.env = common.TestNet
	err := gov.initConfig()
	require.NoError(t, err)
}

func TestTestnetConfigIsValid(t *testing.T) {
	logger := zap.NewNop()
	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, common.GoTest, true, "")

	gov.env = common.TestNet
	err := gov.initConfig()
	require.NoError(t, err)
}

func TestNumDaysForReleaseTimerReset(t *testing.T) {
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)

	tokenAddrStr := "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E" //nolint:gosec
	toAddrStr := "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8"
	tokenBridgeAddrStr := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddr, err := vaa.StringToAddress(tokenBridgeAddrStr)
	require.NoError(t, err)

	gov.setDayLengthInMinutes(24 * 60)
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStr, 1000000, 100000)
	require.NoError(t, err)
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	now := time.Now()
	messageTimestamp := now.Add(-5) // 5 seconds ago

	// message that, when processed, should exceed the big transfer size
	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        messageTimestamp,
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

	canPost, err := gov.ProcessMsgForTime(&msg, now)
	require.NoError(t, err)
	assert.Equal(t, false, canPost)

	msg.MessageIDString()

	// check that the enqueued vaa's release date is now + 1 day
	expectedReleaseTime := uint32(now.Add(24 * time.Hour).Unix()) // #nosec G115 -- This conversion is safe until year 2106
	enqueuedVaas := gov.GetEnqueuedVAAs()
	assert.Equal(t, len(enqueuedVaas), 1)
	assert.Equal(t, enqueuedVaas[0].ReleaseTime, expectedReleaseTime)

	// the release timer gets reset to 5 days
	_, err = gov.resetReleaseTimerForTime(msg.MessageIDString(), now, 5)
	require.NoError(t, err)

	// check that the enqueued vaa's release date is now + 5 days
	enqueuedVaas = gov.GetEnqueuedVAAs()
	assert.Equal(t, len(enqueuedVaas), 1)
	expectedReleaseTime = uint32(now.Add(5 * 24 * time.Hour).Unix()) // #nosec G115 -- This conversion is safe until year 2106
	assert.Equal(t, enqueuedVaas[0].ReleaseTime, expectedReleaseTime)

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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// The first small transfer should be accepted.
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	_, err = gov.resetReleaseTimerForTime(msg3.MessageIDString(), now, 1)
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
	_, outgoing, _, err := sumValue(ce.transfers, now)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), outgoing)
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// Submit a small transfer that will get enqueued due to the low daily limit.
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, emitterAddrStr, "WETH", 1774.62, false)
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
			TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
			TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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

// With the addition of the flow-cancel feature, it's possible to in a way "exceed the daily limit" of outflow from a
// Governor as long as there is corresponding inflow of a flow-canceling asset to allow for additional outflow.
// When the node is restarted, it reloads all transfers and pending transfers. If the actual outflow is greater than
// the daily limit (due to flow cancel) ensure that the calculated limit on start-up is correct.
// This test ensures that governor usage limits are correctly calculated when reloading transfers from the database.
func TestReloadTransfersNearCapacity(t *testing.T) {
	// Setup
	ctx := context.Background()
	gov, err := newChainGovernorForTest(ctx)

	require.NoError(t, err)
	assert.NotNil(t, gov)

	// Set-up time
	gov.setDayLengthInMinutes(24 * 60)
	transferTime := time.Now()

	// Solana USDC used as the flow cancelling asset. This ensures that the flow cancel mechanism works
	// when the Origin chain of the asset does not match the emitter chain
	// NOTE: Replace this Chain:Address pair if the Flow Cancel Token List is modified
	var flowCancelTokenOriginAddress vaa.Address
	flowCancelTokenOriginAddress, err = vaa.StringToAddress("c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61")
	require.NoError(t, err)

	var notFlowCancelTokenOriginAddress vaa.Address
	notFlowCancelTokenOriginAddress, err = vaa.StringToAddress("77777af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f7777")
	require.NoError(t, err)

	// Data for Ethereum
	tokenBridgeAddrStrEthereum := "0x0290fb167208af455bb137780163b7b7a9a10c16" //nolint:gosec
	tokenBridgeAddrEthereum, err := vaa.StringToAddress(tokenBridgeAddrStrEthereum)
	require.NoError(t, err)

	// Data for Sui
	tokenBridgeAddrStrSui := "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9" //nolint:gosec
	tokenBridgeAddrSui, err := vaa.StringToAddress(tokenBridgeAddrStrSui)
	require.NoError(t, err)

	// Data for Solana. Only used to represent the flow cancel asset.
	// "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
	tokenBridgeAddrStrSolana := "0x0e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4" //nolint:gosec

	// Add chain entries to `gov`
	err = gov.setChainForTesting(vaa.ChainIDEthereum, tokenBridgeAddrStrEthereum, 10000, 50000)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSui, tokenBridgeAddrStrSui, 10000, 0)
	require.NoError(t, err)
	err = gov.setChainForTesting(vaa.ChainIDSolana, tokenBridgeAddrStrSolana, 10000, 0)
	require.NoError(t, err)

	// Add flow cancel asset and non-flow cancelable asset to the token entry for `gov`
	err = gov.setTokenForTesting(vaa.ChainIDSolana, flowCancelTokenOriginAddress.String(), "USDC", 1.0, true)
	require.NoError(t, err)
	assert.NotNil(t, gov.tokens[tokenKey{chain: vaa.ChainIDSolana, addr: flowCancelTokenOriginAddress}])
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, notFlowCancelTokenOriginAddress.String(), "NOTCANCELABLE", 1.0, false)
	require.NoError(t, err)

	// This transfer should exhaust the dailyLimit for the emitter chain
	xfer1 := &db.Transfer{
		Timestamp:      transferTime.Add(-10),
		Value:          uint64(10000),
		OriginChain:    vaa.ChainIDSolana,
		OriginAddress:  flowCancelTokenOriginAddress,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddrEthereum,
		TargetAddress:  tokenBridgeAddrSui,
		TargetChain:    vaa.ChainIDSui,
		MsgID:          "2/" + tokenBridgeAddrEthereum.String() + "/125",
		Hash:           "Hash1",
	}

	// This incoming transfer should free up some of the space on the previous emitter chain
	xfer2 := &db.Transfer{
		Timestamp:      transferTime.Add(-9),
		Value:          uint64(2000),
		OriginChain:    vaa.ChainIDSolana,
		OriginAddress:  flowCancelTokenOriginAddress,
		EmitterChain:   vaa.ChainIDSui,
		EmitterAddress: tokenBridgeAddrSui,
		TargetAddress:  tokenBridgeAddrEthereum,
		TargetChain:    vaa.ChainIDEthereum,
		MsgID:          "2/" + tokenBridgeAddrSui.String() + "/126",
		Hash:           "Hash2",
	}

	// Send another transfer out from the original emitter chain so that we "exceed the daily limit" if flow
	// cancel is not applied
	xfer3 := &db.Transfer{
		Timestamp:      transferTime.Add(-8),
		Value:          uint64(50),
		OriginChain:    vaa.ChainIDSolana,
		OriginAddress:  flowCancelTokenOriginAddress,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: tokenBridgeAddrEthereum,
		TargetAddress:  tokenBridgeAddrSui,
		TargetChain:    vaa.ChainIDSui,
		MsgID:          "2/" + tokenBridgeAddrEthereum.String() + "/125",
		Hash:           "Hash3",
	}

	// Simulate reloading from the database.
	// NOTE: The actual execution path we want to test is the following and runs when the node is restarted:
	//	gov.Run () --> gov.loadFromDb() --> gov.loadFromDBAlreadyLocked() --> gov.reloadTransfer()
	// We don't have access to Run() from the test suite and the other functions are mocked to return `nil`.
	// Therefore, the remainder of this test proceeds by operating on a list of `transfersLoadedFromDb` which
	// simulates loading transfers from the database.
	// From here we proceed with the next function we can actually test: `reloadTransfer()`.

	// STEP 0: Initial state
	assert.Equal(t, len(gov.msgsSeen), 0)
	numTrans, netValueTransferred, numPending, valuePending := gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 0, numTrans)
	assert.Equal(t, int64(0), netValueTransferred)
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)

	chainEntryEth, exists := gov.chains[vaa.ChainIDEthereum]
	require.True(t, exists)
	chainEntrySui, exists := gov.chains[vaa.ChainIDSui]
	require.True(t, exists)

	// STEP 1: Load first transfer
	err = gov.reloadTransfer(xfer1)
	require.NoError(t, err)
	assert.Equal(t, len(gov.msgsSeen), 1)
	numTrans, netValueTransferred, _, _ = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 2, numTrans)                   // 1 plus transfer the inverse flow transfer on the TargetChain
	assert.Equal(t, int64(0), netValueTransferred) // Value cancels out for all transfers

	// Sum of absolute value of all transfers, including inverse flow cancel transfers:
	// 2 * (10_000) = 20_000
	_, valueTransferred, _, _ := gov.getStatsForAllChains()
	assert.Equal(t, uint64(20000), valueTransferred)

	governorUsageEth, err := gov.TrimAndSumValueForChain(chainEntryEth, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, uint64(10000), governorUsageEth)
	assert.Zero(t, governorUsageEth-chainEntryEth.dailyLimit) // Make sure we used the whole capacity
	require.NoError(t, err)
	governorUsageSui, err := gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, governorUsageSui)
	require.NoError(t, err)
	sumTransfersSui, _, err := gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(-10000), sumTransfersSui)
	require.NoError(t, err)

	// STEP 2: Load second transfer
	err = gov.reloadTransfer(xfer2)
	require.NoError(t, err)
	assert.Equal(t, len(gov.msgsSeen), 2)
	numTrans, netValueTransferred, _, _ = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 4, numTrans)                   // 2 transfers and their inverse flow transfers on the TargetChain
	assert.Equal(t, int64(0), netValueTransferred) // Value cancels out for all transfers

	// Sum of absolute value of all transfers, including inverse flow cancel transfers:
	// 2 * (10_000 + 2_000) = 24_000
	_, valueTransferred, _, _ = gov.getStatsForAllChains()
	assert.Equal(t, uint64(24000), valueTransferred)

	governorUsageEth, err = gov.TrimAndSumValueForChain(chainEntryEth, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, uint64(8000), governorUsageEth)
	// Remaining capacity
	assert.Equal(t, int(chainEntryEth.dailyLimit-governorUsageEth), 2000) // #nosec G115 -- If this overflowed the test would fail
	require.NoError(t, err)
	governorUsageSui, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Zero(t, governorUsageSui)
	require.NoError(t, err)
	sumTransfersSui, _, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(-8000), sumTransfersSui)
	require.NoError(t, err)

	// STEP 3: Load third transfer
	err = gov.reloadTransfer(xfer3)
	require.NoError(t, err)
	// Sum of absolute value of all transfers, including inverse flow cancel transfers:
	// 2 * (10_000 + 2_000 + 50) = 24_100
	_, valueTransferred, _, _ = gov.getStatsForAllChains()
	assert.Equal(t, uint64(24100), valueTransferred)

	numTrans, netValueTransferred, numPending, valuePending = gov.getStatsForAllChainsCancelFlow()
	assert.Equal(t, 6, numTrans)                   // 3 transfers and their inverse flow transfers on the TargetChain
	assert.Equal(t, int64(0), netValueTransferred) // Value cancels out for all transfers

	governorUsageEth, err = gov.TrimAndSumValueForChain(chainEntryEth, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, uint64(8050), governorUsageEth)
	// Remaining capacity
	assert.Equal(t, int(chainEntryEth.dailyLimit-governorUsageEth), 1950) // #nosec G115 -- If this overflowed the test would fail
	require.NoError(t, err)
	governorUsageSui, err = gov.TrimAndSumValueForChain(chainEntrySui, time.Unix(int64(transferTime.Unix()-1000), 0))
	require.NoError(t, err)
	assert.Zero(t, governorUsageSui)
	sumTransfersSui, _, err = gov.TrimAndSumValue(chainEntrySui.transfers, time.Unix(int64(transferTime.Unix()-1000), 0))
	assert.Equal(t, int64(-8050), sumTransfersSui)
	require.NoError(t, err)

	// Sanity check: make sure these are still empty/zero
	assert.Equal(t, 0, numPending)
	assert.Equal(t, uint64(0), valuePending)
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// The first transfer should be accepted.
	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// A big transfer should get enqueued.
	msg := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// The first transfer should be accepted.
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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

			queries := createCoinGeckoQueries(ids, tc.chunkSize, "")
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

// Test the URL of CoinGecko queries to be correct
func TestCoinGeckoQueryFormat(t *testing.T) {
	id_amount := 10
	ids := make([]string, id_amount)
	for idx := 0; idx < id_amount; idx++ {
		ids[idx] = fmt.Sprintf("id%d", idx)
	}

	// Create and parse the query
	queries := createCoinGeckoQueries(ids, 100, "") // No API key
	require.Equal(t, len(queries), 1)
	query_url, err := url.Parse(queries[0])
	require.Equal(t, err, nil)
	params, err := url.ParseQuery(query_url.RawQuery)
	require.Equal(t, err, nil)

	// Test the portions of the URL for the non-pro version of the API
	require.Equal(t, query_url.Scheme, "https")
	require.Equal(t, query_url.Host, "api.coingecko.com")
	require.Equal(t, query_url.Path, "/api/v3/simple/price")
	require.Equal(t, params.Has("x_cg_pro_api_key"), false)
	require.Equal(t, params.Has("vs_currencies"), true)
	require.Equal(t, params["vs_currencies"][0], "usd")
	require.Equal(t, params.Has("ids"), true)

	// Create and parse the query with an API key
	queries = createCoinGeckoQueries(ids, 100, "FAKE_KEY") // With API key
	require.Equal(t, len(queries), 1)
	query_url, err = url.Parse(queries[0])
	require.Equal(t, err, nil)
	params, err = url.ParseQuery(query_url.RawQuery)
	require.Equal(t, err, nil)

	// Test the portions of the URL actually provided
	require.Equal(t, query_url.Scheme, "https")
	require.Equal(t, query_url.Host, "pro-api.coingecko.com")
	require.Equal(t, query_url.Path, "/api/v3/simple/price")
	require.Equal(t, params.Has("x_cg_pro_api_key"), true)
	require.Equal(t, params["x_cg_pro_api_key"][0], "FAKE_KEY")
	require.Equal(t, params.Has("vs_currencies"), true)
	require.Equal(t, params["vs_currencies"][0], "usd")
	require.Equal(t, params.Has("ids"), true)
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
	err = gov.setTokenForTesting(vaa.ChainIDEthereum, tokenAddrStr, "WETH", 1774.62, false)
	require.NoError(t, err)

	// Create two big transactions.
	msg1 := common.MessagePublication{
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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
		TxID:             hashToTxID("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
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

	// Post the two big transfers and verify they get enqueued.
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
