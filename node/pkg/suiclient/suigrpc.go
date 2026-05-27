package suiclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"slices"

	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// suiGrpcNilResponses counts nil/empty checkpoint responses received from the Sui
// subscription stream. A persistently rising rate indicates the upstream node is
// returning malformed checkpoints and should be alerted on.
var suiGrpcNilResponses = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "wormhole_sui_grpc_nil_responses_total",
		Help: "Total number of nil checkpoint responses received from the Sui gRPC subscription stream",
	})

type GrpcLedgerServiceClientInterface interface {
	GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error)
	GetCheckpoint(ctx context.Context, req *pb.GetCheckpointRequest) (*pb.GetCheckpointResponse, error)
	GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error)
}

type GrpcSubscriptionServiceClientInterface interface {
	SubscribeCheckpoints(ctx context.Context, req *pb.SubscribeCheckpointsRequest) (pb.SubscriptionService_SubscribeCheckpointsClient, error)
}

// The Sui gRPC client accepts interfaces for the ledger and subscription services. This allows creating mocks
// of the gRPC server, to enable thorough testing of parsing logic. Note that the interfaces defined above have
// the same signatures as the gRPC methods. This is because the interface is only meant for drop-in replacements
// of mocks that return different data. Requiring the implementations of the interfaces to do additional parsing
// is unnecessary.
type SuiGrpcClient struct {
	conn                        *grpc.ClientConn
	logger                      *zap.Logger
	pbLedgerServiceClient       GrpcLedgerServiceClientInterface       //pb.LedgerServiceClient
	pbSubscriptionServiceClient GrpcSubscriptionServiceClientInterface //pb.SubscriptionServiceClient
}

// GetObject fetches the latest version of `objectID` populated with the
// requested `fields` (see ObjectField* constants in suiclient.go).
// docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-object
//
// Returned-field nil-checking is the caller's responsibility — any field
// not requested OR not returned by the upstream node comes back nil/empty,
// and this method does not enforce per-field presence.
func (s *SuiGrpcClient) GetObject(ctx context.Context, objectID string, fields []string) (SuiObject, error) {
	return s.GetObjectAtVersion(ctx, objectID, nil, fields)
}

// GetObjectAtVersion fetches `objectID` at the given version, populated with
// the requested `fields` (see ObjectField* constants in suiclient.go). A nil
// `version` requests the latest version. To replace the old
// `suix_tryMultiGetPastObjects` behaviour, call this method once per version
// of interest.
//
// docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-object
//
// Returned-field nil-checking is the caller's responsibility — any field
// not requested OR not returned by the upstream node comes back nil/empty,
// and this method does not enforce per-field presence.
func (s *SuiGrpcClient) GetObjectAtVersion(ctx context.Context, objectID string, version *uint64, fields []string) (SuiObject, error) {
	if len(fields) == 0 {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject requires at least one field for objectID=%s", objectID)
	}

	versionStr := "latest"
	if version != nil {
		versionStr = fmt.Sprintf("%d", *version)
	}

	getObjectRequest := pb.GetObjectRequest{
		ObjectId: &objectID,
		Version:  version,
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetObject(ctx, &getObjectRequest)

	if err != nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject failed for objectID=%s version=%s fields=%v: %w", objectID, versionStr, fields, err)
	}

	if resp == nil || resp.Object == nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject returned nil top-level properties for objectID=%s version=%s fields=%v", objectID, versionStr, fields)
	}

	return grpcObjectToSuiObject(resp.Object), nil
}

