package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/gwrelayer"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/ibc"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/gorilla/mux"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
		dependencies: []string{"accountant", "governor", "gateway-relayer"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			components := p2p.DefaultComponents()
			components.Port = port

			if g.env == common.GoTest {
				components.WarnChannelOverflow = true
				components.SignedHeartbeatLogLevel = zapcore.InfoLevel
			}

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
				(g.gatewayRelayer != nil),
			)

			return nil
		}}
}

// GuardianOptionNoAccountant disables the accountant. It is a shorthand for GuardianOptionAccountant("", "", false, nil)
// Dependencies: none
func GuardianOptionNoAccountant() *GuardianOption {
	return &GuardianOption{
		name: "accountant",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			logger.Info("acct: accountant is disabled", zap.String("component", "gacct"))
			return nil
		}}
}

// GuardianOptionAccountant configures the Accountant module.
// Dependencies: db
func GuardianOptionAccountant(contract string, websocket string, enforcing bool, wormchainConn *wormconn.ClientConn) *GuardianOption {
	return &GuardianOption{
		name:         "accountant",
		dependencies: []string{"db"},
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
			if wormchainConn == nil {
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
				wormchainConn,
				enforcing,
				g.gk,
				g.gst,
				g.acctC.writeC,
				g.env,
			)

			return nil
		}}
}

// GuardianOptionGovernor enables or disables the governor.
// Dependencies: db
func GuardianOptionGovernor(governorEnabled bool) *GuardianOption {
	return &GuardianOption{
		name:         "governor",
		dependencies: []string{"db"},
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

// GuardianOptionGatewayRelayer configures the Gateway Relayer module. If the gateway relayer smart contract is configured, we will instantiate
// the GatewayRelayer and signed VAAs will be passed to it for processing when they are published. It will forward payload three transfers destined
// for the specified contract on wormchain to that contract.
func GuardianOptionGatewayRelayer(gatewayRelayerContract string, wormchainConn *wormconn.ClientConn) *GuardianOption {
	return &GuardianOption{
		name: "gateway-relayer",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			g.gatewayRelayer = gwrelayer.NewGatewayRelayer(
				ctx,
				logger,
				gatewayRelayerContract,
				wormchainConn,
				g.env,
			)

			return nil
		}}
}

// GuardianOptionStatusServer configures the status server, including /readyz and /metrics.
// If g.env == common.UnsafeDevNet || g.env == common.GoTest, pprof will be enabled under /debug/pprof/
// Dependencies: none
func GuardianOptionStatusServer(statusAddr string) *GuardianOption {
	return &GuardianOption{
		name: "status-server",
		f: func(_ context.Context, _ *zap.Logger, g *G) error {
			if statusAddr != "" {
				// Use a custom routing instead of using http.DefaultServeMux directly to avoid accidentally exposing packages
				// that register themselves with it by default (like pprof).
				router := mux.NewRouter()

				// pprof server. NOT necessarily safe to expose publicly - only enable it in dev mode to avoid exposing it by
				// accident. There's benefit to having pprof enabled on production nodes, but we would likely want to expose it
				// via a dedicated port listening on localhost, or via the admin UNIX socket.
				if g.env == common.UnsafeDevNet || g.env == common.GoTest {
					// Pass requests to http.DefaultServeMux, which pprof automatically registers with as an import side-effect.
					router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
				}

				// Simple endpoint exposing node readiness (safe to expose to untrusted clients)
				router.HandleFunc("/readyz", readiness.Handler)

				// Prometheus metrics (safe to expose to untrusted clients)
				router.Handle("/metrics", promhttp.Handler())

				// SECURITY: If making changes, ensure that we always do `router := mux.NewRouter()` before this to avoid accidentally exposing pprof
				server := &http.Server{
					Addr:              statusAddr,
					Handler:           router,
					ReadHeaderTimeout: time.Second, // SECURITY defense against Slowloris Attack
					ReadTimeout:       time.Second,
					WriteTimeout:      time.Second,
				}

				g.runnables["status-server"] = func(ctx context.Context) error {
					logger := supervisor.Logger(ctx)
					go func() {
						if err := server.ListenAndServe(); err != http.ErrServerClosed {
							logger.Error("status server crashed", zap.Error(err))
						}
					}()
					logger.Info("status server listening", zap.String("status_addr", statusAddr))

					<-ctx.Done()
					if err := server.Shutdown(context.Background()); err != nil { // We use context.Background() instead of ctx here because ctx is already canceled at this point and Shutdown would not work then.
						logger := supervisor.Logger(ctx)
						logger.Error("error while shutting down status server: ", zap.Error(err))
					}
					return nil
				}
			}
			return nil
		}}
}

type IbcWatcherConfig struct {
	Websocket string
	Lcd       string
	Contract  string
}

// GuardianOptionWatchers configues all normal watchers and all IBC watchers. They need to be all configured at the same time because they may depend on each other.
// TODO: currently, IBC watchers are partially statically configured in ibc.ChainConfig. It might make sense to refactor this to instead provide this as a parameter here.
// Dependencies: none
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

			watchers := make(map[watchers.NetworkID]interfaces.L1Finalizer)

			for _, wc := range watcherConfigs {
				if _, ok := watchers[wc.GetNetworkID()]; ok {
					return fmt.Errorf("NetworkID already configured: %s", string(wc.GetNetworkID()))
				}

				watcherName := string(wc.GetNetworkID()) + "watch"
				logger.Debug("Setting up watcher: " + watcherName)

				if wc.GetNetworkID() != "solana-confirmed" { // TODO this should not be a special case, see comment in common/readiness.go
					common.MustRegisterReadinessSyncing(wc.GetChainID())
				}

				chainObsvReqC[wc.GetChainID()] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)

				if wc.RequiredL1Finalizer() != "" {
					l1watcher, ok := watchers[wc.RequiredL1Finalizer()]
					if !ok || l1watcher == nil {
						logger.Fatal("L1finalizer does not exist. Please check the order of the watcher configurations in watcherConfigs. The L1 must be configured before this one.",
							zap.String("ChainID", wc.GetChainID().String()),
							zap.String("L1ChainID", string(wc.RequiredL1Finalizer())))
					}
					wc.SetL1Finalizer(l1watcher)
				}

				l1finalizer, runnable, err := wc.Create(chainMsgC[wc.GetChainID()], chainObsvReqC[wc.GetChainID()], g.setC.writeC, g.env)

				if err != nil {
					return fmt.Errorf("error creating watcher: %w", err)
				}

				g.runnablesWithScissors[watcherName] = runnable
				watchers[wc.GetNetworkID()] = l1finalizer
			}

			if ibcWatcherConfig != nil {

				var chainConfig ibc.ChainConfig
				for _, chainID := range ibc.Chains {

					if _, exists := chainMsgC[chainID]; !exists {
						return fmt.Errorf("invalid IBC chain ID: %s", chainID.String())
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
					return errors.New("although IBC is enabled, there are no chains for it to monitor")
				}
			}

			go handleReobservationRequests(ctx, clock.New(), logger, g.obsvReqC.readC, chainObsvReqC)

			return nil
		}}
}

