package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	libp2p_crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	inboundObservationBufferSize         = 50
	inboundSignedVaaBufferSize           = 50
	observationRequestOutboundBufferSize = 50
	observationRequestInboundBufferSize  = 50
	// observationRequestBufferSize is the buffer size of the per-network reobservation channel
	observationRequestBufferSize = 25
)

type GuardianOption struct {
	name         string
	dependencies []string                                                // Array of `componentName` of other components that need to be configured before this component. Dependencies are enforced at runtime.
	f            func(context.Context, *zap.Logger, *GuardianNode) error // Function that is run by the constructor to initialize this component.
}

// GuardianOptionP2P configures p2p networking.
// Dependencies: Accountant, Governor
func GuardianOptionP2P(p2pKey libp2p_crypto.PrivKey, networkId string, bootstrapPeers string, nodeName string, disableHeartbeatVerify bool, port uint) GuardianOption {
	return GuardianOption{
		name:         "p2p",
		dependencies: []string{"accountant", "governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			components := p2p.DefaultComponents()
			components.Port = port

			g.runnables["p2p"] = p2p.Run(
				g.obsvC,
				g.obsvReqC.writeC,
				g.obsvReqSendC.readC,
				g.gossipSendC,
				g.signedInC.writeC,
				p2pKey,
				g.gk,
				g.gst,
				networkId,
				bootstrapPeers,
				nodeName,
				disableHeartbeatVerify,
				g.rootCtxCancel,
				g.acct,
				g.gov,
				nil,
				nil,
				components,
			)

			return nil
		}}
}

// GuardianOptionAccountant configures the Accountant module.
// Requires: wormchainConn
func GuardianOptionAccountant(contract string, websocket string, enforcing bool) GuardianOption {
	return GuardianOption{
		name: "accountant",
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			// Set up the accountant. If the accountant smart contract is configured, we will instantiate the accountant and VAAs
			// will be passed to it for processing. It will forward all token bridge transfers to the accountant contract.
			// If accountantCheckEnabled is set to true, token bridge transfers will not be signed and published until they
			// are approved by the accountant smart contract.
			if contract != "" {
				if websocket == "" {
					return errors.New("acct: if accountantContract is specified, accountantWS is required")
				}
				if g.wormchainConn == nil {
					return errors.New("acct: if accountantContract is specified, the wormchain sending connection must be enabled before.")
				}
				if enforcing {
					logger.Info("acct: accountant is enabled and will be enforced")
				} else {
					logger.Info("acct: accountant is enabled but will not be enforced")
				}

				g.acct = accountant.NewAccountant(
					g.ctx,
					logger,
					g.db,
					g.obsvReqC.writeC,
					contract,
					websocket,
					g.wormchainConn,
					enforcing,
					g.gk,
					g.gst,
					g.acctC.writeC,
					g.env,
				)
			} else {
				logger.Info("acct: accountant is disabled")
			}

			return nil
		}}
}

func GuardianOptionGovernor(governorEnabled bool) GuardianOption {
	return GuardianOption{
		name: "governor",
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			if governorEnabled {
				logger.Info("chain governor is enabled")
				g.gov = governor.NewChainGovernor(logger, g.db, g.env)
			} else {
				logger.Info("chain governor is disabled")
			}
			return nil
		}}
}

func GuardianOptionWatchers(watcherConfigs []watchers.WatcherConfig) GuardianOption {
	return GuardianOption{
		name: "watchers",
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {

			// Per-chain observation requests
			chainObsvReqC := make(map[vaa.ChainID]chan *gossipv1.ObservationRequest)

			// multiplex reobservation requests
			go handleReobservationRequests(g.ctx, clock.New(), logger, g.obsvReqC.readC, chainObsvReqC)

			// Per-chain msgC
			chainMsgC := make(map[vaa.ChainID]chan *common.MessagePublication)
			// aggregate per-chain msgC into msgC.
			// SECURITY defense-in-depth: This way we enforce that a watcher must set the msg.EmitterChain to its chainId, which makes the code easier to audit
			for _, chainId := range vaa.GetAllNetworkIDs() {
				chainMsgC[chainId] = make(chan *common.MessagePublication)
				go func(c <-chan *common.MessagePublication, chainId vaa.ChainID) {
					for {
						select {
						case <-g.ctx.Done():
							return
						case msg := <-c:
							if msg.EmitterChain == chainId {
								g.msgC.writeC <- msg
							} else {
								// SECURITY: This should never happen. If it does, a watcher has been compromised.
								logger.Fatal("SECURITY CRITICAL: Received observation from a chain that was not marked as originating from that chain",
									zap.Stringer("tx", msg.TxHash),
									zap.Stringer("emitter_address", msg.EmitterAddress),
									zap.Uint64("sequence", msg.Sequence),
									zap.Stringer("msgChainId", msg.EmitterChain),
									zap.Stringer("watcherChainId", chainId),
								)
							}
						}
					}
				}(chainMsgC[chainId], chainId)
			}

			watchers := make(map[watchers.NetworkID]interfaces.L1Finalizer)

			for _, w := range watcherConfigs {
				watcherName := string(w.GetNetworkID()) + "watch"
				logger.Info("Starting watcher: " + watcherName)
				common.MustRegisterReadinessSyncing(w.GetChainID())
				chainObsvReqC[w.GetChainID()] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)

				l1finalizer, runnable, err := w.Create(chainMsgC[w.GetChainID()], chainObsvReqC[w.GetChainID()], g.setC.writeC, g.env)

				if err != nil {
					logger.Fatal("error creating watcher", zap.Error(err))
				}

				if w.RequiredL1Finalizer() != "" {
					l1watcher, ok := watchers[w.RequiredL1Finalizer()]
					if !ok || l1watcher == nil {
						logger.Fatal("L1finalizer does not exist. Please check the order of the watcher configuration. The L1 must be configured before this one.",
							zap.String("ChainID", w.GetChainID().String()),
							zap.String("L1ChainID", string(w.RequiredL1Finalizer())))
					}
					w.SetL1Finalizer(l1watcher)
				}

				g.runnablesWithScissors[watcherName] = runnable
				watchers[w.GetNetworkID()] = l1finalizer
			}

			return nil
		}}
}

