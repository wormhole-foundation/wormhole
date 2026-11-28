package governor

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestIsVAAEnqueuedNilMessageID(t *testing.T) {
	logger, _ := zap.NewProduction()
	gov := NewChainGovernor(logger, nil, common.GoTest, true, "")
	enqueued, err := gov.IsVAAEnqueued(nil)
	require.EqualError(t, err, "no message ID specified")
	assert.Equal(t, false, enqueued)
}

type statusChainSpec struct {
	chainID   vaa.ChainID
	numVAAs   int
	txIDBytes int
}

const (
	testDailyLimit       uint64 = 100_000
	testBigTxSize        uint64 = 10_000
	testTokenPrice              = 1.0
	testTokenDecimals    int64  = 100_000_000 // 1e8
	testTokenAmount      int64  = 500_000_000 // 5 tokens ($5)
	testStandardTxIDLen         = 32
	testSolanaTxIDLen           = 64
	testMaxTxIDLen              = 255
	testMaxEntryOverhead        = 160
)

func newStatusTestGovernor(t *testing.T, specs []statusChainSpec) *ChainGovernor {
	t.Helper()

	logger := zap.NewNop()
	gov := NewChainGovernor(logger, nil, common.GoTest, false, "")

	token := &tokenEntry{
		price:    big.NewFloat(testTokenPrice),
		decimals: big.NewInt(testTokenDecimals),
		symbol:   "USDC",
	}

	releaseTime := time.Unix(1_700_000_000, 0).UTC()
	chainIDs := make([]vaa.ChainID, 0, len(specs))

	for _, spec := range specs {
		var emitterAddr vaa.Address
		binary.BigEndian.PutUint16(emitterAddr[len(emitterAddr)-2:], uint16(spec.chainID))

		ce := &chainEntry{
			emitterChainId:          spec.chainID,
			emitterAddr:             emitterAddr,
			dailyLimit:              testDailyLimit,
			bigTransactionSize:      testBigTxSize,
			checkForBigTransactions: true,
			pending:                 make([]*pendingEntry, 0, spec.numVAAs),
		}

		for seq := range spec.numVAAs {
			txID := make([]byte, spec.txIDBytes)
			for j := range txID {
				txID[j] = byte((seq + j) & 0xff)
			}
			msg := common.MessagePublication{
				TxID:           txID,
				Timestamp:      releaseTime.Add(-maxEnqueuedTime),
				Sequence:       uint64(seq), // #nosec G115 -- loop index is non-negative
				EmitterChain:   spec.chainID,
				EmitterAddress: emitterAddr,
			}
			ce.pending = append(ce.pending, &pendingEntry{
				token:  token,
				amount: big.NewInt(testTokenAmount),
				hash:   fmt.Sprintf("%d-%d", spec.chainID, seq),
				dbData: guardianDB.PendingTransfer{
					ReleaseTime: releaseTime,
					Msg:         msg,
				},
			})
		}

		gov.chains[spec.chainID] = ce
		chainIDs = append(chainIDs, spec.chainID)
	}

	sort.Slice(chainIDs, func(i, j int) bool {
		return chainIDs[i] < chainIDs[j]
	})
	gov.chainIds = chainIDs

	return gov
}

func publishAndDecodeStatus(t *testing.T, gov *ChainGovernor) ([]byte, *gossipv1.ChainGovernorStatus) {
	t.Helper()

	ctx := context.Background()
	signer, err := guardiansigner.NewGeneratedSigner(nil)
	require.NoError(t, err)
	pubKey := signer.PublicKey(ctx)
	ourAddr := ethCrypto.PubkeyToAddress(pubKey)

	hb := &gossipv1.Heartbeat{
		NodeName:  "test-guardian",
		Timestamp: time.Unix(1_700_000_000, 0).UnixNano(),
	}

	sendC := make(chan []byte, 1)
	gov.publishStatus(ctx, hb, sendC, time.Unix(1_700_000_000, 0).UTC(), signer, ourAddr)

	var raw []byte
	select {
	case raw = <-sendC:
	default:
		t.Fatal("expected publishStatus to send a message")
	}

	var gossipMsg gossipv1.GossipMessage
	require.NoError(t, proto.Unmarshal(raw, &gossipMsg))

	signed := gossipMsg.GetSignedChainGovernorStatus()
	require.NotNil(t, signed)
	assert.Equal(t, ourAddr.Bytes(), signed.GuardianAddr)

	digest := ethCrypto.Keccak256Hash(append(append([]byte(nil), governorMessagePrefixStatus...), signed.Status...))
	valid, err := signer.Verify(ctx, signed.Signature, digest.Bytes())
	require.NoError(t, err)
	assert.True(t, valid)

	var status gossipv1.ChainGovernorStatus
	require.NoError(t, proto.Unmarshal(signed.Status, &status))

	return raw, &status
}