// GuardianOptionAdminService enables the admin rpc service on a unix socket.
// Dependencies: db, governor
func GuardianOptionAdminService(socketPath string, ethRpc *string, ethContract *string, rpcMap map[string]string) *GuardianOption {
	return &GuardianOption{
		name:         "admin-service",
		dependencies: []string{"governor", "db"},
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
				return fmt.Errorf("failed to create admin service: %w", err)
			}
			g.runnables["admin"] = adminService

			return nil
		}}
}

// GuardianOptionPublicRpcSocket enables the public rpc service on a unix socket
// Dependencies: db, governor
func GuardianOptionPublicRpcSocket(publicGRPCSocketPath string, publicRpcLogDetail common.GrpcLogDetail) *GuardianOption {
	return &GuardianOption{
		name:         "publicrpcsocket",
		dependencies: []string{"db", "governor"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			// local public grpc service socket
			publicrpcUnixService, publicrpcServer, err := publicrpcUnixServiceRunnable(logger, publicGRPCSocketPath, publicRpcLogDetail, g.db, g.gst, g.gov)
			if err != nil {
				return fmt.Errorf("failed to create publicrpc service: %w", err)
			}

			g.runnables["publicrpcsocket"] = publicrpcUnixService
			g.publicrpcServer = publicrpcServer
			return nil
		}}
}

// GuardianOptionPublicrpcTcpService enables the public gRPC service on TCP.
// Dependencies: db, governor, publicrpcsocket
func GuardianOptionPublicrpcTcpService(publicRpc string, publicRpcLogDetail common.GrpcLogDetail) *GuardianOption {
	return &GuardianOption{
		name:         "publicrpc",
		dependencies: []string{"db", "governor", "publicrpcsocket"},
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			publicrpcService := publicrpcTcpServiceRunnable(logger, publicRpc, publicRpcLogDetail, g.db, g.gst, g.gov)
			g.runnables["publicrpc"] = publicrpcService
			return nil
		}}
}

// GuardianOptionPublicWeb enables the public rpc service on http, i.e. gRPC-web and JSON-web.
// Dependencies: db, governor, publicrpcsocket
func GuardianOptionPublicWeb(listenAddr string, publicGRPCSocketPath string, tlsHostname string, tlsProdEnv bool, tlsCacheDir string) *GuardianOption {
	return &GuardianOption{
		name:         "publicweb",
		dependencies: []string{"db", "governor", "publicrpcsocket"},
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

// GuardianOptionDatabase configures the main database to be used for this guardian node.
// Dependencies: none
func GuardianOptionDatabase(db *db.Database) *GuardianOption {
	return &GuardianOption{
		name: "db",
		f: func(ctx context.Context, logger *zap.Logger, g *G) error {
			g.db = db
			return nil
		}}
}

// GuardianOptionProcessor enables the default processor, which is required to make consensus on messages.
// Dependencies: db, governor, accountant
func GuardianOptionProcessor() *GuardianOption {
	return &GuardianOption{
		name: "processor",
		// governor and accountant may be set to nil, but that choice needs to be made before the processor is configured
		dependencies: []string{"db", "governor", "accountant", "gateway-relayer"},

		f: func(ctx context.Context, logger *zap.Logger, g *G) error {

			g.runnables["processor"] = processor.NewProcessor(ctx,
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
				g.gatewayRelayer,
			).Run

			return nil
		}}
}
