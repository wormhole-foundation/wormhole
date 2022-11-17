package guardiand

import (
	"fmt"
	"net"
	"os"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func publicrpcTcpServiceRunnable(logger *zap.Logger, listenAddr string, db *db.Database, gst *common.GuardianSetState, gov *governor.ChainGovernor) (supervisor.Runnable, error) {
	l, err := net.Listen("tcp", listenAddr)

	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	logger.Info("publicrpc server listening", zap.String("addr", l.Addr().String()))

	rpcServer := publicrpc.NewPublicrpcServer(logger, db, gst, gov)
	grpcServer := common.NewInstrumentedGRPCServer(logger)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, rpcServer)

	return supervisor.GRPCServer(grpcServer, l, false), nil
}

func publicrpcUnixServiceRunnable(logger *zap.Logger, socketPath string, db *db.Database, gst *common.GuardianSetState, gov *governor.ChainGovernor) (supervisor.Runnable, *grpc.Server, error) {
	// Delete existing UNIX socket, if present.
	fi, err := os.Stat(socketPath)
	if err == nil {
		fmode := fi.Mode()
		if fmode&os.ModeType == os.ModeSocket {
			err = os.Remove(socketPath)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to remove existing socket at %s: %w", socketPath, err)
			}
		} else {
			return nil, nil, fmt.Errorf("%s is not a UNIX socket", socketPath)
		}
	}

	// Create a new UNIX socket and listen to it.

	// The socket is created with the default umask. We set a restrictive umask in setRestrictiveUmask
	// to ensure that any files we create are only readable by the user - this is much harder to mess up.
	// The umask avoids a race condition between file creation and chmod.

	laddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid listen address: %v", err)
	}
	l, err := net.ListenUnix("unix", laddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on %s: %w", socketPath, err)
	}

	logger.Info("publicrpc (unix socket) server listening on", zap.String("path", socketPath))

	publicrpcService := publicrpc.NewPublicrpcServer(logger, db, gst, gov)

	grpcServer := common.NewInstrumentedGRPCServer(logger)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, publicrpcService)
	return supervisor.GRPCServer(grpcServer, l, false), grpcServer, nil
}
