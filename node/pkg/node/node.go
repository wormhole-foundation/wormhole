package node

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/gwrelayer"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	// gossipSendBufferSize configures the size of the gossip network send buffer
	gossipSendBufferSize = 5000

	// inboundObservationBufferSize configures the size of the obsvC channel that contains observations from other Guardians.
	// One observation takes roughly 0.1ms to process on one core, so the whole queue could be processed in 1s
	inboundObservationBufferSize = 10000

	// inboundSignedVaaBufferSize configures the size of the signedInC channel that contains VAAs from other Guardians.
	// One VAA takes roughly 0.01ms to process if we already have one in the database and 2ms if we don't.
	// So in the worst case the entire queue can be processed in 2s.
	inboundSignedVaaBufferSize = 1000

	// observationRequestInboundBufferSize configures the size of obsvReqC.
	// Messages from there are immediately sent to the per-chain observation request channels, which are more important to configure.
	observationRequestInboundBufferSize = 500

	// observationRequestOutboundBufferSize configures the size of obsvReqSendC
	// and thereby somewhat limits the amout of observation requests that can be sent in bursts to the network.
	observationRequestOutboundBufferSize = 100

	// observationRequestPerChainBufferSize is the buffer size of the per-network reobservation channel
	observationRequestPerChainBufferSize = 100
)

type PrometheusCtxKey struct{}

type G struct {
	// rootCtxCancel is a context.CancelFunc. It MUST be a root context for any context that is passed to any member function of G.
	// It can be used by components to shut down the entire node if they encounter an unrecoverable state.
	rootCtxCancel context.CancelFunc
	env           common.Environment

	// keys
	gk *ecdsa.PrivateKey

	// components
	db              *db.Database
	gst             *common.GuardianSetState
	acct            *accountant.Accountant
	gov             *governor.ChainGovernor
	gatewayRelayer  *gwrelayer.GatewayRelayer
	publicrpcServer *grpc.Server

	// runnables
	runnablesWithScissors map[string]supervisor.Runnable
	runnables             map[string]supervisor.Runnable

	// various channels
	// Outbound gossip message queue (needs to be read/write because p2p needs read/write)
	gossipSendC chan []byte
	// Inbound observations. This is read/write because the processor also writes to it as a fast-path when handling locally made observations.
	obsvC chan *common.MsgWithTimeStamp[gossipv1.SignedObservation]
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
	// acctC is the channel where messages will be put after they reached quorum in the accountant.
	acctC channelPair[*common.MessagePublication]
}

func NewGuardianNode(
	env common.Environment,
	gk *ecdsa.PrivateKey,
) *G {
	g := G{
		env: env,
		gk:  gk,
	}
	return &g
}

// initializeBasic sets up everything that every GuardianNode needs before any options can be applied.
func (g *G) initializeBasic(rootCtxCancel context.CancelFunc) {
	g.rootCtxCancel = rootCtxCancel

	// Setup various channels...
	g.gossipSendC = make(chan []byte, gossipSendBufferSize)
	g.obsvC = make(chan *common.MsgWithTimeStamp[gossipv1.SignedObservation], inboundObservationBufferSize)
	g.msgC = makeChannelPair[*common.MessagePublication](0)
	g.setC = makeChannelPair[*common.GuardianSet](1) // This needs to be a buffered channel because of a circular dependency between processor and accountant during startup.
	g.signedInC = makeChannelPair[*gossipv1.SignedVAAWithQuorum](inboundSignedVaaBufferSize)
	g.obsvReqC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestInboundBufferSize)
	g.obsvReqSendC = makeChannelPair[*gossipv1.ObservationRequest](observationRequestOutboundBufferSize)
	g.acctC = makeChannelPair[*common.MessagePublication](accountant.MsgChannelCapacity)

	// Guardian set state managed by processor
	g.gst = common.NewGuardianSetState(nil)

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

		g.initializeBasic(rootCtxCancel)
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

		if g.gatewayRelayer != nil {
			logger.Info("Starting gateway relayer")
			if err := g.gatewayRelayer.Start(ctx); err != nil {
				logger.Fatal("failed to start gateway relayer", zap.Error(err), zap.String("component", "gwrelayer"))
			}
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
