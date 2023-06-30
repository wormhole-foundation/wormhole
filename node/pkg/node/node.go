package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/http"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/ibc"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/gorilla/mux"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

const (
	inboundObservationBufferSize         = 5000
	inboundSignedVaaBufferSize           = 50
	observationRequestOutboundBufferSize = 50
	observationRequestInboundBufferSize  = 50
	// observationRequestBufferSize is the buffer size of the per-network reobservation channel
	observationRequestBufferSize = 25
)

type PrometheusCtxKey struct{}

type GuardianOption struct {
	name         string
	dependencies []string                                     // Array of other option's `name`. These options need to be configured before this option. Dependencies are enforced at runtime.
	f            func(context.Context, *zap.Logger, *G) error // Function that is run by the constructor to initialize this component.
}

// GuardianOptionP2P configures p2p networking.
// Dependencies: Accountant, Governor
func GuardianOptionP2P(p2pKey libp2p_crypto.PrivKey, networkId string, bootstrapPeers string, nodeName string, disableHeartbeatVerify bool, port uint, ibcFeaturesFunc func() string) *GuardianOption {
	return &GuardianOption{
		name:         "p2p",
		dependencies: []string{"accountant", "governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
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
				ibcFeaturesFunc,
				(g.queryHandler != nil),
				g.signedQueryReqC.writeC,
				g.queryResponsePublicationC.readC,
			)

			return nil
		}}
}

// GuardianOptionAccountant configures the Accountant module.
// Requires: wormchainConn
func GuardianOptionAccountant(contract string, websocket string, enforcing bool) *GuardianOption {
	return &GuardianOption{
		name: "accountant",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			// Set up the accountant. If the accountant smart contract is configured, we will instantiate the accountant and VAAs
			// will be passed to it for processing. It will forward all token bridge transfers to the accountant contract.
			// If accountantCheckEnabled is set to true, token bridge transfers will not be signed and published until they
			// are approved by the accountant smart contract.
			if contract == "" {
				logger.Info("acct: accountant is disabled", zap.String("component", "gacct"))
				return nil
			}

			if websocket == "" {
				return errors.New("acct: if accountantContract is specified, accountantWS is required")
			}
			if g.wormchainConn == nil {
				return errors.New("acct: if accountantContract is specified, the wormchain sending connection must be enabled before.")
			}
			if enforcing {
				logger.Info("acct: accountant is enabled and will be enforced", zap.String("component", "gacct"))
			} else {
				logger.Info("acct: accountant is enabled but will not be enforced", zap.String("component", "gacct"))
			}

			g.acct = accountant.NewAccountant(
				ctx,
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

			return nil
		}}
}

func GuardianOptionGovernor(governorEnabled bool) *GuardianOption {
	return &GuardianOption{
		name: "governor",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			if governorEnabled {
				logger.Info("chain governor is enabled")
				g.gov = governor.NewChainGovernor(logger, g.db, g.env)
			} else {
				logger.Info("chain governor is disabled")
			}
			return nil
		}}
}

// GuardianOptionQueryHandler configures the Cross Chain Query module.
func GuardianOptionQueryHandler(ccqEnabled bool, allowedRequesters string) *GuardianOption {
	return &GuardianOption{
		name: "query",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			if !ccqEnabled {
				logger.Info("ccq: cross chain query is disabled", zap.String("component", "ccq"))
				return nil
			}

			g.queryHandler = query.NewQueryHandler(
				logger,
				g.env,
				allowedRequesters,
				g.signedQueryReqC.readC,
				g.chainQueryReqC,
				g.queryResponseC.readC,
				g.queryResponsePublicationC.writeC,
			)

			return nil
		}}
}

