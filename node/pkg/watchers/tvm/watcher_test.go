package tvm

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xssnick/tonutils-go/address"

	"go.uber.org/zap/zaptest"
)

const (
	TestContractAddressRAW = "EQBJjO0MsE60REHkAoKBWts9y_tH0mow08qQ__aX-kXLHKll"
	TestStartLT            = uint64(39540691000000)
	TestReobsTxIDHex       = "6c5eb2129e4f93307b73eae480df6a42654e9135b309700ee2879182db9e02a5"

	TestTimeout = 25 * time.Second
)

func TestWatcher_Subscription_Reobservation_Head(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 16)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 16)

	cfg := WatcherConfig{
		NetworkID:       "ton-testnet",
		ChainID:         vaa.ChainIDTON,
		ConfigURL:       "https://ton.org/testnet-global.config.json",
		ContractAddress: TestContractAddressRAW,
	}

	addr := address.MustParseAddr(TestContractAddressRAW)

	w := NewWatcher(cfg.ChainID, cfg.ConfigURL, TestStartLT, addr, msgC, obsvReqC)

	rootCtx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	logger := zaptest.NewLogger(t)

	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "tvmwatch", w.Run); err != nil {
			assert.NoError(t, err)
			return err
		}
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		<-rootCtx.Done()
		supervisor.Signal(ctx, supervisor.SignalDone)
		return nil
	}, supervisor.WithPropagatePanic)

	// 1 - subscription
	subCtx, subCancel := context.WithTimeout(rootCtx, 20*time.Second)
	defer subCancel()

	first := mustRecvMsg(t, subCtx, msgC)
	t.Logf("subscription: got message seq=%d ts=%s", first.Sequence, first.Timestamp)

	//2 - reobservation
	reobsTxID := mustHex(t, TestReobsTxIDHex)

	req := &gossipv1.ObservationRequest{
		ChainId:   uint32(vaa.ChainIDTON),
		TxHash:    reobsTxID,
		Timestamp: time.Now().UnixNano(),
	}

	select {
	case obsvReqC <- req:
	default:
		t.Fatalf("obsvReqC is full")
	}

	waitObsvDrained(t, obsvReqC, 3*time.Second)

	reobsCtx, reobsCancel := context.WithTimeout(rootCtx, 12*time.Second)
	defer reobsCancel()

	reobservation := mustRecvReobservateMsg(t, reobsCtx, msgC)
	assert.True(t, reobservation.IsReobservation, "expected IsReobservation=true for re-observed message")
	assert.Equal(t, vaa.ChainIDTON, reobservation.EmitterChain, "chain mismatch in re-observed message")

	//3 - check get_block_height
	time.Sleep(1 * time.Second)
	assert.NotZero(t, w.CurrentHeight, "current height is zero")
}

func mustHex(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	require.NoError(t, err, "bad hex: %s", s)
	require.Len(t, b, 32, "TxID must be 32 bytes")
	return b
}

func mustRecvMsg(t *testing.T, ctx context.Context, ch <-chan *common.MessagePublication) *common.MessagePublication {
	t.Helper()
	select {
	case msg := <-ch:
		require.NotNil(t, msg)
		return msg
	case <-ctx.Done():
		t.Fatalf("timeout while waiting for message (%v)", ctx.Err())
		return nil
	}

}

func mustRecvReobservateMsg(t *testing.T, ctx context.Context, ch <-chan *common.MessagePublication) *common.MessagePublication {
	t.Helper()
	for {
		select {
		case msg := <-ch:
			require.NotNil(t, msg)
			if msg.IsReobservation {
				return msg
			}
		case <-ctx.Done():
			t.Fatalf("timeout while waiting for message (%v)", ctx.Err())
			return nil
		}
	}
}

func waitObsvDrained(t *testing.T, ch <-chan *gossipv1.ObservationRequest, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if len(ch) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("obsvReqC is not drained (len=%d) after %s", len(ch), d)
}
