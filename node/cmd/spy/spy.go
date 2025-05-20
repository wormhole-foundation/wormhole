package spy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
)

var (
	envStr *string

	p2pNetworkID   *string
	p2pPort        *uint
	p2pBootstrap   *string
	protectedPeers []string

	statusAddr *string

	nodeKeyPath *string

	logLevel *string

	spyRPC *string

	sendTimeout *time.Duration

	ethRPC      *string
	ethContract *string
)

func init() {
	envStr = SpyCmd.Flags().String("env", "", `environment (may be "testnet" or "mainnet", required unless "--bootstrap" is specified)`)
	p2pNetworkID = SpyCmd.Flags().String("network", "", "P2P network identifier (optional for testnet or mainnet, overrides default, required for devnet)")
	p2pPort = SpyCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = SpyCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (optional for testnet or mainnet, overrides default, required for devnet)")
	SpyCmd.Flags().StringSliceVarP(&protectedPeers, "protectedPeers", "", []string{}, "")

	statusAddr = SpyCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = SpyCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	logLevel = SpyCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	spyRPC = SpyCmd.Flags().String("spyRPC", "", "Listen address for gRPC interface")

	sendTimeout = SpyCmd.Flags().Duration("sendTimeout", 5*time.Second, "Timeout for sending a message to a subscriber")

	ethRPC = SpyCmd.Flags().String("ethRPC", "", "Ethereum RPC for verifying VAAs (optional)")
	ethContract = SpyCmd.Flags().String("ethContract", "", "Ethereum core bridge address for verifying VAAs (required if ethRPC is specified)")
}

// SpyCmd represents the node command
var SpyCmd = &cobra.Command{
	Use:   "spy",
	Short: "Run gossip spy client",
	Run:   runSpy,
}

type spyServer struct {
	spyv1.UnimplementedSpyRPCServiceServer
	logger          *zap.Logger
	subsSignedVaa   map[string]*subscriptionSignedVaa
	subsSignedVaaMu sync.Mutex
	vaaVerifier     *VaaVerifier
}

type message struct {
	vaaBytes []byte
}

type filterSignedVaa struct {
	chainId     vaa.ChainID
	emitterAddr vaa.Address
}
type subscriptionSignedVaa struct {
	filters []filterSignedVaa
	ch      chan message
}

func subscriptionId() string {
	return uuid.New().String()
}

func (s *spyServer) PublishSignedVAA(vaaBytes []byte) error {
	s.subsSignedVaaMu.Lock()
	defer s.subsSignedVaaMu.Unlock()

	var v *vaa.VAA
	var err error
	verified := s.vaaVerifier == nil

	for _, sub := range s.subsSignedVaa {
		if len(sub.filters) == 0 {
			if !verified {
				verified = true
				v, err = s.verifyVAA(v, vaaBytes)
				if err != nil {
					return err
				}
			}
			sub.ch <- message{vaaBytes: vaaBytes}
			continue
		}

		if v == nil {
			v, err = vaa.Unmarshal(vaaBytes)
			if err != nil {
				return err
			}
		}

		for _, fi := range sub.filters {
			if fi.chainId == v.EmitterChain && fi.emitterAddr == v.EmitterAddress {
				if !verified {
					verified = true
					v, err = s.verifyVAA(v, vaaBytes)
					if err != nil {
						return err
					}
				}
				sub.ch <- message{vaaBytes: vaaBytes}
			}
		}

	}

	return nil
}

func (s *spyServer) verifyVAA(v *vaa.VAA, vaaBytes []byte) (*vaa.VAA, error) {
	if s.vaaVerifier == nil {
		panic("verifier is nil")
	}

	if v == nil {
		var err error
		v, err = vaa.Unmarshal(vaaBytes)
		if err != nil {
			return v, fmt.Errorf(`failed to unmarshal VAA: %w`, err)
		}
	}

	valid, err := s.vaaVerifier.VerifySignatures(v)
	if err != nil {
		return v, fmt.Errorf(`failed to verify VAA: %w`, err)
	}

	if !valid {
		return v, errors.New(`invalid VAA signature`)
	}

	return v, nil
}

func (s *spyServer) SubscribeSignedVAA(req *spyv1.SubscribeSignedVAARequest, resp spyv1.SpyRPCService_SubscribeSignedVAAServer) error {
	var fi []filterSignedVaa
	if req.Filters != nil {
		for _, f := range req.Filters {
			switch t := f.Filter.(type) {
			case *spyv1.FilterEntry_EmitterFilter:
				addr, err := vaa.StringToAddress(t.EmitterFilter.EmitterAddress)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode emitter address: %v", err))
				}
				if t.EmitterFilter.GetChainId() > math.MaxUint16 {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("emitter chain id must be a valid 16 bit unsigned integer: %v", t.EmitterFilter.ChainId.Number()))
				}
				fi = append(fi, filterSignedVaa{
					chainId:     vaa.ChainID(t.EmitterFilter.ChainId), // #nosec G115 -- This is validated above
					emitterAddr: addr,
				})
			default:
				return status.Error(codes.InvalidArgument, "unsupported filter type")
			}
		}
	}

	s.subsSignedVaaMu.Lock()
	id := subscriptionId()
	sub := &subscriptionSignedVaa{
		ch:      make(chan message, 1),
		filters: fi,
	}
	s.subsSignedVaa[id] = sub
	s.subsSignedVaaMu.Unlock()

	defer func() {
		for {
			// The channel sender locks the subscription mutex before sending to the channel.
			// If the channel is full, then the sender will block and we'll never be able to lock the mutex (resulting in deadlock).
			// So we empty the channel before trying acquire the lock.
			_ = DoWithTimeout(func() error { <-sub.ch; return nil }, time.Millisecond)
			if s.subsSignedVaaMu.TryLock() {
				delete(s.subsSignedVaa, id)
				s.subsSignedVaaMu.Unlock()
				return
			}
		}
	}()

	for {
		select {
		case <-resp.Context().Done():
			return resp.Context().Err()
		case msg := <-sub.ch:
			if err := DoWithTimeout(func() error {
				return resp.Send(&spyv1.SubscribeSignedVAAResponse{VaaBytes: msg.vaaBytes})
			}, *sendTimeout); err != nil {
				return err
			}
		}
	}
}

