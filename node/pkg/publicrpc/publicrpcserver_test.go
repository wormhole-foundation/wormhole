package publicrpc

import (
	"context"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetSignedVAANoMessage(t *testing.T) {
	msg := publicrpcv1.GetSignedVAARequest{}
	ctx := context.Background()

	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger}

	resp, err := server.GetSignedVAA(ctx, &msg)
	assert.Nil(t, resp)

	expected_err := status.Error(codes.InvalidArgument, "no message ID specified")
	assert.Equal(t, expected_err, err)
}

func TestGetSignedVAANoAddress(t *testing.T) {
	msg := publicrpcv1.GetSignedVAARequest{MessageId: &publicrpcv1.MessageID{}}
	ctx := context.Background()

	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger}

	resp, err := server.GetSignedVAA(ctx, &msg)
	assert.Nil(t, resp)

	expected_err := status.Error(codes.InvalidArgument, "VAA ID emitter address must be 32 bytes")
	assert.Equal(t, expected_err, err)
}

func TestGetSignedVAABadAddress(t *testing.T) {
	chainID := uint32(1)
	emitterAddr := "AAAA"
	seq := uint64(1)

	msg := publicrpcv1.GetSignedVAARequest{
		MessageId: &publicrpcv1.MessageID{
			EmitterChain:   publicrpcv1.ChainID(chainID),
			EmitterAddress: emitterAddr,
			Sequence:       seq,
		},
	}

	ctx := context.Background()

	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger}

	resp, err := server.GetSignedVAA(ctx, &msg)
	assert.Nil(t, resp)

	expected_err := status.Error(codes.InvalidArgument, "VAA ID emitter address must be 32 bytes")
	assert.Equal(t, expected_err, err)
}

func TestGovernorIsVAAEnqueuedNoMessage(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	gov := governor.NewChainGovernor(logger, nil, common.GoTest, false, "")
	server := &PublicrpcServer{logger: logger, gov: gov}

	// A message without the messageId set should not panic but return an error instead.
	msg := publicrpcv1.GovernorIsVAAEnqueuedRequest{}
	assert.NotPanics(t, func() {
		_, err := server.GovernorIsVAAEnqueued(ctx, &msg)
		assert.Error(t, err)
		expected_err := status.Error(codes.InvalidArgument, "no message ID specified")
		assert.Equal(t, expected_err, err)
	})
}

func TestGetLastHeartbeatsNilGuardianSet(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	gst := common.NewGuardianSetState(nil)
	server := &PublicrpcServer{logger: logger, gst: gst}

	resp, err := server.GetLastHeartbeats(ctx, &publicrpcv1.GetLastHeartbeatsRequest{})
	assert.Nil(t, resp)
	expected_err := status.Error(codes.Unavailable, "guardian set not fetched from chain yet")
	assert.Equal(t, expected_err, err)
}

func TestGetCurrentGuardianSetNilGuardianSet(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	gst := common.NewGuardianSetState(nil)
	server := &PublicrpcServer{logger: logger, gst: gst}

	resp, err := server.GetCurrentGuardianSet(ctx, &publicrpcv1.GetCurrentGuardianSetRequest{})
	assert.Nil(t, resp)
	expected_err := status.Error(codes.Unavailable, "guardian set not fetched from chain yet")
	assert.Equal(t, expected_err, err)
}

func TestGovernorGetAvailableNotionalByChainNilGovernor(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, gov: nil}

	resp, err := server.GovernorGetAvailableNotionalByChain(ctx, &publicrpcv1.GovernorGetAvailableNotionalByChainRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Entries)
}

func TestGovernorGetEnqueuedVAAsNilGovernor(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, gov: nil}

	resp, err := server.GovernorGetEnqueuedVAAs(ctx, &publicrpcv1.GovernorGetEnqueuedVAAsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Entries)
}

func TestGovernorGetTokenListNilGovernor(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, gov: nil}

	resp, err := server.GovernorGetTokenList(ctx, &publicrpcv1.GovernorGetTokenListRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Entries)
}

