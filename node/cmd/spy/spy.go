package spy

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

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
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string

	statusAddr *string

	nodeKeyPath *string

	logLevel *string

	spyRPC *string
)

func init() {
	p2pNetworkID = SpyCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = SpyCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = SpyCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = SpyCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = SpyCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	logLevel = SpyCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	spyRPC = SpyCmd.Flags().String("spyRPC", "", "Listen address for gRPC interface")
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
	subsAllVaa      map[string]*subscriptionAllVaa
	subsAllVaaMu    sync.Mutex
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
type subscriptionAllVaa struct {
	filters []*spyv1.FilterEntry
	ch      chan *spyv1.SubscribeSignedVAAByTypeResponse
}

func subscriptionId() string {
	return uuid.New().String()
}

func (s *spyServer) PublishSignedVAA(vaaBytes []byte) error {
	s.subsSignedVaaMu.Lock()
	defer s.subsSignedVaaMu.Unlock()

	var v *vaa.VAA

	for _, sub := range s.subsSignedVaa {
		if len(sub.filters) == 0 {
			sub.ch <- message{vaaBytes: vaaBytes}
			continue
		}

		if v == nil {
			var err error
			v, err = vaa.Unmarshal(vaaBytes)
			if err != nil {
				return err
			}
		}

		for _, fi := range sub.filters {
			if fi.chainId == v.EmitterChain && fi.emitterAddr == v.EmitterAddress {
				sub.ch <- message{vaaBytes: vaaBytes}
			}
		}

	}

	return nil
}

// TransactionIdMatches checks if both TxIds have the same value.
func TransactionIdMatches(g *gossipv1.SignedBatchVAAWithQuorum, t *spyv1.BatchFilter) bool {
	return bytes.Equal(g.TxId, t.TxId)
}

// BatchMatchFilter asserts that the obervation matches the values of the filter.
func BatchMatchesFilter(g *gossipv1.SignedBatchVAAWithQuorum, f *spyv1.BatchFilter) bool {
	// check the chain ID
	if g.ChainId != uint32(f.ChainId) {
		return false
	}

	// check the transaction ID
	txMatch := TransactionIdMatches(g, f)
	if !txMatch {
		return false
	}

	// check the Nonce
	if f.Nonce >= 1 {
		// filter has a nonce, so make sure it matches
		if g.Nonce != f.Nonce {
			// filter's nonce does not match the nonce of the Batch.
			return false
		}
	}

	return true
}

// HandleGossipVAA compares a gossip message to client subscriptions & filters,
// and forwards the VAA to those requesting it.
func (s *spyServer) HandleGossipVAA(g *gossipv1.SignedVAAWithQuorum) error {
	s.subsAllVaaMu.Lock()
	defer s.subsAllVaaMu.Unlock()

	v, err := vaa.Unmarshal(g.Vaa)
	if err != nil {
		s.logger.Error("failed unmarshaing VAA bytes from gossipv1.SignedVAAWithQuorum.",
			zap.Error(err))
		return err
	}

	// resType defines which oneof proto will be retuned - res type "SignedVaa" is *gossipv1.SignedVAAWithQuorum
	resType := &spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa{
		SignedVaa: g,
	}

	// envelope is the highest level proto struct, the wrapper proto that contains one of the VAA types.
	envelope := &spyv1.SubscribeSignedVAAByTypeResponse{
		VaaType: resType,
	}

	// loop through the subscriptions and send responses to everyone that wants this VAA
	for _, sub := range s.subsAllVaa {
		if len(sub.filters) == 0 {
			// this subscription has no filters, send them the VAA.
			sub.ch <- envelope
			continue
		}

		// this subscription has filters.
		for _, filterEntry := range sub.filters {
			filter := filterEntry.GetFilter()
			switch t := filter.(type) {
			case *spyv1.FilterEntry_EmitterFilter:
				filterAddr := t.EmitterFilter.EmitterAddress
				filterChain := vaa.ChainID(t.EmitterFilter.ChainId)

				if v.EmitterChain == filterChain && v.EmitterAddress.String() == filterAddr {
					// it is a match, send the response
					sub.ch <- envelope
				}
			default:
				panic(fmt.Sprintf("unsupported filter type in subscriptions: %T", filter))
			}
		}

	}

	return nil
}

// HandleGossipBatchVAA compares a gossip message to client subscriptions & filters,
// and forwards the VAA to those requesting it.
func (s *spyServer) HandleGossipBatchVAA(g *gossipv1.SignedBatchVAAWithQuorum) error {
	s.subsAllVaaMu.Lock()
	defer s.subsAllVaaMu.Unlock()

	b, err := vaa.UnmarshalBatch(g.BatchVaa)
	if err != nil {
		s.logger.Error("failed unmarshaing BatchVAA bytes from gossipv1.SignedBatchVAAWithQuorum.",
			zap.Error(err))
		return err
	}

	// resType defines which oneof proto will be retuned -
	// res type "SignedBatchVaa" is *gossipv1.SignedBatchVAAWithQuorum
	resType := &spyv1.SubscribeSignedVAAByTypeResponse_SignedBatchVaa{
		SignedBatchVaa: g,
	}

	// envelope is the highest level proto struct, the wrapper proto that contains one of the VAA types.
	envelope := &spyv1.SubscribeSignedVAAByTypeResponse{
		VaaType: resType,
	}

	// loop through the subscriptions and send responses to everyone that wants this VAA
	for _, sub := range s.subsAllVaa {
		if len(sub.filters) == 0 {
			// this subscription has no filters, send them the VAA.
			sub.ch <- envelope
			continue
		}

		// this subscription has filters.
		for _, filterEntry := range sub.filters {
			filter := filterEntry.GetFilter()
			switch t := filter.(type) {
			case *spyv1.FilterEntry_EmitterFilter:

				filterChain := uint32(t.EmitterFilter.ChainId)
				if g.ChainId != filterChain {
					// VAA does not pass the filter
					continue
				}

				// BatchVAAs do not have EmitterAddress at the top level - each Observation
				// in the Batch has an EmitterAddress.

				// In order to make it easier for integrators, allow subscribing to BatchVAAs by
				// EmitterFilter. Send BatchVAAs to subscriptions with an EmitterFilter that
				// matches 1 (or more) Obervation(s) in the batch.

				filterAddr := t.EmitterFilter.EmitterAddress

				// check each Observation to see if it meets the criteria of the filter.
				for _, obs := range b.Observations {
					if obs.Observation.EmitterAddress.String() == filterAddr {
						// it is a match, send the response to the subscriber.
						sub.ch <- envelope
						break
					}

				}
			case *spyv1.FilterEntry_BatchFilter:
				if BatchMatchesFilter(g, t.BatchFilter) {
					sub.ch <- envelope
				}
			case *spyv1.FilterEntry_BatchTransactionFilter:
				// make a BatchFilter struct from the BatchTransactionFilter since the latter is
				// a subset of the former's properties, so we can use TransactionIdMatches.
				batchFilter := &spyv1.BatchFilter{
					ChainId: t.BatchTransactionFilter.ChainId,
					TxId:    t.BatchTransactionFilter.TxId,
				}

				if BatchMatchesFilter(g, batchFilter) {
					sub.ch <- envelope
				}
			default:
				panic(fmt.Sprintf("unsupported filter type in subscriptions: %T", filter))
			}
		}
	}
	return nil
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
				fi = append(fi, filterSignedVaa{
					chainId:     vaa.ChainID(t.EmitterFilter.ChainId),
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
		s.subsSignedVaaMu.Lock()
		defer s.subsSignedVaaMu.Unlock()
		delete(s.subsSignedVaa, id)
	}()

	for {
		select {
		case <-resp.Context().Done():
			return resp.Context().Err()
		case msg := <-sub.ch:
			if err := resp.Send(&spyv1.SubscribeSignedVAAResponse{
				VaaBytes: msg.vaaBytes,
			}); err != nil {
				return err
			}
		}
	}
}

// SubscribeSignedVAAByType fields requests for subscriptions. Each new subscription adds a channel and request params (filters)
// to the map of active subscriptions.
func (s *spyServer) SubscribeSignedVAAByType(req *spyv1.SubscribeSignedVAAByTypeRequest, resp spyv1.SpyRPCService_SubscribeSignedVAAByTypeServer) error {
	var fi []*spyv1.FilterEntry
	if req.Filters != nil {
		for _, f := range req.Filters {
			switch t := f.Filter.(type) {

			case *spyv1.FilterEntry_EmitterFilter:
				// validate the emitter address is valid by decoding it
				_, err := vaa.StringToAddress(t.EmitterFilter.EmitterAddress)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode emitter address: %v", err))
				}
				fi = append(fi, &spyv1.FilterEntry{Filter: t})

			case *spyv1.FilterEntry_BatchFilter,
				*spyv1.FilterEntry_BatchTransactionFilter:
				fi = append(fi, &spyv1.FilterEntry{Filter: t})
			default:
				return status.Error(codes.InvalidArgument, "unsupported filter type")
			}
		}
	}

	s.subsAllVaaMu.Lock()
	id := subscriptionId()
	sub := &subscriptionAllVaa{
		ch:      make(chan *spyv1.SubscribeSignedVAAByTypeResponse, 1),
		filters: fi,
	}
	s.subsAllVaa[id] = sub
	s.subsAllVaaMu.Unlock()

	defer func() {
		s.subsAllVaaMu.Lock()
		defer s.subsAllVaaMu.Unlock()
		delete(s.subsAllVaa, id)
	}()

	for {
		select {
		case <-resp.Context().Done():
			return resp.Context().Err()
		case msg := <-sub.ch:
			if err := resp.Send(msg); err != nil {
				return err
			}
		}
	}
}

func newSpyServer(logger *zap.Logger) *spyServer {
	return &spyServer{
		logger:        logger.Named("spyserver"),
		subsSignedVaa: make(map[string]*subscriptionSignedVaa),
		subsAllVaa:    make(map[string]*subscriptionAllVaa),
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

	// Outbound gossip message queue
	sendC := make(chan []byte)

	// Inbound observations
	obsvC := make(chan *gossipv1.SignedObservation, 50)

	// Inbound observation requests
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)

	// Inbound observation requests
	queryReqC := make(chan *gossipv1.SignedQueryRequest, 50)

	// Inbound signed VAAs
	signedInC := make(chan *gossipv1.SignedVAAWithQuorum, 50)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState(nil)

	// RPC server
	s := newSpyServer(logger)
	rpcSvc, _, err := spyServerRunnable(s, logger, *spyRPC)
	if err != nil {
		logger.Fatal("failed to start RPC server", zap.Error(err))
	}

	// Ignore observations
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-obsvC:
			}
		}
	}()

	// Ignore observation requests
	// Note: without this, the whole program hangs on observation requests
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-obsvReqC:
			}
		}
	}()

	// Ignore query requests
	// Note: without this, the whole program hangs on query requests
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-queryReqC:
			}
		}
	}()

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
					logger.Error("failed to publish signed VAA", zap.Error(err))
				}
				if err := s.HandleGossipVAA(v); err != nil {
					logger.Error("failed to HandleGossipVAA", zap.Error(err))
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
		if err := supervisor.Run(ctx,
			"p2p",
			p2p.Run(obsvC,
				obsvReqC,
				nil,
				sendC,
				signedInC,
				priv,
				nil,
				gst,
				*p2pNetworkID,
				*p2pBootstrap,
				"",
				false,
				rootCtxCancel,
				nil,
				nil,
				nil,
				nil,
				components,
				nil, // ibc feature string
				queryReqC,
			)); err != nil {
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