func newSpyServer(logger *zap.Logger) *spyServer {
	return &spyServer{
		logger:        logger.Named("spyserver"),
		subsSignedVaa: make(map[string]*subscriptionSignedVaa),
	}
}

// DoWithTimeout runs f and returns its error. If the deadline d elapses first,
// it returns a grpc DeadlineExceeded error instead.
func DoWithTimeout(f func() error, d time.Duration) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- f()
		close(errChan)
	}()
	t := time.NewTimer(d)
	select {
	case <-t.C:
		return status.Errorf(codes.DeadlineExceeded, "too slow")
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	}
}

func spyServerRunnable(s *spyServer, logger *zap.Logger, listenAddr string) (supervisor.Runnable, *grpc.Server, error) {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}

	logger.Info("spy server listening", zap.String("addr", l.Addr().String()))

	grpcServer := common.NewInstrumentedGRPCServer(logger, common.GrpcLogDetailFull)
	spyv1.RegisterSpyRPCServiceServer(grpcServer, s)

	return supervisor.GRPCServer(grpcServer, l, false), grpcServer, nil
}

func runSpy(cmd *cobra.Command, args []string) {
	common.SetRestrictiveUmask()

	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("wormhole-spy").Desugar()

	ipfslog.SetAllLoggers(lvl)

	if *envStr != "" {
		// If they specify --env then use the defaults for the network parameters and don't allow them to override them.
		if *p2pNetworkID != "" || *p2pBootstrap != "" {
			logger.Fatal(`If "--env" is specified, "--network" and "--bootstrap" may not be specified`)
		}
		env, err := common.ParseEnvironment(*envStr)
		if err != nil || (env != common.MainNet && env != common.TestNet) {
			logger.Fatal(`Invalid value for "--env", should be "mainnet" or "testnet"`)
		}
		*p2pNetworkID = p2p.GetNetworkId(env)
		*p2pBootstrap, err = p2p.GetBootstrapPeers(env)
		if err != nil {
			logger.Fatal("failed to determine p2p bootstrap peers", zap.String("env", string(env)), zap.Error(err))
		}
	} else {
		// If they don't specify --env, then --network and --bootstrap are required.
		if *p2pNetworkID == "" {
			logger.Fatal(`If "--env" is not specified, "--network" must be specified`)
		}
		if *p2pBootstrap == "" {
			logger.Fatal(`If "--env" is not specified, "--bootstrap" must be specified`)
		}
	}

	// Status server
	if *statusAddr != "" {
		router := mux.NewRouter()

		router.Handle("/metrics", promhttp.Handler())

		go func() {
			logger.Info("status server listening on [::]:6060")
			logger.Error("status server crashed", zap.Error(http.ListenAndServe(*statusAddr, router))) // #nosec G114 local status server not vulnerable to DoS attack
		}()
	}

	// Verify flags

	if *nodeKeyPath == "" {
		logger.Fatal("Please specify --nodeKey")
	}
	if *p2pBootstrap == "" {
		logger.Fatal("Please specify --bootstrap")
	}

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Inbound signed VAAs
	signedInC := make(chan *gossipv1.SignedVAAWithQuorum, 1024)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState(nil)

	// RPC server
	s := newSpyServer(logger)
	rpcSvc, _, err := spyServerRunnable(s, logger, *spyRPC)
	if err != nil {
		logger.Fatal("failed to start RPC server", zap.Error(err))
	}

	// VAA verifier (optional)
	if *ethRPC != "" {
		if *ethContract == "" {
			logger.Fatal(`If "--ethRPC" is specified, "--ethContract" must also be specified`)
		}
		s.vaaVerifier = NewVaaVerifier(logger, *ethRPC, *ethContract)
		if err := s.vaaVerifier.GetInitialGuardianSet(); err != nil {
			logger.Fatal(`Failed to read initial guardian set for VAA verification`, zap.Error(err))
		}
	}

	// Log signed VAAs
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case v := <-signedInC:
				logger.Info("Received signed VAA",
					zap.Any("vaa", v.Vaa))
				if err := s.PublishSignedVAA(v.Vaa); err != nil {
					logger.Error("failed to publish signed VAA", zap.Error(err), zap.Any("vaa", v.Vaa))
				}
			}
		}
	}()

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		components := p2p.DefaultComponents()
		components.Port = *p2pPort
		params, err := p2p.NewRunParams(
			*p2pBootstrap,
			*p2pNetworkID,
			priv,
			gst,
			rootCtxCancel,
			p2p.WithSignedVAAListener(signedInC),
			p2p.WithComponents(components),
			p2p.WithProtectedPeers(protectedPeers),
		)
		if err != nil {
			return err
		}

		if err := supervisor.Run(ctx,
			"p2p",
			p2p.Run(params)); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "spyrpc", rpcSvc); err != nil {
			return err
		}

		logger.Info("Started internal services")

		<-ctx.Done()
		return nil
	},
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
	// TODO: wait for things to shut down gracefully
}
