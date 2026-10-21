package node

import (
	"fmt"
	"net"
	"os"

	"github.com/certusone/wormhole/node/pkg/adminrpc"
	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/manager"
	"github.com/certusone/wormhole/node/pkg/notary"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"go.uber.org/zap"
)

func adminServiceRunnable(
	logger *zap.Logger,
	socketPath string,
	injectC chan<- *common.MessagePublication,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	delegateSigBroadcastSendC chan<- []byte,
	db *guardianDB.Database,
	gst *common.GuardianSetState,
	gov *governor.ChainGovernor,
	notary *notary.Notary,
	rpcMap map[string]string,
	reobservers interfaces.Reobservers,
	managerSvc *manager.ManagerService,
) (supervisor.Runnable, error) {
	// Delete existing UNIX socket, if present.
	fi, err := os.Stat(socketPath)
	if err == nil {
		fmode := fi.Mode()
		if fmode&os.ModeType == os.ModeSocket {
			err = os.Remove(socketPath)
			if err != nil {
				return nil, fmt.Errorf("failed to remove existing socket at %s: %w", socketPath, err)
			}
		} else {
			return nil, fmt.Errorf("%s is not a UNIX socket", socketPath)
		}
	}

	// Create a new UNIX socket and listen to it.

	// The socket is created with the default umask. We set a restrictive umask in setRestrictiveUmask
	// to ensure that any files we create are only readable by the user - this is much harder to mess up.
	// The umask avoids a race condition between file creation and chmod.

	laddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %v", err)
	}
	l, err := net.ListenUnix("unix", laddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", socketPath, err)
	}

	logger.Info("admin server listening on", zap.String("path", socketPath))

	nodeService := adminrpc.NewPrivService(
		db,
		injectC,
		obsvReqSendC,
		delegateSigBroadcastSendC,
		logger.Named("adminservice"),
		signedInC,
		gov,
		notary,
		rpcMap,
		reobservers,
	)

	publicrpcService := publicrpc.NewPublicrpcServer(logger, db, gst, gov, managerSvc)

	grpcServer := common.NewInstrumentedGRPCServer(logger, common.GrpcLogDetailMinimal)
	nodev1.RegisterNodePrivilegedServiceServer(grpcServer, nodeService)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, publicrpcService)
	return supervisor.GRPCServer(grpcServer, l, false), nil
}
