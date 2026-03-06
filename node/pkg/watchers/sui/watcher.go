package sui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"encoding/hex"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/suiclient"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mr-tron/base58"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	txverifier "github.com/certusone/wormhole/node/pkg/txverifier"
)

// suiGrpcDialOpts returns the gRPC dial options for connecting to the Sui endpoint. In unsafe
// dev mode the local Sui node serves plaintext gRPC, so TLS is disabled; otherwise the default
// (TLS) transport configured by suiclient.NewSuiGrpcClient is used.
func suiGrpcDialOpts(unsafeDevMode bool) []grpc.DialOption {
	if unsafeDevMode {
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	return nil
}

type (
	// Watcher is responsible for looking over Sui blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		suiRPC           string
		suiMoveEventType string

		unsafeDevMode bool

		msgChan       chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component

		// Note: suiclient.SuiClient is an interface. A `nil` check is therefore fine for checking initialization.
		suiClient suiclient.SuiClient

		// Sui transaction verifier
		suiTxVerifier     *txverifier.SuiTransferVerifier
		txVerifierEnabled bool
	}
)

var (
	suiMessagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_sui_observations_confirmed_total",
			Help: "Total number of verified Sui observations found",
		})
	currentSuiHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_sui_current_height",
			Help: "Current Sui block height",
		})
	suiTransferVerifierFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_sui_txverifier_failures",
			Help: "Total number of messages that failed transfer verification",
		})
)

// NewWatcher creates a new Sui appid watcher
func NewWatcher(
	suiRPC string,
	suiMoveEventType string,
	unsafeDevMode bool,
	messageEvents chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	env common.Environment,
	txVerifierEnabled bool,
) (*Watcher, error) {
	var suiTxVerifier *txverifier.SuiTransferVerifier

	if txVerifierEnabled {

		// Extracted from the suiMoveEventType passed to guardiand as CLI arg
		var suiCoreBridgePackageId string

		//	Read from a hardcoded map in the txverifier package, based on the environment
		//	NOTE: this is the original package ID, not the current one. The original package ID is used to address
		//	the token registry that holds the wrapped and native assets of the bridge.
		var suiTokenBridgePackageId string

		// Read from the sdk, based on the environment
		var suiTokenBridgeEmitter string

		// Split the suiMoveEventType into its components. If the format is incorrect, return an error.
		eventTypeComponents := strings.Split(suiMoveEventType, "::")
		if len(eventTypeComponents) != 3 {
			return nil, fmt.Errorf("suiMoveEventType is not in the correct format, expected <package_id>::<module_name>::<event_name>, got: %s", suiMoveEventType)
		}

		suiCoreBridgePackageId = eventTypeComponents[0]

		// Retrieve the token bridge package ID and token bridge emitter address, based on the environment
		switch env {
		case common.MainNet:
			suiTokenBridgePackageId = txverifier.SuiOriginalTokenBridgePackageIds[common.MainNet]
			suiTokenBridgeEmitter = "0x" + hex.EncodeToString(sdk.KnownTokenbridgeEmitters[vaa.ChainIDSui])
		case common.TestNet:
			suiTokenBridgePackageId = txverifier.SuiOriginalTokenBridgePackageIds[common.TestNet]
			suiTokenBridgeEmitter = "0x" + hex.EncodeToString(sdk.KnownTestnetTokenbridgeEmitters[vaa.ChainIDSui])
		case common.UnsafeDevNet, common.AccountantMock, common.GoTest:
			suiTokenBridgePackageId = txverifier.SuiOriginalTokenBridgePackageIds[common.UnsafeDevNet]
			suiTokenBridgeEmitter = "0x" + hex.EncodeToString(sdk.KnownTokenbridgeEmitters[vaa.ChainIDSui])
		}

		// Create the Sui gRPC client used by the transfer verifier to query transactions and objects.
		suiClient, err := suiclient.NewSuiGrpcClient(suiRPC, nil, suiGrpcDialOpts(unsafeDevMode)...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Sui gRPC client for transfer verifier: %w", err)
		}

		// Create the new suiTxVerifier
		suiTxVerifier = txverifier.NewSuiTransferVerifier(
			suiCoreBridgePackageId,
			suiTokenBridgeEmitter,
			suiTokenBridgePackageId,
			suiClient,
		)

	}

	return &Watcher{
		suiRPC:            suiRPC,
		suiMoveEventType:  suiMoveEventType,
		unsafeDevMode:     unsafeDevMode,
		msgChan:           messageEvents,
		obsvReqC:          obsvReqC,
		readinessSync:     common.MustConvertChainIdToReadinessSyncing(vaa.ChainIDSui),
		suiTxVerifier:     suiTxVerifier,
		txVerifierEnabled: txVerifierEnabled,
	}, nil
}

