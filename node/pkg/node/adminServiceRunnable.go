package node

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/certusone/wormhole/node/pkg/adminrpc"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"go.uber.org/zap"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func adminServiceRunnable(
	logger *zap.Logger,
	socketPath string,
	injectC chan<- *common.MessagePublication,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	db *db.Database,
	gst *common.GuardianSetState,
	gov *governor.ChainGovernor,
	gk *ecdsa.PrivateKey,
	ethRpc *string,
	ethContract *string,
	rpcMap map[string]string,
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var evmConnector connectors.Connector
	if ethRpc != nil && ethContract != nil {
		contract := ethcommon.HexToAddress(*ethContract)
		evmConnector, err = connectors.NewEthereumBaseConnector(ctx, "eth", *ethRpc, contract, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to connecto to ethereum")
		}
	}

	nodeService := adminrpc.NewPrivService(
		db,
		injectC,
		obsvReqSendC,
		logger.Named("adminservice"),
		signedInC,
		gov,
		evmConnector,
		gk,
		ethcrypto.PubkeyToAddress(gk.PublicKey),
		rpcMap,
	)

	publicrpcService := publicrpc.NewPublicrpcServer(logger, db, gst, gov)

	grpcServer := common.NewInstrumentedGRPCServer(logger, common.GrpcLogDetailMinimal)
	nodev1.RegisterNodePrivilegedServiceServer(grpcServer, nodeService)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, publicrpcService)
	return supervisor.GRPCServer(grpcServer, l, false), nil
}
