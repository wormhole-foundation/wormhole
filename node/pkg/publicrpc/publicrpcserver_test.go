package publicrpc

import (
	"context"
	"testing"

	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetSignedVAA(t *testing.T) {
	type test struct {
		label     string
		req       *publicrpcv1.GetSignedVAARequest
		errString string
	}

	tests := []test{
		{label: "no message",
			req:       &publicrpcv1.GetSignedVAARequest{},
			errString: "rpc error: code = InvalidArgument desc = no message ID specified"},
		{label: "empty message",
			req:       &publicrpcv1.GetSignedVAARequest{MessageId: &publicrpcv1.MessageID{}},
			errString: "rpc error: code = InvalidArgument desc = address must be 32 bytes"},
		{label: "invalid address",
			req: &publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(uint32(1)),
					EmitterAddress: "AAAA",
					Sequence:       uint64(1),
				},
			},
			errString: "rpc error: code = InvalidArgument desc = address must be 32 bytes"},
		{label: "no emitter chain",
			req: &publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterAddress: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					Sequence:       uint64(1),
				},
			},
			errString: "rpc error: code = InvalidArgument desc = invalid chain specified"},
		{label: "invalid emitter chain",
			req: &publicrpcv1.GetSignedVAARequest{
				MessageId: &publicrpcv1.MessageID{
					EmitterChain:   publicrpcv1.ChainID(uint32(500)),
					EmitterAddress: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					Sequence:       uint64(1),
				},
			},
			errString: "rpc error: code = InvalidArgument desc = invalid chain specified"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := zap.NewProduction()
			server := &PublicrpcServer{logger: logger}

			resp, err := server.GetSignedVAA(ctx, tc.req)
			assert.Nil(t, resp)

			if tc.errString == "" {
				assert.Equal(t, nil, err)
			} else {
				assert.Equal(t, tc.errString, err.Error())
			}
		})
	}

}