func GuardianOptionAdminService(socketPath string, ethRpc *string, ethContract *string) GuardianOption {
	return GuardianOption{
		name:         "admin-service",
		dependencies: []string{"governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			adminService, err := adminServiceRunnable(
				logger,
				socketPath,
				g.injectC.writeC,
				g.signedInC.writeC,
				g.obsvReqSendC.writeC,
				g.db,
				g.gst,
				g.gov,
				g.gk,
				ethRpc,
				ethContract,
			)
			if err != nil {
				logger.Fatal("failed to create admin service socket", zap.Error(err))
			}
			g.runnables["admin"] = adminService

			return nil
		}}
}

func GuardianOptionPublicRpcSocket(publicGRPCSocketPath string, publicRpcLogDetail common.GrpcLogDetail) GuardianOption {
	return GuardianOption{
		name:         "publicrpcsocket",
		dependencies: []string{"governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			// local public grpc service socket
			publicrpcUnixService, publicrpcServer, err := publicrpcUnixServiceRunnable(logger, publicGRPCSocketPath, publicRpcLogDetail, g.db, g.gst, g.gov)
			if err != nil {
				logger.Fatal("failed to create publicrpc service socket", zap.Error(err))
			}

			g.runnables["publicrpcsocket"] = publicrpcUnixService
			g.publicrpcServer = publicrpcServer
			return nil
		}}
}

func GuardianOptionPublicrpcTcpService(publicRpc string, publicRpcLogDetail common.GrpcLogDetail) GuardianOption {
	return GuardianOption{
		name:         "publicrpc",
		dependencies: []string{"governor", "publicrpcsocket"},
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			publicrpcService, err := publicrpcTcpServiceRunnable(logger, publicRpc, publicRpcLogDetail, g.db, g.gst, g.gov)
			if err != nil {
				return err
			}
			g.runnables["publicrpc"] = publicrpcService
			return nil
		}}
}

func GuardianOptionPublicWeb(listenAddr string, publicGRPCSocketPath string, tlsHostname string, tlsProdEnv bool, tlsCacheDir string) GuardianOption {
	return GuardianOption{
		name:         "publicweb",
		dependencies: []string{"publicrpcsocket"},
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			publicwebService, err := publicwebServiceRunnable(logger, listenAddr, publicGRPCSocketPath, g.publicrpcServer,
				tlsHostname, tlsProdEnv, tlsCacheDir)
			if err != nil {
				log.Fatal("failed to create publicrpc web service", zap.Error(err))
			}
			g.runnables["publicweb"] = publicwebService
			return nil
		}}
}

func GuardianOptionBigTablePersistence(config *reporter.BigTableConnectionConfig) GuardianOption {
	return GuardianOption{
		name: "bigtable",
		f: func(ctx context.Context, logger *zap.Logger, g *GuardianNode) error {
			g.runnables["bigtable"] = reporter.BigTableWriter(g.attestationEvents, config)
			return nil
		}}
}

type GuardianNode struct {
	ctx           context.Context
	rootCtxCancel context.CancelFunc
	env           common.Environment

	// keys
	gk *ecdsa.PrivateKey

	// components
	db                *db.Database
	gst               *common.GuardianSetState
	acct              *accountant.Accountant
	gov               *governor.ChainGovernor
	attestationEvents *reporter.AttestationEventReporter
	wormchainConn     *wormconn.ClientConn
	publicrpcServer   *grpc.Server

	// runnables
	runnablesWithScissors map[string]supervisor.Runnable
	runnables             map[string]supervisor.Runnable

	// various channels
	// Outbound gossip message queue (needs to be read/write because p2p needs read/write)
	gossipSendC chan []byte
	// Inbound observations (TODO: why is this a read/write channel instead of channelPair?)
	obsvC chan *gossipv1.SignedObservation
	// Finalized guardian observations aggregated across all chains
	msgC channelPair[*common.MessagePublication]
	// Ethereum incoming guardian set updates
	setC channelPair[*common.GuardianSet]
	// Inbound signed VAAs
	signedInC channelPair[*gossipv1.SignedVAAWithQuorum]
	// Inbound observation requests from the p2p service (for all chains)
	obsvReqC channelPair[*gossipv1.ObservationRequest]
	// Outbound observation requests
	obsvReqSendC channelPair[*gossipv1.ObservationRequest]
	// Injected VAAs (manually generated rather than created via observation)
	injectC channelPair[*vaa.VAA]
	// TODO describe
	acctC channelPair[*common.MessagePublication]
}

