package suiclient

// TODO: better error handling
// TODO: debug logging
// TODO: ensure channels are done safely

import (
	"context"
	"fmt"
	"log"
	"math"
	"slices"
	"time"

	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	SuiGrpcTimeout           = 10 * time.Second
	SuiGrpcSteamNilThreshold = 100
	SuiGrpcInvalidVersion    = math.MaxUint64
)

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
	logger                      *zap.Logger
	pbLedgerServiceClient       GrpcLedgerServiceClientInterface       //pb.LedgerServiceClient
	pbSubscriptionServiceClient GrpcSubscriptionServiceClientInterface //pb.SubscriptionServiceClient
}

// Calls `GetObjectAtVersion` with an invalid version, used to signal retrieving the
// most current version of the object.
// docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-object
func (s *SuiGrpcClient) GetObject(ctx context.Context, objectID string) (SuiObject, error) {
	return s.GetObjectAtVersion(ctx, objectID, SuiGrpcInvalidVersion)
}

// To replace the old `suix_tryMultiGetPastObjects` behaviour, this method needs to be called
// for each version of interest.
//
// docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-object
func (s *SuiGrpcClient) GetObjectAtVersion(ctx context.Context, objectID string, version uint64) (SuiObject, error) {
	// If the requested version is SuiGrpcInvalidVersion, nil the pointer to request the latest version of the object.
	requestedVersion := &version
	if *requestedVersion == SuiGrpcInvalidVersion {
		requestedVersion = nil
	}

	fields := []string{
		"object_id",
		// NOTE: "contents" exists only for Move structs. If other types of
		// objects are desired, "bcs" needs to be specified
		"contents",
		"object_type",
	}
	getObjectRequest := pb.GetObjectRequest{
		ObjectId: &objectID,
		Version:  requestedVersion,
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetObject(ctx, &getObjectRequest)

	// gRPC call error check
	if err != nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject failed: %v", err)
	}

	// nil-checks for top-level properties
	if resp == nil || resp.Object == nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject returned nil top-level properties")
	}

	// nil-checks for ObjectId and ObjectType
	if resp.Object.ObjectId == nil || resp.Object.ObjectType == nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject returned nil ObjectId/ObjectType")
	}

	// nil-checks for Contents
	if resp.Object.Contents == nil || resp.Object.Contents.Name == nil || resp.Object.Contents.Value == nil {
		return SuiObject{}, fmt.Errorf("sui gRPC GetObject returned nil Contents properties")
	}

	return SuiObject{
		ID:         *resp.Object.ObjectId,
		ObjectType: *resp.Object.ObjectType,
		BcsType:    *resp.Object.Contents.Name,
		BcsBytes:   resp.Object.Contents.Value,
	}, nil
}

