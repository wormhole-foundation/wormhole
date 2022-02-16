package publicrpc

import (
	"context"
	"github.com/certusone/wormhole/node/pkg/common"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PublicrpcServer struct {
	publicrpcv1.UnsafePublicRPCServiceServer
	logger *zap.Logger
	gst    *common.GuardianSetState
}

func NewPublicrpcServer(
	logger *zap.Logger,
	gst *common.GuardianSetState,
) *PublicrpcServer {
	return &PublicrpcServer{
		logger: logger.Named("publicrpcserver"),
		gst:    gst,
	}
}

func (s *PublicrpcServer) GetLastHeartbeats(ctx context.Context, req *publicrpcv1.GetLastHeartbeatsRequest) (*publicrpcv1.GetLastHeartbeatsResponse, error) {
	gs := s.gst.Get()
	if gs == nil {
		return nil, status.Error(codes.Unavailable, "guardian set not fetched from chain yet")
	}

	resp := &publicrpcv1.GetLastHeartbeatsResponse{
		Entries: make([]*publicrpcv1.GetLastHeartbeatsResponse_Entry, 0),
	}

	// Fetch all heartbeats (including from nodes not in the guardian set - which
	// can happen either with --disableHeartbeatVerify or when the guardian set changes)
	for addr, v := range s.gst.GetAll() {
		for peerId, hb := range v {
			resp.Entries = append(resp.Entries, &publicrpcv1.GetLastHeartbeatsResponse_Entry{
				VerifiedGuardianAddr: addr.Hex(),
				P2PNodeAddr:          peerId.Pretty(),
				RawHeartbeat:         hb,
			})
		}
	}

	return resp, nil
}

func (s *PublicrpcServer) GetCurrentGuardianSet(ctx context.Context, req *publicrpcv1.GetCurrentGuardianSetRequest) (*publicrpcv1.GetCurrentGuardianSetResponse, error) {
	gs := s.gst.Get()
	if gs == nil {
		return nil, status.Error(codes.Unavailable, "guardian set not fetched from chain yet")
	}

	resp := &publicrpcv1.GetCurrentGuardianSetResponse{
		GuardianSet: &publicrpcv1.GuardianSet{
			Index:     gs.Index,
			Addresses: make([]string, len(gs.Keys)),
		},
	}

	for i, v := range gs.Keys {
		resp.GuardianSet.Addresses[i] = v.Hex()
	}

	return resp, nil
}
