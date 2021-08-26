package guardiand

import (
	"fmt"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

func publicrpcServiceRunnable(logger *zap.Logger, listenAddr string, db *db.Database, gst *common.GuardianSetState) (supervisor.Runnable, *grpc.Server, error) {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}

	logger.Info("publicrpc server listening", zap.String("addr", l.Addr().String()))

	rpcServer := publicrpc.NewPublicrpcServer(logger, db, gst)
	grpcServer := newGRPCServer(logger)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, rpcServer)

	return supervisor.GRPCServer(grpcServer, l, false), grpcServer, nil
}