func GuardianOptionStatusServer(statusAddr string) *GuardianOption {
	return &GuardianOption{
		name: "status-server",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			if statusAddr != "" {
				// Use a custom routing instead of using http.DefaultServeMux directly to avoid accidentally exposing packages
				// that register themselves with it by default (like pprof).
				router := mux.NewRouter()

				// pprof server. NOT necessarily safe to expose publicly - only enable it in dev mode to avoid exposing it by
				// accident. There's benefit to having pprof enabled on production nodes, but we would likely want to expose it
				// via a dedicated port listening on localhost, or via the admin UNIX socket.
				if g.env == common.UnsafeDevNet {
					// Pass requests to http.DefaultServeMux, which pprof automatically registers with as an import side-effect.
					router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
				}

				// Simple endpoint exposing node readiness (safe to expose to untrusted clients)
				router.HandleFunc("/readyz", readiness.Handler)

				// Prometheus metrics (safe to expose to untrusted clients)
				router.Handle("/metrics", promhttp.Handler())

				go func() {
					logger.Info("status server listening", zap.String("status_addr", statusAddr))
					// SECURITY: If making changes, ensure that we always do `router := mux.NewRouter()` before this to avoid accidentally exposing pprof
					logger.Error("status server crashed", zap.Error(http.ListenAndServe(statusAddr, router))) // #nosec G114 local status server not vulnerable to DoS attack
				}()
			}
			return nil
		}}
}

type IbcWatcherConfig struct {
	Websocket string
	Lcd       string
	Contract  string
}

