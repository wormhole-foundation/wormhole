package publicrpc

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
)

// gRPC server & method for handling streaming proto connection
type publicrpcServer struct {
	publicrpcv1.UnimplementedPublicrpcServer
	rawHeartbeatListeners *PublicRawHeartbeatConnections
	logger                *zap.Logger
}

func (s *publicrpcServer) GetRawHeartbeats(req *publicrpcv1.GetRawHeartbeatsRequest, stream publicrpcv1.Publicrpc_GetRawHeartbeatsServer) error {
	s.logger.Info("gRPC heartbeat stream opened by client")

	// create a channel and register it for heartbeats
	receiveChan := make(chan *publicrpcv1.Heartbeat, 50)
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

func PublicrpcServiceRunnable(logger *zap.Logger, listenAddr string, rawHeartbeatListeners *PublicRawHeartbeatConnections) supervisor.Runnable {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("failed to listen for publicrpc service", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("publicrpc server listening on %s", listenAddr))

	rpcServer := &publicrpcServer{
		rawHeartbeatListeners: rawHeartbeatListeners,
		logger:                logger.Named("publicrpcserver"),
	}

	grpcServer := grpc.NewServer()
	publicrpcv1.RegisterPublicrpcServer(grpcServer, rpcServer)
	return supervisor.GRPCServer(grpcServer, l, false)
}
