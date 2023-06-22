package publicrpc

import (
	"context"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/stretchr/testify/assert"
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

	expected_err := status.Error(codes.InvalidArgument, "address must be 32 bytes")
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

	expected_err := status.Error(codes.InvalidArgument, "address must be 32 bytes")
	assert.Equal(t, expected_err, err)
}

func TestGovernorIsVAAEnqueuedNoMessage(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	gov := governor.NewChainGovernor(logger, nil, common.GoTest)
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
