package guardiand

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type reobservationTestContext struct {
	context.Context
	clock         *clock.Mock
	obsvReqC      chan *gossipv1.ObservationRequest
	chainObsvReqC map[vaa.ChainID]chan *gossipv1.ObservationRequest
}

func setUpReobservationTest() (reobservationTestContext, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	clock := clock.NewMock()

	obsvReqC := make(chan *gossipv1.ObservationRequest)

	chainObsvReqC := make(map[vaa.ChainID]chan *gossipv1.ObservationRequest)
	for i := 0; i < 10; i++ {
		chainObsvReqC[vaa.ChainID(i)] = make(chan *gossipv1.ObservationRequest)
	}

	go handleReobservationRequests(ctx, clock, zap.NewNop(), obsvReqC, chainObsvReqC)

	tc := reobservationTestContext{
		Context:       ctx,
		clock:         clock,
		obsvReqC:      obsvReqC,
		chainObsvReqC: chainObsvReqC,
	}
	return tc, cancel
}

func readFromChannel(parent context.Context, c <-chan *gossipv1.ObservationRequest) (*gossipv1.ObservationRequest, bool) {
	ctx, cancel := context.WithTimeout(parent, 50*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, false
	case r := <-c:
		return r, true
	}
}

func TestReobservationRequest(t *testing.T) {
	ctx, cancel := setUpReobservationTest()
	defer cancel()

	req := &gossipv1.ObservationRequest{
		ChainId: 1,
		TxHash:  []byte{0xe5, 0x9c, 0x1b, 0xe5, 0x0b, 0xe7, 0xe4, 0x7e},
	}

	ctx.obsvReqC <- req

	actual, ok := readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)

	assert.Equal(t, req, actual)
}

func TestDuplicateReobservation(t *testing.T) {
	ctx, cancel := setUpReobservationTest()
	defer cancel()

	req := &gossipv1.ObservationRequest{
		ChainId: 1,
		TxHash:  []byte{0xe5, 0x9c, 0x1b, 0xe5, 0x0b, 0xe7, 0xe4, 0x7e},
	}

	ctx.obsvReqC <- req

	actual, ok := readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)

	// Receiving the same request again should not trigger another re-observation.
	ctx.obsvReqC <- req

	_, ok = readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	assert.False(t, ok)
}

func TestMultipleReobservations(t *testing.T) {
	ctx, cancel := setUpReobservationTest()
	defer cancel()

	req := &gossipv1.ObservationRequest{
		ChainId: 1,
		TxHash:  []byte{0xe5, 0x9c, 0x1b, 0xe5, 0x0b, 0xe7, 0xe4, 0x7e},
	}

	ctx.obsvReqC <- req

	actual, ok := readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)

	// Send a request for the same chain id but different tx hash.
	req.TxHash = []byte{0x6e, 0xf0, 0xa6, 0xba, 0x47, 0x3d, 0x34, 0x51}

	ctx.obsvReqC <- req

	actual, ok = readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)

	// Send a request for the same tx hash but different chain id.
	req.ChainId = 3
	ctx.obsvReqC <- req

	actual, ok = readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)
}

func TestReobserveUnknownChainId(t *testing.T) {
	ctx, cancel := setUpReobservationTest()
	defer cancel()

	req := &gossipv1.ObservationRequest{
		ChainId: uint32(len(ctx.chainObsvReqC)) + 1,
		TxHash:  []byte{0xe5, 0x9c, 0x1b, 0xe5, 0x0b, 0xe7, 0xe4, 0x7e},
	}

	ctx.obsvReqC <- req

	_, ok := readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	assert.False(t, ok)
}

func TestReobservationCacheEviction(t *testing.T) {
	ctx, cancel := setUpReobservationTest()
	defer cancel()

	req := &gossipv1.ObservationRequest{
		ChainId: 1,
		TxHash:  []byte{0xe5, 0x9c, 0x1b, 0xe5, 0x0b, 0xe7, 0xe4, 0x7e},
	}

	ctx.obsvReqC <- req

	actual, ok := readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)

	// Advance the clock by 7.5 minutes, which should trigger the ticker but not cause eviction.
	ctx.clock.Add(7*time.Minute + 30*time.Second)

	// Receiving the same request again should not trigger another re-observation.
	ctx.obsvReqC <- req

	_, ok = readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	assert.False(t, ok)

	// Advance the clock by another 7 minutes, which should evict the re-observation request
	// from the cache.
	ctx.clock.Add(7 * time.Minute)

	// This time the request should be passed through.
	ctx.obsvReqC <- req

	actual, ok = readFromChannel(ctx, ctx.chainObsvReqC[vaa.ChainID(req.ChainId)])
	require.True(t, ok)
	assert.Equal(t, req, actual)
}