func NewGuardianNode(
	ctx context.Context,
	rootCtxCancel context.CancelFunc,
	env common.Environment,
	db *db.Database,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
) *GuardianNode {
	g := GuardianNode{
		ctx:           ctx,
		rootCtxCancel: rootCtxCancel,
		env:           env,
		db:            db,
		gk:            gk,
		wormchainConn: wormchainConn,
	}
	return &g
}

// initializeBasic sets up everything that every GuardianNode needs before any options can be applied.
func (g *GuardianNode) initializeBasic(logger *zap.Logger) {
	// Setup various channels...
	g.gossipSendC = make(chan []byte)
	g.obsvC = make(chan *gossipv1.SignedObservation, inboundObservationBufferSize)
	g.msgC = makeChannelPair[*common.MessagePublication](0)
	g.setC = makeChannelPair[*common.GuardianSet](0)
	g.signedInC = makeChannelPair[*gossipv1.SignedVAAWithQuorum](inboundSignedVaaBufferSize)
	g.obsvReqC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestOutboundBufferSize)
	g.obsvReqSendC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestInboundBufferSize)
	g.injectC = makeChannelPair[*vaa.VAA](0)
	g.acctC = makeChannelPair[*common.MessagePublication](0)

	// Guardian set state managed by processor
	g.gst = common.NewGuardianSetState(nil)

	// provides methods for reporting progress toward message attestation, and channels for receiving attestation lifecyclye events.
	g.attestationEvents = reporter.EventListener(logger)

	// allocate maps
	g.runnablesWithScissors = make(map[string]supervisor.Runnable)
	g.runnables = make(map[string]supervisor.Runnable)
}

// applyOptions applies `options` to the GuardianNode.
// Each option must have a unique option.name.
// If an option has `dependencies`, they must be defined before that option.
func (g *GuardianNode) applyOptions(ctx context.Context, logger *zap.Logger, options []GuardianOption) error {
	configuredComponents := make(map[string]struct{}) // using `map[string]struct{}` to implement a set here

	for _, option := range options {
		// check that this component has not been configured yet
		if _, ok := configuredComponents[option.name]; ok {
			return errors.New(fmt.Sprintf("Component %s has been configured already and cannot be configured a second time.", option.name))
		}

		// check that all dependencies have been met
		for _, dep := range option.dependencies {
			if _, ok := configuredComponents[dep]; !ok {
				return errors.New(fmt.Sprintf("Component %s requires %s to be configured first. Check the order of your options.", option.name, dep))
			}
		}

		// run the config
		option.f(ctx, logger, g)

		// mark the component as configured
		configuredComponents[option.name] = struct{}{}
	}

	return nil
}

func (g *GuardianNode) Run(options ...GuardianOption) supervisor.Runnable {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)

		g.initializeBasic(logger)
		if err := g.applyOptions(ctx, logger, options); err != nil {
			logger.Fatal("failed to initialize GuardianNode", zap.Error(err))
		}

		// Start the watchers
		for watcherName, runnable := range g.runnablesWithScissors {
			if err := supervisor.Run(ctx, watcherName, common.WrapWithScissors(runnable, watcherName)); err != nil {
				return err
			}
		}

		if g.acct != nil {
			if err := g.acct.Start(ctx); err != nil {
				logger.Fatal("acct: failed to start accountant", zap.Error(err))
			}
		}

		if g.gov != nil {
			err := g.gov.Run(ctx)
			if err != nil {
				logger.Fatal("failed to create chain governor", zap.Error(err))
			}
		}

		if err := supervisor.Run(ctx, "processor", processor.NewProcessor(ctx,
			g.db,
			g.msgC.readC,
			g.setC.readC,
			g.gossipSendC,
			g.obsvC,
			g.obsvReqSendC.writeC,
			g.injectC.readC,
			g.signedInC.readC,
			g.gk,
			g.gst,
			g.attestationEvents,
			g.gov,
			g.acct,
			g.acctC.readC,
		).Run); err != nil {
			return err
		}

		// Start any other runnables
		for name, runnable := range g.runnables {
			if err := supervisor.Run(ctx, name, runnable); err != nil {
				return err
			}
		}

		logger.Info("Started internal services")

		<-ctx.Done()

		return nil
	}
}

type channelPair[T any] struct {
	readC  <-chan T
	writeC chan<- T
}

func makeChannelPairDEPRECATED[T any](cap int) (<-chan T, chan<- T) {
	out := make(chan T, cap)
	return out, out
}

func makeChannelPair[T any](cap int) channelPair[T] {
	out := make(chan T, cap)
	return channelPair[T]{out, out}
}
