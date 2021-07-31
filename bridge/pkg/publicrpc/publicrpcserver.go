package publicrpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/db"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicrpcServer implements the publicrpc gRPC service.
type PublicrpcServer struct {
	publicrpcv1.UnimplementedPublicrpcServer
	rawHeartbeatListeners *RawHeartbeatConns
	logger                *zap.Logger
	db                    *db.Database
}

func NewPublicrpcServer(logger *zap.Logger, rawHeartbeatListeners *RawHeartbeatConns, db *db.Database) *PublicrpcServer {
	return &PublicrpcServer{
		rawHeartbeatListeners: rawHeartbeatListeners,
		logger:                logger.Named("publicrpcserver"),
		db:                    db,
	}
}

func (s *PublicrpcServer) GetRawHeartbeats(req *publicrpcv1.GetRawHeartbeatsRequest, stream publicrpcv1.Publicrpc_GetRawHeartbeatsServer) error {
	s.logger.Info("gRPC heartbeat stream opened by client")

	// create a channel and register it for heartbeats
	receiveChan := make(chan *gossipv1.Heartbeat, 50)
	// clientId is the reference to the subscription that we will use for unsubscribing when the client disconnects.
	clientId := s.rawHeartbeatListeners.subscribeHeartbeats(receiveChan)

	for {
		select {
		// Exit on stream context done
		case <-stream.Context().Done():
			s.logger.Info("raw heartbeat stream closed by client", zap.Int("clientId", clientId))
			s.rawHeartbeatListeners.unsubscribeHeartbeats(clientId)
			return stream.Context().Err()
		case msg := <-receiveChan:
			stream.Send(msg)
		}
	}
}

func (s *PublicrpcServer) GetSignedVAA(ctx context.Context, req *publicrpcv1.GetSignedVAARequest) (*publicrpcv1.GetSignedVAAResponse, error) {
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
		EmitterChain:   vaa.ChainID(req.MessageId.EmitterChain.Number()),
		EmitterAddress: addr,
		Sequence:       uint64(req.MessageId.Sequence),
	})

	if err != nil {
		if err == db.ErrVAANotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		s.logger.Error("failed to fetch VAA", zap.Error(err), zap.Any("request", req))
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &publicrpcv1.GetSignedVAAResponse{
		VaaBytes: b,
	}, nil
}