func TestGetSignedManagerTransactionNilManager(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, manager: nil}

	resp, err := server.GetSignedManagerTransaction(ctx, &publicrpcv1.GetSignedManagerTransactionRequest{})
	assert.Nil(t, resp)
	expected_err := status.Error(codes.Unavailable, "manager service not enabled")
	assert.Equal(t, expected_err, err)
}

func TestGetSignedManagerTransactionByHashNilManager(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, manager: nil}

	resp, err := server.GetSignedManagerTransactionByHash(ctx, &publicrpcv1.GetSignedManagerTransactionByHashRequest{})
	assert.Nil(t, resp)
	expected_err := status.Error(codes.Unavailable, "manager service not enabled")
	assert.Equal(t, expected_err, err)
}

func TestAggregatedTxToResponse(t *testing.T) {
	aggTx := &guardianDB.AggregatedTransaction{
		VAAHash:          []byte{0x01, 0x02, 0x03},
		VAAID:            "1/0000000000000000000000000000000000000000000000000000000000000001/42",
		DestinationChain: vaa.ChainIDEthereum,
		ManagerSetIndex:  5,
		Required:         3,
		Total:            5,
		Signatures: map[uint8][][]byte{
			0: {{0xAA, 0xBB}},
			1: {{0xCC, 0xDD}},
		},
	}

	resp := aggregatedTxToResponse(aggTx)
	assert.Equal(t, "010203", resp.VaaHash)
	assert.Equal(t, aggTx.VAAID, resp.VaaId)
	assert.Equal(t, uint32(aggTx.DestinationChain), resp.DestinationChain)
	assert.Equal(t, aggTx.ManagerSetIndex, resp.ManagerSetIndex)
	assert.Equal(t, uint32(aggTx.Required), resp.Required)
	assert.Equal(t, uint32(aggTx.Total), resp.Total)
	assert.False(t, resp.IsComplete)
	assert.Len(t, resp.Signatures, 2)
}

func TestAggregatedTxToByHashResponse(t *testing.T) {
	aggTx := &guardianDB.AggregatedTransaction{
		VAAHash:          []byte{0x04, 0x05, 0x06},
		VAAID:            "2/0000000000000000000000000000000000000000000000000000000000000002/99",
		DestinationChain: vaa.ChainIDBSC,
		ManagerSetIndex:  10,
		Required:         1,
		Total:            3,
		Signatures: map[uint8][][]byte{
			0: {{0xEE, 0xFF}},
		},
	}

	resp := aggregatedTxToByHashResponse(aggTx)
	assert.Equal(t, "040506", resp.VaaHash)
	assert.Equal(t, aggTx.VAAID, resp.VaaId)
	assert.Equal(t, uint32(aggTx.DestinationChain), resp.DestinationChain)
	assert.Equal(t, aggTx.ManagerSetIndex, resp.ManagerSetIndex)
	assert.Equal(t, uint32(aggTx.Required), resp.Required)
	assert.Equal(t, uint32(aggTx.Total), resp.Total)
	assert.True(t, resp.IsComplete)
	assert.Len(t, resp.Signatures, 1)
}

func TestGetSignedVAAPythNet(t *testing.T) {
	chainID := uint32(vaa.ChainIDPythNet)
	emitterAddr := "0000000000000000000000000000000000000000000000000000000000000001"
	seq := uint64(1)

	msg := publicrpcv1.GetSignedVAARequest{
		MessageId: &publicrpcv1.MessageID{
			EmitterChain:   publicrpcv1.ChainID(chainID),
			EmitterAddress: emitterAddr,
			Sequence:       seq,
		},
	}

	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger}

	resp, err := server.GetSignedVAA(ctx, &msg)
	assert.Nil(t, resp)
	expected_err := status.Error(codes.InvalidArgument, "not supported for PythNet")
	assert.Equal(t, expected_err, err)
}

func TestGovernorIsVAAEnqueuedNilGovernor(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	server := &PublicrpcServer{logger: logger, gov: nil}

	msg := publicrpcv1.GovernorIsVAAEnqueuedRequest{
		MessageId: &publicrpcv1.MessageID{
			EmitterChain:   1,
			EmitterAddress: "0000000000000000000000000000000000000000000000000000000000000001",
			Sequence:       1,
		},
	}
	resp, err := server.GovernorIsVAAEnqueued(ctx, &msg)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.IsEnqueued)
}