// GetLatestCheckpoint fetches the most recent checkpoint populated with the
// requested `fields` (see CheckpointField* constants in suiclient.go).
//
// Returned-field nil-checking is the caller's responsibility — any field
// not requested OR not returned by the upstream node comes back nil/empty,
// and this method does not enforce per-field presence.
func (s *SuiGrpcClient) GetLatestCheckpoint(ctx context.Context, fields []string) (SuiCheckpoint, error) {
	if len(fields) == 0 {
		return SuiCheckpoint{}, fmt.Errorf("sui gRPC GetLatestCheckpoint requires at least one field")
	}

	getCheckpointRequest := pb.GetCheckpointRequest{
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetCheckpoint(ctx, &getCheckpointRequest)

	if err != nil {
		return SuiCheckpoint{}, fmt.Errorf("sui gRPC GetLatestCheckpoint failed for fields=%v: %w", fields, err)
	}

	if resp == nil || resp.Checkpoint == nil {
		return SuiCheckpoint{}, fmt.Errorf("sui gRPC GetLatestCheckpoint returned nil top-level properties for fields=%v", fields)
	}

	return grpcCheckpointToSuiCheckpoint(resp.Checkpoint), nil
}

// GetTransaction fetches the transaction identified by `digest` populated with
// the requested `fields` (see TransactionField* constants in suiclient.go).
// Replaces `sui_getTransactionBlock`.
// Docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-transaction
//
// Returned-field nil-checking is the caller's responsibility — any field
// not requested OR not returned by the upstream node comes back nil/empty,
// and this method does not enforce per-field presence.
func (s *SuiGrpcClient) GetTransaction(ctx context.Context, digest string, fields []string) (SuiTransaction, error) {
	if len(fields) == 0 {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction requires at least one field for digest=%s", digest)
	}

	getTransactionRequest := pb.GetTransactionRequest{
		Digest:   &digest,
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetTransaction(ctx, &getTransactionRequest)

	if err != nil {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction failed for digest=%s fields=%v: %w", digest, fields, err)
	}

	if resp == nil || resp.Transaction == nil {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction returned nil top-level properties for digest=%s fields=%v", digest, fields)
	}

	return grpcExecutedTransactionToSuiTransaction(resp.Transaction), nil
}

func (s *SuiGrpcClient) createCheckpointStream(ctx context.Context, fields []string) (pb.SubscriptionService_SubscribeCheckpointsClient, error) {

	// Prepare SubscribeCheckpointsRequest
	subscribeCheckpointsRequest := pb.SubscribeCheckpointsRequest{
		ReadMask: fieldMask(fields),
	}

	stream, err := s.pbSubscriptionServiceClient.SubscribeCheckpoints(ctx, &subscribeCheckpointsRequest)

	return stream, err
}

func (s *SuiGrpcClient) SubscribeToTransactionEvent(ctx context.Context, event string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error) {
	eventTypes := []string{
		event,
	}

	return s.SubscribeToTransactionEvents(ctx, eventTypes, eventWriteChannel)
}

func (s *SuiGrpcClient) SubscribeToTransactionEvents(ctx context.Context, eventTypes []string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error) {
	if len(eventTypes) == 0 {
		return SuiSubscription{}, fmt.Errorf("sui gRPC SubscribeToTransactionEvents requires at least one event type")
	}
	if eventWriteChannel == nil {
		return SuiSubscription{}, fmt.Errorf("sui gRPC SubscribeToTransactionEvents requires a non-nil eventWriteChannel")
	}

	// This stream is only concerned with transaction events
	fields := []string{
		"transactions.events",
	}

	// Create a cancel context for use in the subscription
	ctx, cancel := context.WithCancel(ctx)

	// Create a stream
	stream, err := s.createCheckpointStream(ctx, fields)

	if err != nil {
		cancel()
		return SuiSubscription{}, fmt.Errorf("sui gRPC CheckpointStream creation failed for eventTypes=%v: %w", eventTypes, err)
	}

	// Set up subscription
	errorChannel := make(chan error, 1)
	doneChannel := make(chan struct{})

	subscription := SuiSubscription{
		err:       errorChannel,
		done:      doneChannel,
		ctxCancel: cancel,
	}

	go func() {
		defer close(doneChannel)
		defer close(errorChannel)
		defer cancel()

		for {
			// stream.Recv() is interrupted automatically when the context is cancelled.
			resp, err := stream.Recv()

			if err != nil {

				// This is indicative of a context getting cancelled or a context timing out.
				if ctx.Err() != nil {
					s.logger.Debug("Closing Sui gRPC subscription")
					return
				}

				// An RPC communication error occurred. Note that this terminates the goroutine.
				errorChannel <- err
				return
			}

			// Check that the response and Checkpoint are non-nil before further processing.
			if resp == nil || resp.Checkpoint == nil {
				// Increment the prometheus counter so a persistent stream of nil responses can be
				// observed and alerted on by operators.
				suiGrpcNilResponses.Inc()
				continue
			}

			executedTransactions := resp.Checkpoint.Transactions

			// Iterate over all executed transactions.
			for _, tx := range executedTransactions {

				// If there are no events, proceed to next transaction
				if tx.Events == nil || len(tx.Events.Events) == 0 {
					continue
				}

				// Iterate over events.
				for _, grpcEvent := range tx.Events.Events {

					// EventType cannot be nil.
					if grpcEvent.EventType == nil {
						continue
					}

					// If the event type does not match the event types passed to the function, ignore it.
					if !slices.Contains(eventTypes, *grpcEvent.EventType) {
						continue
					}

					suiEvent := grpcEventToSuiEvent(grpcEvent)

					// The grpcEvent was malformed, so ignore it.
					if suiEvent == nil {
						s.logger.Warn("Sui gRPC event was malformed",
							zap.Any("grpcEvent", grpcEvent),
						)
						continue
					}

					// Writing to eventWriteChannel could be blocking if there are no readers. Use a select to include
					// a simultaneous check to bail out if the context is cancelled.
					select {
					case eventWriteChannel <- *suiEvent:
					case <-ctx.Done():
						return
					}

				}
			}
		}

	}()

	return subscription, nil
}

// Close the gRPC connection. This should be called when the client is no longer needed, to avoid leaking resources.
func (s *SuiGrpcClient) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Create a new SuiClient, with the gRPC service as implementation. Additional gRPC dial options
// (e.g. for custom transport credentials, interceptors, or per-RPC metadata via interceptors) may
// be supplied; they are appended after the defaults so callers can override them.
func NewSuiGrpcClient(rpcURL string, logger *zap.Logger, extraOpts ...grpc.DialOption) (SuiClient, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Setting the minimum TLS version is a linting requirement, but should ideally be adhered to in production.
	creds := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	opts := make([]grpc.DialOption, 0, 1+len(extraOpts))
	opts = append(opts, grpc.WithTransportCredentials(creds))
	opts = append(opts, extraOpts...)

	conn, err := grpc.NewClient(rpcURL, opts...)

	if err != nil {
		return nil, fmt.Errorf("sui gRPC client creation failed: %w", err)
	}

	grpcLedgerServiceClient := &GrpcLedgerServiceClient{
		pbLedgerServiceClient: pb.NewLedgerServiceClient(conn),
	}

	grpcSubscriptionServiceClient := &GrpcSubscriptionServiceClient{
		pbSubscriptionServiceClient: pb.NewSubscriptionServiceClient(conn),
	}

	return newSuiGrpcClientWithServices(logger, conn, grpcLedgerServiceClient, grpcSubscriptionServiceClient), nil
}

// A private function to construct the gRPC client from its most basic components. This is kept private, since the intended use for production is
// via NewSuiGrpcClient, which creates live service clients.  For testing, this function can be used to supply mock versions of the service clients.
// There is no need to check that `ledgerServiceClient` or `subscriptionServiceClient` is nil, because the intended use is via `NewSuiGrpcClient`,
// which instantiates these objects.
func newSuiGrpcClientWithServices(logger *zap.Logger, conn *grpc.ClientConn, ledgerServiceClient GrpcLedgerServiceClientInterface, subscriptionServiceClient GrpcSubscriptionServiceClientInterface) SuiClient {
	return &SuiGrpcClient{
		conn:                        conn,
		logger:                      logger,
		pbLedgerServiceClient:       ledgerServiceClient,
		pbSubscriptionServiceClient: subscriptionServiceClient,
	}
}

/*
	The following functions are utility functions specifically for Sui gRPC.
*/

// Accepts a list of strings to create a field mask from. Used by the various gPRC calls for specifying fields to include in responses.
func fieldMask(fields []string) *fieldmaskpb.FieldMask {
	return &fieldmaskpb.FieldMask{
		Paths: fields,
	}
}

// Convert an event from the Sui gRPC API to a SuiEvent. This guarantees that the gRPC event is well-formed. If any of the required fields
// are missing, the function returns `nil`, signalling to the caller that the API response is malformed.
func grpcEventToSuiEvent(grpcEvent *pb.Event) *SuiEvent {

	// nil-check the first set of properties
	if grpcEvent == nil || grpcEvent.PackageId == nil || grpcEvent.Module == nil || grpcEvent.Sender == nil {
		return nil
	}

	// nil-check the remaining properties
	if grpcEvent.EventType == nil || grpcEvent.Contents == nil || grpcEvent.Contents.Name == nil || grpcEvent.Contents.Value == nil {
		return nil
	}

	return &SuiEvent{
		PackageID:         *grpcEvent.PackageId,
		TransactionModule: *grpcEvent.Module,
		Sender:            *grpcEvent.Sender,
		EventType:         *grpcEvent.EventType,
		BcsType:           *grpcEvent.Contents.Name,
		BcsBytes:          grpcEvent.Contents.Value,
	}
}

// grpcExecutedTransactionToSuiTransaction maps a gRPC transaction into the
// wrapper struct. Fields are populated only when present in the proto; absent
// fields are left as their zero value (nil/empty). The caller is responsible
// for nil-checking the fields they read on the returned SuiTransaction.
func grpcExecutedTransactionToSuiTransaction(grpcTransaction *pb.ExecutedTransaction) SuiTransaction {
	out := SuiTransaction{}

	if grpcTransaction == nil {
		return out
	}

	out.Digest = grpcTransaction.Digest
	out.Checkpoint = grpcTransaction.Checkpoint

	if grpcTransaction.Timestamp != nil {
		ts := grpcTransaction.Timestamp.AsTime()
		out.Timestamp = &ts
	}

	if grpcTransaction.Events != nil {
		for _, event := range grpcTransaction.Events.Events {
			if suiEventPtr := grpcEventToSuiEvent(event); suiEventPtr != nil {
				out.Events = append(out.Events, *suiEventPtr)
			}
		}
	}

	return out
}

// grpcObjectToSuiObject maps a gRPC object into the wrapper struct. Fields are
// populated only when present in the proto; absent fields are left as their
// zero value (nil/empty). The Bcs and Contents sub-messages are flattened into
// the corresponding {Bcs,Contents}{Type,Bytes} pairs. The caller is
// responsible for nil-checking the fields they read.
func grpcObjectToSuiObject(grpcObject *pb.Object) SuiObject {
	out := SuiObject{}

	if grpcObject == nil {
		return out
	}

	out.ObjectID = grpcObject.ObjectId
	out.Version = grpcObject.Version
	out.Digest = grpcObject.Digest
	out.ObjectType = grpcObject.ObjectType
	out.PreviousTransaction = grpcObject.PreviousTransaction
	out.StorageRebate = grpcObject.StorageRebate
	out.Balance = grpcObject.Balance

	if grpcObject.Bcs != nil {
		out.BcsType = grpcObject.Bcs.Name
		out.BcsBytes = grpcObject.Bcs.Value
	}

	if grpcObject.Contents != nil {
		out.ContentsType = grpcObject.Contents.Name
		out.ContentsBytes = grpcObject.Contents.Value
	}

	return out
}

// grpcCheckpointToSuiCheckpoint maps a gRPC checkpoint into the wrapper
// struct. Fields are populated only when present in the proto; absent fields
// are left as their zero value (nil/empty). The caller is responsible for
// nil-checking the fields they read.
func grpcCheckpointToSuiCheckpoint(grpcCheckpoint *pb.Checkpoint) SuiCheckpoint {
	out := SuiCheckpoint{}

	if grpcCheckpoint == nil {
		return out
	}

	out.SequenceNumber = grpcCheckpoint.SequenceNumber
	out.Digest = grpcCheckpoint.Digest

	return out
}
