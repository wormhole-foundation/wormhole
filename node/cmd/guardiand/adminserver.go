package guardiand

import (
	"context"
	"errors"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/db"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"math"
	"net"
	"os"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/certusone/wormhole/node/pkg/common"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

type nodePrivilegedService struct {
	nodev1.UnimplementedNodePrivilegedServiceServer
	injectC chan<- *vaa.VAA
	logger  *zap.Logger
}

// adminGuardianSetUpdateToVAA converts a nodev1.GuardianSetUpdate message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminGuardianSetUpdateToVAA(req *nodev1.GuardianSetUpdate, guardianSetIndex uint32, timestamp uint32) (*vaa.VAA, error) {
	if len(req.Guardians) == 0 {
		return nil, errors.New("empty guardian set specified")
	}

	if len(req.Guardians) > common.MaxGuardianCount {
		return nil, fmt.Errorf("too many guardians - %d, maximum is %d", len(req.Guardians), common.MaxGuardianCount)
	}

	addrs := make([]ethcommon.Address, len(req.Guardians))
	for i, g := range req.Guardians {
		if !ethcommon.IsHexAddress(g.Pubkey) {
			return nil, fmt.Errorf("invalid pubkey format at index %d (%s)", i, g.Name)
		}

		ethAddr := ethcommon.HexToAddress(g.Pubkey)
		for j, pk := range addrs {
			if pk == ethAddr {
				return nil, fmt.Errorf("duplicate pubkey at index %d (duplicate of %d): %s", i, j, g.Name)
			}
		}

		addrs[i] = ethAddr
	}

	v := &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: guardianSetIndex,
		Timestamp:        time.Unix(int64(timestamp), 0),
		Payload: vaa.BodyGuardianSetUpdate{
			Keys:     addrs,
			NewIndex: guardianSetIndex + 1,
		}.Serialize(),
	}

	return v, nil
}

// adminContractUpgradeToVAA converts a nodev1.ContractUpgrade message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminContractUpgradeToVAA(req *nodev1.ContractUpgrade, guardianSetIndex uint32, timestamp uint32) (*vaa.VAA, error) {
	if len(req.NewContract) != 32 {
		return nil, errors.New("invalid new_contract address")
	}

	if req.ChainId > math.MaxUint8 {
		return nil, errors.New("invalid chain_id")
	}

	newContractAddress := vaa.Address{}
	copy(newContractAddress[:], req.NewContract)

	v := &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: guardianSetIndex,
		Timestamp:        time.Unix(int64(timestamp), 0),
		Payload: vaa.BodyContractUpgrade{
			ChainID:     vaa.ChainID(req.ChainId),
			NewContract: newContractAddress,
		}.Serialize(),
	}

	return v, nil
}

func (s *nodePrivilegedService) InjectGovernanceVAA(ctx context.Context, req *nodev1.InjectGovernanceVAARequest) (*nodev1.InjectGovernanceVAAResponse, error) {
	s.logger.Info("governance VAA injected via admin socket", zap.String("request", req.String()))

	var (
		v   *vaa.VAA
		err error
	)
	switch payload := req.Payload.(type) {
	case *nodev1.InjectGovernanceVAARequest_GuardianSet:
		v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, req.CurrentSetIndex, req.Timestamp)
	case *nodev1.InjectGovernanceVAARequest_ContractUpgrade:
		v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, req.CurrentSetIndex, req.Timestamp)
	default:
		panic(fmt.Sprintf("unsupported VAA type: %T", payload))
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Generate digest of the unsigned VAA.
	digest, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}

	s.logger.Info("governance VAA constructed",
		zap.Any("vaa", v),
		zap.String("digest", digest.String()),
	)

	s.injectC <- v

	return &nodev1.InjectGovernanceVAAResponse{Digest: digest.Bytes()}, nil
}

func adminServiceRunnable(logger *zap.Logger, socketPath string, injectC chan<- *vaa.VAA, db *db.Database, gst *common.GuardianSetState) (supervisor.Runnable, error) {
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

	nodeService := &nodePrivilegedService{
		injectC: injectC,
		logger:  logger.Named("adminservice"),
	}

	publicrpcService := publicrpc.NewPublicrpcServer(logger, db, gst)

	grpcServer := newGRPCServer(logger)
	nodev1.RegisterNodePrivilegedServiceServer(grpcServer, nodeService)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, publicrpcService)
	return supervisor.GRPCServer(grpcServer, l, false), nil
}

func newGRPCServer(logger *zap.Logger) *grpc.Server {
	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(logger),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(logger),
		)),
	)

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(server)

	return server
}