// processEvent decodes a single Sui gRPC event into a Wormhole MessagePublication and,
// after optional transfer verification, publishes it to the message channel. `txDigest`
// is the base58-encoded Sui transaction digest the event belongs to; it is supplied
// explicitly because gRPC events fetched via GetTransaction do not carry it.
func (e *Watcher) processEvent(ctx context.Context, logger *zap.Logger, event suiclient.SuiEvent, txDigest string, isReobservation bool) error {
	// The subscription is already filtered by event type, but reobservation returns every
	// event in the transaction, so re-check the type here before decoding.
	if event.EventType != e.suiMoveEventType {
		return nil
	}

	msg, err := suiclient.DecodeBcs[txverifier.WormholeMessage](event.BcsBytes)
	if err != nil {
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return fmt.Errorf("processEvent failed to decode WormholeMessage BCS bytes: %w", err)
	}

	txHashBytes, err := base58.Decode(txDigest)
	if err != nil {
		return fmt.Errorf("processEvent failed to base58-decode txDigest %s: %w", txDigest, err)
	}

	if len(txHashBytes) != 32 {
		logger.Error(
			"Transaction hash is not 32 bytes",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("txHash", txDigest),
		)
		return errors.New("transaction hash is not 32 bytes")
	}

	txHashEthFormat := eth_common.BytesToHash(txHashBytes)

	observation := &common.MessagePublication{
		TxID:             txHashEthFormat.Bytes(),
		Timestamp:        time.Unix(int64(msg.Timestamp), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            msg.Nonce,
		Sequence:         msg.Sequence,
		EmitterChain:     vaa.ChainIDSui,
		EmitterAddress:   vaa.Address(msg.Sender),
		Payload:          msg.Payload,
		ConsistencyLevel: msg.ConsistencyLevel,
		IsReobservation:  isReobservation,
		Unreliable:       false,
	}

	// Verifies the observation through the Sui transaction verifier, if enabled, followed
	// by publishing the observation to the message channel.
	if err := e.verifyAndPublish(ctx, observation, txDigest, logger); err != nil {
		suiTransferVerifierFailures.Inc()
		logger.Error("Message publication error",
			zap.String("TxDigest", txDigest),
			zap.Error(err))
	}

	return nil
}

func (e *Watcher) verifyAndPublish(
	ctx context.Context,
	msg *common.MessagePublication,
	txDigest string,
	logger *zap.Logger,
) error {
	if msg == nil {
		return errors.New("MessagePublication is nil")
	}

	if e.suiTxVerifier != nil {
		verifiedMsg, err := e.verify(ctx, msg, txDigest, logger)

		if err != nil {
			return err
		}

		msg = &verifiedMsg
	}

	e.msgChan <- msg // Note on channel capacity: The channel to the processor is buffered and shared across chains, if it backs up we should stop processing new observations

	suiMessagesConfirmed.Inc()
	if msg.IsReobservation {
		watchers.ReobservationsByChain.WithLabelValues("sui", "std").Inc()
	}

	logger.Info("message observed",
		msg.ZapFields()...,
	)

	return nil
}

// handleReobservation fetches the transaction identified by a re-observation request and
// re-processes each of its Wormhole message events.
func (e *Watcher) handleReobservation(ctx context.Context, logger *zap.Logger, client suiclient.SuiClient, r *gossipv1.ObservationRequest) {
	// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
	// and only writes to the channel for this chain id.
	// If either of the below cases are true, something has gone wrong.
	if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != vaa.ChainIDSui {
		panic("invalid chain ID")
	}

	tx58 := base58.Encode(r.TxHash)

	txn, err := client.GetTransaction(ctx, tx58, []string{
		suiclient.TransactionFieldDigest,
		suiclient.TransactionFieldEvents,
	})
	if err != nil {
		logger.Error("sui_fetch_obvs_req failed", zap.String("txhash", tx58), zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDSui, 1)
		return
	}

	for i, event := range txn.Events {
		// Events returned by GetTransaction do not carry their transaction digest, so it is
		// supplied explicitly here.
		if err := e.processEvent(ctx, logger, event, tx58, true); err != nil {
			logger.Info("sui_fetch_obvs_req skipping event data in result", zap.String("txhash", tx58), zap.Int("index", i), zap.Error(err))
		}
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
		ContractAddress: e.suiMoveEventType,
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", "sui"),
		zap.String("suiRPC", e.suiRPC),
		zap.String("suiMoveEventType", e.suiMoveEventType),
		zap.Bool("unsafeDevMode", e.unsafeDevMode),
	)

	// Use an injected client (e.g. from tests) if present, otherwise create one for the
	// lifetime of this Run. The client is kept local rather than stored on the Watcher so
	// that each supervisor-driven restart of Run establishes a fresh connection and the
	// concurrent goroutines below cannot observe a closed or nil client during shutdown.
	client := e.suiClient
	if client == nil {
		grpcClient, err := suiclient.NewSuiGrpcClient(e.suiRPC, logger, suiGrpcDialOpts(e.unsafeDevMode)...)
		if err != nil {
			return fmt.Errorf("failed to create Sui gRPC client: %w", err)
		}
		client = grpcClient
		defer func() {
			if cerr := client.Close(); cerr != nil {
				logger.Error("failed to close Sui gRPC client", zap.Error(cerr))
			}
		}()
	}

	// Get the latest checkpoint sequence number to confirm connectivity before reporting healthy.
	initialCheckpoint, err := client.GetLatestCheckpoint(ctx, []string{suiclient.CheckpointFieldSequenceNumber})
	if err != nil {
		return fmt.Errorf("failed to get latest checkpoint sequence number: %w", err)
	} else if initialCheckpoint.SequenceNumber == nil {
		return fmt.Errorf("latest checkpoint response missing sequence number")
	}

	currentSuiHeight.Set(float64(*initialCheckpoint.SequenceNumber))

	timer := time.NewTicker(time.Second * 5)
	defer timer.Stop()

	errC := make(chan error)

	supervisor.Signal(ctx, supervisor.SignalHealthy)
	readiness.SetReady(e.readinessSync)

	common.RunWithScissors(ctx, errC, "sui_data_pump", func(ctx context.Context) error {
		eventChan := make(chan suiclient.SuiTransactionEvent, 64)

		subscription, err := client.SubscribeToTransactionEvent(ctx, e.suiMoveEventType, eventChan)
		if err != nil {
			return fmt.Errorf("sui_data_pump failed to subscribe to events: %w", err)
		}
		defer subscription.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_data_pump context done")
				return ctx.Err()

			case subErr := <-subscription.Err():
				return fmt.Errorf("sui_data_pump subscription error: %w", subErr)

			case txEvent := <-eventChan:
				if txEvent.TxDigest == "" {
					logger.Warn("sui_data_pump received event with empty TxDigest, skipping")
					continue
				}

				if err := e.processEvent(ctx, logger, txEvent.Event, txEvent.TxDigest, false); err != nil {
					logger.Error("sui_data_pump processEvent error", zap.Error(err))
				}
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_block_height", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_block_height context done")
				return ctx.Err()

			case <-timer.C:
				checkpoint, err := client.GetLatestCheckpoint(ctx, []string{suiclient.CheckpointFieldSequenceNumber})
				if err != nil {
					logger.Error("Failed to get latest checkpoint sequence number", zap.Error(err))
				} else if checkpoint.SequenceNumber == nil {
					logger.Error("latest checkpoint response missing sequence number")
				} else {
					height := int64(*checkpoint.SequenceNumber) // #nosec G115 -- Sui checkpoint sequence numbers will not exceed math.MaxInt64 for the foreseeable future
					currentSuiHeight.Set(float64(*checkpoint.SequenceNumber))
					logger.Debug("sui_getLatestCheckpointSequenceNumber", zap.Int64("result", height))

					p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDSui, &gossipv1.Heartbeat_Network{
						Height:          height,
						ContractAddress: e.suiMoveEventType,
					})
				}

				readiness.SetReady(e.readinessSync)
			}
		}
	})

	common.RunWithScissors(ctx, errC, "sui_fetch_obvs_req", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				logger.Error("sui_fetch_obvs_req context done")
				return ctx.Err()
			case r := <-e.obsvReqC:
				e.handleReobservation(ctx, logger, client, r)
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