func (s *SuiGrpcClient) GetLatestCheckpointSN(ctx context.Context) (uint64, error) {

	// NOTE: "digest" can be included here, if there is any need in the future.
	fields := []string{
		"sequence_number",
	}

	getCheckpointRequest := pb.GetCheckpointRequest{
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetCheckpoint(ctx, &getCheckpointRequest)

	// gRPC call error check
	if err != nil {
		return 0, fmt.Errorf("sui gRPC GetCheckpoint failed: %v", err)
	}

	// nil-check
	if resp == nil || resp.Checkpoint == nil || resp.Checkpoint.SequenceNumber == nil {
		return 0, fmt.Errorf("sui gRPC GetCheckpoint returned nil properties")
	}

	return *resp.Checkpoint.SequenceNumber, nil
}

// Replaces `sui_getTransactionBlock`.
// Docs: https://www.quicknode.com/docs/sui/sui-grpc/ledger/get-transaction
func (s *SuiGrpcClient) GetTransaction(ctx context.Context, digest string) (SuiTransaction, error) {

	fields := []string{
		"digest",
		"events",
		"effects",
	}

	getTransactionRequest := pb.GetTransactionRequest{
		Digest:   &digest,
		ReadMask: fieldMask(fields),
	}

	resp, err := s.pbLedgerServiceClient.GetTransaction(ctx, &getTransactionRequest)

	// gRPC call error check
	if err != nil {
		log.Fatalf("GetTransaction failed: %v", err)
	}

	// nil-check for top-level properties
	if resp == nil || resp.Transaction == nil {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction returned nil properties")
	}

	// nil-check for inner properties
	if resp.Transaction.Digest == nil || resp.Transaction.Events == nil {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction returned nil Digest/Events")
	}

	suiTransaction := grpcExecutedTransactionToSuiTransaction(resp.Transaction)

	if suiTransaction == nil {
		return SuiTransaction{}, fmt.Errorf("sui gRPC GetTransaction failed to convert gRPC tx to Sui tx")
	}

	return *suiTransaction, nil
}

func (s *SuiGrpcClient) createCheckpointStream(ctx context.Context, fields []string) (pb.SubscriptionService_SubscribeCheckpointsClient, error) {

	// Prepare SubscribeCheckpointsRequest
	subscribeCheckpointsRequest := pb.SubscribeCheckpointsRequest{
		ReadMask: fieldMask(fields),
	}

	stream, err := s.pbSubscriptionServiceClient.SubscribeCheckpoints(ctx, &subscribeCheckpointsRequest)

	return stream, err
}

func (s *SuiGrpcClient) SubscribeToEvent(ctx context.Context, event string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error) {
	eventTypes := []string{
		event,
	}

	return s.SubscribeToEvents(ctx, eventTypes, eventWriteChannel)
}

func (s *SuiGrpcClient) SubscribeToEvents(ctx context.Context, eventTypes []string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error) {

	// This stream is only concerned with transaction events
	fields := []string{
		"transactions.events",
	}

	// Create a stream
	stream, err := s.createCheckpointStream(ctx, fields)

	if err != nil {
		return SuiSubscription{}, fmt.Errorf("sui gRPC CheckpointStream creation failed: %v", err)
	}

	// Create a cancel context for use in the subscription
	ctx, cancel := context.WithCancel(ctx)

	// Set up subscription
	errorChannel := make(chan error, 1)

	subscription := SuiSubscription{
		err:       errorChannel,
		ctxCancel: cancel,
	}

	go func() {
		defer cancel()

		streamNilRespCounter := uint64(0)

		for {
			select {
			case <-ctx.Done():
				s.logger.Debug("Closing Sui gRPC subscription")
				return
			default:
				resp, err := stream.Recv()

				// Note that the stream isn't ended here. It's up to the subscription creator to decide if they want to end the stream by
				// unsubscribing from the subscription.
				if err != nil {
					errorChannel <- err
				}

				// Check that the response and Checkpoint are non-nil before further processing.
				if resp == nil || resp.Checkpoint == nil {

					// Whenever the stream produces nil, the nil responses counter is incremented. When the counter
					// reaches a certain threshold, a debug log is produced.
					streamNilRespCounter = streamNilRespCounter + 1
					if streamNilRespCounter%SuiGrpcSteamNilThreshold == 0 {
						s.logger.Debug("Sui gRPC nil response update", zap.Uint64("streamNilRespCounter", uint64(streamNilRespCounter)))
					}

					continue
				}

				executedTransactions := resp.Checkpoint.Transactions

				// Iterate over all executed transactions
				for _, tx := range executedTransactions {

					// If there are no events, proceed to next transaction
					if tx.Events == nil || len(tx.Events.Events) == 0 {
						continue
					}

					grpcEvents := tx.Events.Events

					// Iterate over events
					for _, grpcEvent := range grpcEvents {

						// If the event types match, write it to the channel
						if slices.Contains(eventTypes, *grpcEvent.EventType) {
							suiEvent := grpcEventToSuiEvent(grpcEvent)

							// Only write to the event channel if the gRPC Event -> SuiEvent conversion succeeded
							if suiEvent != nil {
								eventWriteChannel <- *suiEvent
							}

						}
					}
				}
			}
		}
	}()

	return subscription, nil
}

// Create a new SuiClient, with the gRPC service as iplementation.
func NewSuiGrpcClient(rpcURL string, logger *zap.Logger) (SuiClient, error) {

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(rpcURL, opts...)

	if err != nil {
		return nil, fmt.Errorf("sui gRPC client creation failed: %v", err)
	}

	grpcLedgerServiceClient := &GrpcLedgerServiceClient{
		pbLedgerServiceClient: pb.NewLedgerServiceClient(conn),
	}

	grpcSubscriptionServiceClient := &GrpcSubscriptionServiceClient{
		pbSubscriptionServiceClient: pb.NewSubscriptionServiceClient(conn),
	}

	return newSuiGrpcClientWithServices(logger, grpcLedgerServiceClient, grpcSubscriptionServiceClient), nil
}

// A private function to construct the gRPC client from its most basic components. This is kept private, since the intended use for production is
// via NewSuiGrpcClient, which creates live service clients.  For testing, this function can be used to supply mock versions of the service clients.
// There is no need to check that `ledgerServiceClient` or `subscriptionServiceClient` is nil, because the intended use is via `NewSuiGrpcClient`,
// which instantiates these objects.
func newSuiGrpcClientWithServices(logger *zap.Logger, ledgerServiceClient GrpcLedgerServiceClientInterface, subscriptionServiceClient GrpcSubscriptionServiceClientInterface) SuiClient {
	return &SuiGrpcClient{
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

// Convert a transaction from the Sui gRPC API to a SuiTransaction. This guarantees that the gRPC transaction is well-formed. If any of
// the required fields are missing, the function returns `nil`, signalling to the caller that the API response is malformed.
func grpcExecutedTransactionToSuiTransaction(grpcTransaction *pb.ExecutedTransaction) *SuiTransaction {
	var suiEvents []SuiEvent

	// nil-check the required transaction properties
	if grpcTransaction == nil || grpcTransaction.Events == nil || grpcTransaction.Digest == nil {
		return nil
	}

	for _, event := range grpcTransaction.Events.Events {

		// Convert the gRPC event to a SuiEvent
		suiEventPtr := grpcEventToSuiEvent(event)

		// Dereference the converted suiEvent if it's non-nil, implying successful conversion
		if suiEventPtr != nil {
			suiEvents = append(suiEvents, *suiEventPtr)
		}

	}

	return &SuiTransaction{
		Digest: *grpcTransaction.Digest,
		Events: suiEvents,
	}
}
