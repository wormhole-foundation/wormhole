package publicrpc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicrpcServer implements the publicrpc gRPC service.
type PublicrpcServer struct {
	publicrpcv1.UnsafePublicRPCServiceServer
	logger *zap.Logger
	db     *db.Database
	gst    *common.GuardianSetState
	gov    *governor.ChainGovernor
}

func NewPublicrpcServer(
	logger *zap.Logger,
	db *db.Database,
	gst *common.GuardianSetState,
	gov *governor.ChainGovernor,
) *PublicrpcServer {
	return &PublicrpcServer{
		logger: logger.Named("publicrpcserver"),
		db:     db,
		gst:    gst,
		gov:    gov,
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
				P2PNodeAddr:          peerId.String(),
				RawHeartbeat:         hb,
			})
		}
	}

	return resp, nil
}

func (s *PublicrpcServer) GetSignedVAA(ctx context.Context, req *publicrpcv1.GetSignedVAARequest) (*publicrpcv1.GetSignedVAAResponse, error) {
	if req.MessageId == nil {
		return nil, status.Error(codes.InvalidArgument, "no message ID specified")
	}

	if req.MessageId.GetEmitterChain() > math.MaxUint16 {
		return nil, status.Error(codes.InvalidArgument, "emitter chain id must be no greater than 16 bits")
	}

	chainID := vaa.ChainID(req.MessageId.GetEmitterChain()) // #nosec G115 -- This conversion is checked above

	// This interface is not supported for PythNet messages because those VAAs are not stored in the database.
	if chainID == vaa.ChainIDPythNet {
		return nil, status.Error(codes.InvalidArgument, "not supported for PythNet")
	}

	address, err := hex.DecodeString(req.MessageId.EmitterAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode address: %v", err))
	}
	if len(address) != 32 {
		return nil, status.Error(codes.InvalidArgument, "address must be 32 bytes")
	}

	addr := vaa.Address{}
	copy(addr[:], address)

	b, err := s.db.GetSignedVAABytes(db.VAAID{
		EmitterChain:   chainID,
		EmitterAddress: addr,
		Sequence:       req.MessageId.Sequence,
	})

	if err != nil {
		if errors.Is(err, db.ErrVAANotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		s.logger.Error("failed to fetch VAA", zap.Error(err), zap.Any("request", req))
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &publicrpcv1.GetSignedVAAResponse{
		VaaBytes: b,
	}, nil
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

func (s *PublicrpcServer) GovernorGetAvailableNotionalByChain(ctx context.Context, req *publicrpcv1.GovernorGetAvailableNotionalByChainRequest) (*publicrpcv1.GovernorGetAvailableNotionalByChainResponse, error) {
	resp := &publicrpcv1.GovernorGetAvailableNotionalByChainResponse{}

	if s.gov != nil {
		resp.Entries = s.gov.GetAvailableNotionalByChain()
	} else {
		resp.Entries = make([]*publicrpcv1.GovernorGetAvailableNotionalByChainResponse_Entry, 0)
	}

	return resp, nil
}

func (s *PublicrpcServer) GovernorGetEnqueuedVAAs(ctx context.Context, req *publicrpcv1.GovernorGetEnqueuedVAAsRequest) (*publicrpcv1.GovernorGetEnqueuedVAAsResponse, error) {
	resp := &publicrpcv1.GovernorGetEnqueuedVAAsResponse{}

	if s.gov != nil {
		resp.Entries = s.gov.GetEnqueuedVAAs()
	} else {
		resp.Entries = make([]*publicrpcv1.GovernorGetEnqueuedVAAsResponse_Entry, 0)
	}

	return resp, nil
}

func (s *PublicrpcServer) GovernorIsVAAEnqueued(ctx context.Context, req *publicrpcv1.GovernorIsVAAEnqueuedRequest) (*publicrpcv1.GovernorIsVAAEnqueuedResponse, error) {
	resp := &publicrpcv1.GovernorIsVAAEnqueuedResponse{}

	if req.MessageId == nil {
		return nil, status.Error(codes.InvalidArgument, "no message ID specified")
	}

	if s.gov != nil {
		var err error
		resp.IsEnqueued, err = s.gov.IsVAAEnqueued(req.MessageId)
		if err != nil {
			return resp, err
		}
	} else {
		resp.IsEnqueued = false
	}

	return resp, nil
}

func (s *PublicrpcServer) GovernorGetTokenList(ctx context.Context, req *publicrpcv1.GovernorGetTokenListRequest) (*publicrpcv1.GovernorGetTokenListResponse, error) {
	resp := &publicrpcv1.GovernorGetTokenListResponse{}

	if s.gov != nil {
		resp.Entries = s.gov.GetTokenList()
	} else {
		resp.Entries = make([]*publicrpcv1.GovernorGetTokenListResponse_Entry, 0)
	}

	return resp, nil
}