func GuardianOptionWatchers(watcherConfigs []watchers.WatcherConfig, ibcWatcherConfig *IbcWatcherConfig) *GuardianOption {
	return &GuardianOption{
		name: "watchers",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {

			chainObsvReqC := make(map[vaa.ChainID]chan *gossipv1.ObservationRequest)

			chainMsgC := make(map[vaa.ChainID]chan *common.MessagePublication)

			for _, chainId := range vaa.GetAllNetworkIDs() {
				chainMsgC[chainId] = make(chan *common.MessagePublication)
				go func(c <-chan *common.MessagePublication, chainId vaa.ChainID) {
					zeroAddress := vaa.Address{}
					for {
						select {
						case <-ctx.Done():
							return
						case msg := <-c:
							if msg.EmitterChain != chainId {
								level := zapcore.FatalLevel
								if g.env == common.GoTest {
									// If we're in gotest, we don't want to os.Exit() here because that's hard to catch.
									// Since continuing execution here doesn't have any side effects here, it's fine to have a
									// differing behavior in GoTest mode.
									level = zapcore.ErrorLevel
								}
								logger.Log(level, "SECURITY CRITICAL: Received observation from a chain that was not marked as originating from that chain",
									zap.Stringer("tx", msg.TxHash),
									zap.Stringer("emitter_address", msg.EmitterAddress),
									zap.Uint64("sequence", msg.Sequence),
									zap.Stringer("msgChainId", msg.EmitterChain),
									zap.Stringer("watcherChainId", chainId),
								)
							} else if msg.EmitterAddress == zeroAddress {
								level := zapcore.FatalLevel
								if g.env == common.GoTest {
									// If we're in gotest, we don't want to os.Exit() here because that's hard to catch.
									// Since continuing execution here doesn't have any side effects here, it's fine to have a
									// differing behavior in GoTest mode.
									level = zapcore.ErrorLevel
								}
								logger.Log(level, "SECURITY ERROR: Received observation with EmitterAddress == 0x00",
									zap.Stringer("tx", msg.TxHash),
									zap.Stringer("emitter_address", msg.EmitterAddress),
									zap.Uint64("sequence", msg.Sequence),
									zap.Stringer("msgChainId", msg.EmitterChain),
									zap.Stringer("watcherChainId", chainId),
								)
							} else {
								g.msgC.writeC <- msg
							}
						}
					}
				}(chainMsgC[chainId], chainId)
			}

			// Per-chain query response channel
			chainQueryResponseC := make(map[vaa.ChainID]chan *query.PerChainQueryResponseInternal)
			// aggregate per-chain msgC into msgC.
			// SECURITY defense-in-depth: This way we enforce that a watcher must set the msg.EmitterChain to its chainId, which makes the code easier to audit
			for _, chainId := range vaa.GetAllNetworkIDs() {
				chainQueryResponseC[chainId] = make(chan *query.PerChainQueryResponseInternal)
				go func(c <-chan *query.PerChainQueryResponseInternal, chainId vaa.ChainID) {
					for {
						select {
						case <-ctx.Done():
							return
						case response := <-c:
							if response.ChainId != chainId {
								// SECURITY: This should never happen. If it does, a watcher has been compromised.
								logger.Fatal("SECURITY CRITICAL: Received query response from a chain that was not marked as originating from that chain",
									zap.Uint16("responseChainId", uint16(response.ChainId)),
									zap.Stringer("watcherChainId", chainId),
								)
							} else {
								g.queryResponseC.writeC <- response
							}
						}
					}
				}(chainQueryResponseC[chainId], chainId)
			}

			watchers := make(map[watchers.NetworkID]interfaces.L1Finalizer)

			for _, wc := range watcherConfigs {
				if _, ok := watchers[wc.GetNetworkID()]; ok {
					logger.Fatal("NetworkID already configured", zap.String("network_id", string(wc.GetNetworkID())))
				}

				watcherName := string(wc.GetNetworkID()) + "watch"
				logger.Debug("Setting up watcher: " + watcherName)

				if wc.GetNetworkID() != "solana-confirmed" { // TODO this should not be a special case, see comment in common/readiness.go
					common.MustRegisterReadinessSyncing(wc.GetChainID())
				}

				chainObsvReqC[wc.GetChainID()] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
				g.chainQueryReqC[wc.GetChainID()] = make(chan *query.PerChainQueryInternal, query.QueryRequestBufferSize)

				if wc.RequiredL1Finalizer() != "" {
					l1watcher, ok := watchers[wc.RequiredL1Finalizer()]
					if !ok || l1watcher == nil {
						logger.Fatal("L1finalizer does not exist. Please check the order of the watcher configurations in watcherConfigs. The L1 must be configured before this one.",
							zap.String("ChainID", wc.GetChainID().String()),
							zap.String("L1ChainID", string(wc.RequiredL1Finalizer())))
					}
					wc.SetL1Finalizer(l1watcher)
				}

				l1finalizer, runnable, err := wc.Create(chainMsgC[wc.GetChainID()], chainObsvReqC[wc.GetChainID()], g.chainQueryReqC[wc.GetChainID()], chainQueryResponseC[wc.GetChainID()], g.setC.writeC, g.env)

				if err != nil {
					logger.Fatal("error creating watcher", zap.Error(err))
				}

				g.runnablesWithScissors[watcherName] = runnable
				watchers[wc.GetNetworkID()] = l1finalizer
			}

			if ibcWatcherConfig != nil {

				var chainConfig ibc.ChainConfig
				for _, chainID := range ibc.Chains {

					if _, exists := chainMsgC[chainID]; !exists {
						return errors.New("invalid IBC chain ID")
					}

					if _, exists := chainObsvReqC[chainID]; exists {
						logger.Warn("not monitoring chain with IBC because it is already registered.", zap.Stringer("chainID", chainID))
						continue
					}

					chainObsvReqC[chainID] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
					common.MustRegisterReadinessSyncing(chainID)

					chainConfig = append(chainConfig, ibc.ChainConfigEntry{
						ChainID:  chainID,
						MsgC:     chainMsgC[chainID],
						ObsvReqC: chainObsvReqC[chainID],
					})
				}

				if len(chainConfig) > 0 {
					logger.Info("Starting IBC watcher")
					readiness.RegisterComponent(common.ReadinessIBCSyncing)
					g.runnablesWithScissors["ibcwatch"] = ibc.NewWatcher(ibcWatcherConfig.Websocket, ibcWatcherConfig.Lcd, ibcWatcherConfig.Contract, chainConfig).Run
				} else {
					logger.Error("Although IBC is enabled, there are no chains for it to monitor")
				}
			}

			go handleReobservationRequests(ctx, clock.New(), logger, g.obsvReqC.readC, chainObsvReqC)

			return nil
		}}
}

func GuardianOptionAdminService(socketPath string, ethRpc *string, ethContract *string, rpcMap map[string]string) *GuardianOption {
	return &GuardianOption{
		name:         "admin-service",
		dependencies: []string{"governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
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
				rpcMap,
			)
			if err != nil {
				logger.Fatal("failed to create admin service socket", zap.Error(err))
			}
			g.runnables["admin"] = adminService

			return nil
		}}
}