func TestSignedStatusGossipSizeMatchesMarshaled(t *testing.T) {
	// Test boundary sizes where protobuf length varints change width (0, 127->128, 16383->16384, 256 KiB).
	payloadSizes := []int{0, 1, 10, 126, 127, 128, 16_383, 16_384, MaxGovernorStatusMessageSize}
	sig := make([]byte, ethCrypto.SignatureLength)
	addr := make([]byte, ethCommon.AddressLength)

	for _, payloadSize := range payloadSizes {
		msg := &gossipv1.GossipMessage{
			Message: &gossipv1.GossipMessage_SignedChainGovernorStatus{
				SignedChainGovernorStatus: &gossipv1.SignedChainGovernorStatus{
					Status:       make([]byte, payloadSize),
					Signature:    sig,
					GuardianAddr: addr,
				},
			},
		}
		raw, err := proto.Marshal(msg)
		require.NoError(t, err)
		assert.Equal(t, len(raw), signedStatusGossipSize(payloadSize, len(addr)), "payloadSize=%d", payloadSize)
	}
}

func TestPublishStatusUnderCapIncludesAllEnqueuedVAAs(t *testing.T) {
	const (
		ethPending  = 5
		nearPending = 120
	)
	gov := newStatusTestGovernor(t, []statusChainSpec{
		{chainID: vaa.ChainIDNear, numVAAs: nearPending, txIDBytes: testStandardTxIDLen},
		{chainID: vaa.ChainIDEthereum, numVAAs: ethPending, txIDBytes: testStandardTxIDLen},
	})

	raw, status := publishAndDecodeStatus(t, gov)

	assert.LessOrEqual(t, len(raw), MaxGovernorStatusMessageSize)
	require.Len(t, status.Chains, 2)

	// Deterministic ascending chain ID order: Ethereum (2), then NEAR (15).
	assert.Equal(t, uint32(vaa.ChainIDEthereum), status.Chains[0].ChainId)
	assert.Equal(t, uint64(ethPending), status.Chains[0].Emitters[0].TotalEnqueuedVaas)
	assert.Len(t, status.Chains[0].Emitters[0].EnqueuedVaas, ethPending)

	assert.Equal(t, uint32(vaa.ChainIDNear), status.Chains[1].ChainId)
	assert.Equal(t, uint64(nearPending), status.Chains[1].Emitters[0].TotalEnqueuedVaas)
	assert.Len(t, status.Chains[1].Emitters[0].EnqueuedVaas, nearPending)
}

