package publicrpc

import (
	"context"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestGetSignedVAA(t *testing.T) {
	type test struct {
		label string
		req   publicrpcv1.GetSignedVAARequest
		err   error
	}

	tests := []test{
		{label: "no message",
			req: publicrpcv1.GetSignedVAARequest{},
			err: status.Error(codes.InvalidArgument, "no message ID specified")},
		{label: "empty message",
			req: publicrpcv1.GetSignedVAARequest{MessageId: &publicrpcv1.MessageID{}},
			err: status.Error(codes.InvalidArgument, "address must be 32 bytes")},
		{label: "invalid address",
			req: publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(uint32(1)),
					EmitterAddress: "AAAA",
					Sequence:       uint64(1),
				},
			},
			err: status.Error(codes.InvalidArgument, "address must be 32 bytes")},
		{label: "no emitter chain",
			req: publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterAddress: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					Sequence:       uint64(1),
				},
			},
			err: status.Error(codes.InvalidArgument, "invalid chain specified")},
		{label: "invalid emitter chain",
			req: publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(uint32(500)),
					EmitterAddress: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					Sequence:       uint64(1),
				},
			},
			err: status.Error(codes.InvalidArgument, "invalid chain specified")},
		{label: "no sequence",
			req: publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(uint32(1)),
					EmitterAddress: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				},
			},
			err: status.Error(codes.InvalidArgument, "no sequence specified")},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := zap.NewProduction()
			server := &PublicrpcServer{logger: logger}

			resp, err := server.GetSignedVAA(ctx, &tc.req)
			assert.Nil(t, resp)
			assert.Equal(t, tc.err, err)
		})
	}

}