func GuardianOptionPublicRpcSocket(publicGRPCSocketPath string, publicRpcLogDetail common.GrpcLogDetail) *GuardianOption {
	return &GuardianOption{
		name:         "publicrpcsocket",
		dependencies: []string{"governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
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

func GuardianOptionPublicrpcTcpService(publicRpc string, publicRpcLogDetail common.GrpcLogDetail) *GuardianOption {
	return &GuardianOption{
		name:         "publicrpc",
		dependencies: []string{"governor", "publicrpcsocket"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			publicrpcService, err := publicrpcTcpServiceRunnable(logger, publicRpc, publicRpcLogDetail, g.db, g.gst, g.gov)
			if err != nil {
				return err
			}
			g.runnables["publicrpc"] = publicrpcService
			return nil
		}}
}

func GuardianOptionPublicWeb(listenAddr string, publicGRPCSocketPath string, tlsHostname string, tlsProdEnv bool, tlsCacheDir string) *GuardianOption {
	return &GuardianOption{
		name:         "publicweb",
		dependencies: []string{"publicrpcsocket"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			publicwebService := publicwebServiceRunnable(logger, listenAddr, publicGRPCSocketPath, g.publicrpcServer,
				tlsHostname, tlsProdEnv, tlsCacheDir)
			g.runnables["publicweb"] = publicwebService
			return nil
		}}
}

func GuardianOptionBigTablePersistence(config *reporter.BigTableConnectionConfig) *GuardianOption {
	return &GuardianOption{
		name: "bigtable",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			g.runnables["bigtable"] = reporter.BigTableWriter(g.attestationEvents, config)
			return nil
		}}
}

type G struct {
	// rootCtxCancel is a context.CancelFunc. It MUST be a root context for any context that is passed to any member function of G.
	// It can be used by components to shut down the entire node if they encounter an unrecoverable state.
	rootCtxCancel context.CancelFunc
	env           common.Environment

	// keys
	gk *ecdsa.PrivateKey

	// components
	db                *db.Database
	gst               *common.GuardianSetState
	acct              *accountant.Accountant
	gov               *governor.ChainGovernor
	queryHandler      *query.QueryHandler
	attestationEvents *reporter.AttestationEventReporter
	publicrpcServer   *grpc.Server

	// runnables
	runnablesWithScissors map[string]supervisor.Runnable
	runnables             map[string]supervisor.Runnable

	// various channels
	// Outbound gossip message queue (needs to be read/write because p2p needs read/write)
	gossipSendC chan []byte
	// Inbound observations. This is read/write because the processor also writes to it as a fast-path when handling locally made observations.
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
	// acctC is the channel where messages will be put after they reached quorum in the accountant.
	acctC channelPair[*common.MessagePublication]

	// Cross Chain Query Handler channels
	chainQueryReqC            map[vaa.ChainID]chan *query.PerChainQueryInternal
	signedQueryReqC           channelPair[*gossipv1.SignedQueryRequest]
	queryResponseC            channelPair[*query.PerChainQueryResponseInternal]
	queryResponsePublicationC channelPair[*query.QueryResponsePublication]
}

func NewGuardianNode(
	env common.Environment,
	gk *ecdsa.PrivateKey,
) *G {
	g := G{
		env:           env,
		db:            db,
		gk:            gk,
		wormchainConn: wormchainConn,

		// Cross Chain Query Handler channels
		chainQueryReqC:            make(map[vaa.ChainID]chan *query.PerChainQueryInternal),
		signedQueryReqC:           makeChannelPair[*gossipv1.SignedQueryRequest](query.SignedQueryRequestChannelSize),
		queryResponseC:            makeChannelPair[*query.PerChainQueryResponseInternal](0),
		queryResponsePublicationC: makeChannelPair[*query.QueryResponsePublication](0),
	}
	return &g
}

// initializeBasic sets up everything that every GuardianNode needs before any options can be applied.
func (g *G) initializeBasic(logger *zap.Logger, rootCtxCancel context.CancelFunc) {
	g.rootCtxCancel = rootCtxCancel

	// Setup various channels...
	g.gossipSendC = make(chan []byte)
	g.obsvC = make(chan *gossipv1.SignedObservation, inboundObservationBufferSize)
	g.msgC = makeChannelPair[*common.MessagePublication](0)
	g.setC = makeChannelPair[*common.GuardianSet](1) // This needs to be a buffered channel because of a circular dependency between processor and accountant during startup.
	g.signedInC = makeChannelPair[*gossipv1.SignedVAAWithQuorum](inboundSignedVaaBufferSize)
	g.obsvReqC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestOutboundBufferSize)
	g.obsvReqSendC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestInboundBufferSize)
	g.injectC = makeChannelPair[*vaa.VAA](0)
	g.acctC = makeChannelPair[*common.MessagePublication](accountant.MsgChannelCapacity)

	// Guardian set state managed by processor
	g.gst = common.NewGuardianSetState(nil)

	// provides methods for reporting progress toward message attestation, and channels for receiving attestation lifecycle events.
	g.attestationEvents = reporter.EventListener(logger)

	// allocate maps
	g.runnablesWithScissors = make(map[string]supervisor.Runnable)
	g.runnables = make(map[string]supervisor.Runnable)
}