func TestPublishStatusEnforcesByteCapAndRoundRobinsAcrossChains(t *testing.T) {
	const (
		ethPending  = 200
		nearPending = 4_000
		basePending = 10
	)
	gov := newStatusTestGovernor(t, []statusChainSpec{
		{chainID: vaa.ChainIDBase, numVAAs: basePending, txIDBytes: testStandardTxIDLen},
		{chainID: vaa.ChainIDNear, numVAAs: nearPending, txIDBytes: testStandardTxIDLen},
		{chainID: vaa.ChainIDEthereum, numVAAs: ethPending, txIDBytes: testStandardTxIDLen},
	})

	raw, status := publishAndDecodeStatus(t, gov)

	assert.LessOrEqual(t, len(raw), MaxGovernorStatusMessageSize)
	assert.Greater(t, len(raw), MaxGovernorStatusMessageSize-testMaxEntryOverhead, "expected status to pack close to byte cap")

	require.Len(t, status.Chains, 3)

	// Ethereum (chain 2)
	ethEmitter := status.Chains[0].Emitters[0]
	assert.Equal(t, uint32(vaa.ChainIDEthereum), status.Chains[0].ChainId)
	assert.Equal(t, uint64(ethPending), ethEmitter.TotalEnqueuedVaas)
	assert.Len(t, ethEmitter.EnqueuedVaas, ethPending)

	// NEAR (chain 15) is truncated, preserving strict FIFO prefix order.
	nearEmitter := status.Chains[1].Emitters[0]
	assert.Equal(t, uint32(vaa.ChainIDNear), status.Chains[1].ChainId)
	assert.Equal(t, uint64(nearPending), nearEmitter.TotalEnqueuedVaas)
	assert.Greater(t, len(nearEmitter.EnqueuedVaas), 2_000)
	assert.Less(t, len(nearEmitter.EnqueuedVaas), nearPending)
	for i, v := range nearEmitter.EnqueuedVaas {
		assert.Equal(t, uint64(i), v.Sequence) // #nosec G115 -- loop index is non-negative
	}

	// Base (chain 30) is NOT starved by NEAR despite having a higher chain ID.
	baseEmitter := status.Chains[2].Emitters[0]
	assert.Equal(t, uint32(vaa.ChainIDBase), status.Chains[2].ChainId)
	assert.Equal(t, uint64(basePending), baseEmitter.TotalEnqueuedVaas)
	assert.Len(t, baseEmitter.EnqueuedVaas, basePending)
}

func TestPublishStatusEnforcesByteCapWithLargeTxIDs(t *testing.T) {
	const (
		solanaPending = 2_000
		maxTxPending  = 600
	)
	gov := newStatusTestGovernor(t, []statusChainSpec{
		{chainID: vaa.ChainIDSolana, numVAAs: solanaPending, txIDBytes: testSolanaTxIDLen},
		{chainID: vaa.ChainIDEthereum, numVAAs: maxTxPending, txIDBytes: testMaxTxIDLen},
	})

	raw, status := publishAndDecodeStatus(t, gov)

	assert.LessOrEqual(t, len(raw), MaxGovernorStatusMessageSize)
	require.Len(t, status.Chains, 2)

	solEmitter := status.Chains[0].Emitters[0]
	ethEmitter := status.Chains[1].Emitters[0]

	assert.Equal(t, uint64(solanaPending), solEmitter.TotalEnqueuedVaas)
	assert.Equal(t, uint64(maxTxPending), ethEmitter.TotalEnqueuedVaas)
	assert.NotEmpty(t, solEmitter.EnqueuedVaas)
	assert.NotEmpty(t, ethEmitter.EnqueuedVaas)
	assert.Less(t, len(solEmitter.EnqueuedVaas)+len(ethEmitter.EnqueuedVaas), solanaPending+maxTxPending)
}

func TestBuildStatusPayloadTinyCapPreservesChainSummaries(t *testing.T) {
	const (
		ethPending  = 5
		nearPending = 12
		tinyCap     = 1
	)
	gov := newStatusTestGovernor(t, []statusChainSpec{
		{chainID: vaa.ChainIDEthereum, numVAAs: ethPending, txIDBytes: testStandardTxIDLen},
		{chainID: vaa.ChainIDNear, numVAAs: nearPending, txIDBytes: testStandardTxIDLen},
	})

	hb := &gossipv1.Heartbeat{NodeName: "test-guardian", Timestamp: 1}
	payload := gov.buildStatusPayload(hb, time.Unix(1_700_000_000, 0).UTC(), 1, ethCommon.AddressLength, tinyCap)

	require.Len(t, payload.Chains, 2)
	assert.Equal(t, uint64(ethPending), payload.Chains[0].Emitters[0].TotalEnqueuedVaas)
	assert.Empty(t, payload.Chains[0].Emitters[0].EnqueuedVaas)
	assert.Equal(t, uint64(nearPending), payload.Chains[1].Emitters[0].TotalEnqueuedVaas)
	assert.Empty(t, payload.Chains[1].Emitters[0].EnqueuedVaas)
}
