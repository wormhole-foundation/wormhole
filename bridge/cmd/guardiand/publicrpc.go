package guardiand

import (
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/db"
	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/publicrpc"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

func publicrpcServiceRunnable(logger *zap.Logger, listenAddr string, hl *publicrpc.RawHeartbeatConns, db *db.Database, gst *common.GuardianSetState) (supervisor.Runnable, *grpc.Server, error) {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}

	logger.Info("publicrpc server listening", zap.String("addr", l.Addr().String()))

	rpcServer := publicrpc.NewPublicrpcServer(logger, hl, db, gst)
	grpcServer := newGRPCServer(logger)
	publicrpcv1.RegisterPublicrpcServer(grpcServer, rpcServer)

	return supervisor.GRPCServer(grpcServer, l, false), grpcServer, nil
}