// applyOptions applies `options` to the GuardianNode.
// Each option must have a unique option.name.
// If an option has `dependencies`, they must be defined before that option.
func (g *G) applyOptions(ctx context.Context, logger *zap.Logger, options []*GuardianOption) error {
	configuredComponents := make(map[string]struct{}) // using `map[string]struct{}` to implement a set here

	for _, option := range options {
		// check that this component has not been configured yet
		if _, ok := configuredComponents[option.name]; ok {
			return fmt.Errorf("Component %s is already configured and cannot be configured a second time.", option.name)
		}

		// check that all dependencies have been met
		for _, dep := range option.dependencies {
			if _, ok := configuredComponents[dep]; !ok {
				return fmt.Errorf("Component %s requires %s to be configured first. Check the order of your options.", option.name, dep)
			}
		}

		// run the config
		err := option.f(ctx, logger, g)
		if err != nil {
			return fmt.Errorf("Error applying option for component %s: %w", option.name, err)
		}

		// mark the component as configured
		configuredComponents[option.name] = struct{}{}
	}

	return nil
}

func (g *G) Run(rootCtxCancel context.CancelFunc, options ...*GuardianOption) supervisor.Runnable {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)

		g.initializeBasic(logger, rootCtxCancel)
		if err := g.applyOptions(ctx, logger, options); err != nil {
			logger.Fatal("failed to initialize GuardianNode", zap.Error(err))
		}
		logger.Info("GuardianNode initialization done.") // Do not modify this message, node_test.go relies on it.

		// Start the watchers
		for runnableName, runnable := range g.runnablesWithScissors {
			logger.Info("Starting runnablesWithScissors: " + runnableName)
			if err := supervisor.Run(ctx, runnableName, common.WrapWithScissors(runnable, runnableName)); err != nil {
				logger.Fatal("error starting runnablesWithScissors", zap.Error(err))
			}
		}

		// TODO there is an opportunity to refactor the startup of the accountant and governor:
		// Ideally they should just register a g.runnables["governor"] and g.runnables["accountant"] instead of being treated as special cases.
		if g.acct != nil {
			logger.Info("Starting accountant")
			if err := g.acct.Start(ctx); err != nil {
				logger.Fatal("acct: failed to start accountant", zap.Error(err))
			}
			defer g.acct.Close()
		}

		if g.gov != nil {
			logger.Info("Starting governor")
			if err := g.gov.Run(ctx); err != nil {
				logger.Fatal("failed to create chain governor", zap.Error(err))
			}
		}

		if g.queryHandler != nil {
			logger.Info("Starting query handler", zap.String("component", "ccq"))
			if err := g.queryHandler.Start(ctx); err != nil {
				logger.Fatal("failed to create chain governor", zap.Error(err), zap.String("component", "ccq"))
			}
		}

		logger.Info("Starting processor")
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
			logger.Fatal("failed to start processor", zap.Error(err))
		}

		// Start any other runnables
		for name, runnable := range g.runnables {
			if err := supervisor.Run(ctx, name, runnable); err != nil {
				logger.Fatal("failed to start other runnable", zap.Error(err))
			}
		}

		logger.Info("Started internal services")
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		<-ctx.Done()

		return nil
	}
}

type channelPair[T any] struct {
	readC  <-chan T
	writeC chan<- T
}

func makeChannelPair[T any](cap int) channelPair[T] {
	out := make(chan T, cap)
	return channelPair[T]{out, out}
}
