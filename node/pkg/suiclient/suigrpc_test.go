package suiclient

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	mystenbcs "github.com/block-vision/sui-go-sdk/mystenbcs"
	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// subscriptionExitTimeout bounds how long a test waits for a subscription's background
// goroutine to exit. It is generous relative to the normal (microsecond) exit time but
// short enough that a mutant which never terminates the goroutine fails rather than hangs.
const subscriptionExitTimeout = 2 * time.Second

// successfulEffects returns transaction effects carrying an explicit successful execution
// status, as expected by the status checks in GetTransaction and the event subscription.
func successfulEffects() *pb.TransactionEffects {
	success := true
	return &pb.TransactionEffects{Status: &pb.ExecutionStatus{Success: &success}}
}

// A mock LedgerService client for testing. Each of the requests (GetObject, GetCheckpoint, GetTransaction) can
// be simulated by calling the appropriate `SetNext*Response` function, to return a prepared response.
type MockLedgerServiceClient struct {
	nextGetObjectResponse      *pb.GetObjectResponse
	nextGetObjectError         error
	nextGetCheckpointResponse  *pb.GetCheckpointResponse
	nextGetCheckpointError     error
	nextGetTransactionResponse *pb.GetTransactionResponse
	nextGetTransactionError    error

	// lastGetTransactionRequest captures the most recent request so tests can assert
	// on the field mask that GetTransaction builds.
	lastGetTransactionRequest *pb.GetTransactionRequest
}

func (m *MockLedgerServiceClient) SetNextGetObjectResponse(next *pb.GetObjectResponse) {
	m.nextGetObjectResponse = next
}

func (m *MockLedgerServiceClient) SetNextGetObjectError(err error) {
	m.nextGetObjectError = err
}

func (m *MockLedgerServiceClient) GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error) {
	return m.nextGetObjectResponse, m.nextGetObjectError
}

func (m *MockLedgerServiceClient) SetNextGetCheckpointResponse(next *pb.GetCheckpointResponse) {
	m.nextGetCheckpointResponse = next
}

func (m *MockLedgerServiceClient) SetNextGetCheckpointError(err error) {
	m.nextGetCheckpointError = err
}

func (m *MockLedgerServiceClient) GetCheckpoint(ctx context.Context, req *pb.GetCheckpointRequest) (*pb.GetCheckpointResponse, error) {
	return m.nextGetCheckpointResponse, m.nextGetCheckpointError
}

func (m *MockLedgerServiceClient) SetNextGetTransactionResponse(next *pb.GetTransactionResponse) {
	m.nextGetTransactionResponse = next
}

func (m *MockLedgerServiceClient) SetNextGetTransactionError(err error) {
	m.nextGetTransactionError = err
}

func (m *MockLedgerServiceClient) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	m.lastGetTransactionRequest = req
	return m.nextGetTransactionResponse, m.nextGetTransactionError
}

func FuzzSuiGrpcClientGetObject(f *testing.F) {

	ledgerService := &MockLedgerServiceClient{}

	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	corpusString := "random string"
	corpusUint := uint64(0)

	// Add a seed input for each property that can be nil
	f.Add(true, false, false, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, true, false, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, true, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, true, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, false, corpusString, true, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, false, corpusString, false, true, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)

	// The following properties of a GetObjectResponse are acually used, and are fuzzed:
	// resp	*pb.GetObjectResponse
	//	.Object	*pb.Object
	//		.ObjectId 	*string
	//		.ObjectType	*string
	//		.Contents 	*pb,Bcs
	//			.Name	*string
	//			.Value	[]byte
	f.Fuzz(func(t *testing.T,
		// Response Object
		respNilOrNot bool, // determines if response itself is nil
		objectNilOrNot bool, // determines if resp.Object should be nil
		object_objectIdNilOrNot bool,
		object_objectId string,
		object_objectTypeNilOrNot bool,
		object_objectType string,
		object_contentsNilOrNot bool, // determines if resp.Object.Contents should be nil
		object_contents_nameNilOrNot bool,
		object_contents_name string,
		object_contents_value []byte,

		// GetObject input
		input_objectId string,
		input_version uint64,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetObjectResponse{}

		// set resp to nil or not
		if respNilOrNot {
			resp = nil
		} else {
			// set resp.Object to nil or not
			if objectNilOrNot {
				resp.Object = nil
			} else {
				resp.Object = &pb.Object{}

				// Set resp.Object.ObjectId
				if !object_objectIdNilOrNot {
					resp.Object.ObjectId = &object_objectId
				}

				// Set resp.Object.ObjectType
				if !object_objectTypeNilOrNot {
					resp.Object.ObjectType = &object_objectType
				}

				// set resp.Object.Contents to nil or not
				if object_contentsNilOrNot {
					resp.Object.Contents = nil
				} else {
					resp.Object.Contents = &pb.Bcs{}

					// Set resp.Object.Contents.Name
					if !object_contents_nameNilOrNot {
						resp.Object.Contents.Name = &object_contents_name
					}

					// Set resp.Object.Contents.Value
					if len(object_contents_value) == 0 || object_contents_value == nil {
						resp.Object.Contents.Value = nil
					} else {
						resp.Object.Contents.Value = object_contents_value
					}
				}

			}
		}

		ledgerService.SetNextGetObjectResponse(resp)

		fields := []string{
			ObjectFieldObjectID,
			ObjectFieldObjectType,
			ObjectFieldContents,
		}

		// Request GetObject if version is even
		if input_version%2 == 0 {
			_, _ = grpcClient.GetObject(context.Background(), input_objectId, fields)
		} else {
			// Request GetObjectAtVersion if version is odd
			_, _ = grpcClient.GetObjectAtVersion(context.Background(), input_objectId, &input_version, fields)
		}

		ledgerService.SetNextGetObjectResponse(nil)
	})
}

func FuzzSuiGrpcClientGetCheckpoint(f *testing.F) {

	ledgerService := &MockLedgerServiceClient{}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	// Add a seed input for each property that can be nil
	f.Add(true, false, uint64(0))
	f.Add(false, true, uint64(0))

	// Properties being used:
	// resp *pb.GetCheckpointResponse
	//	Checkpoint *pb.Checkpoint
	//		SequenceNumber *uint64
	f.Fuzz(func(t *testing.T,
		respNilOrNot bool,
		checkpointNilOrNot bool,
		sequenceNumber uint64,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetCheckpointResponse{}

		if respNilOrNot {
			resp = nil
		} else {
			if checkpointNilOrNot {
				resp.Checkpoint = nil
			} else {
				resp.Checkpoint = &pb.Checkpoint{}

				if sequenceNumber%2 == 0 {
					resp.Checkpoint.SequenceNumber = nil
				} else {
					resp.Checkpoint.SequenceNumber = &sequenceNumber
				}
			}
		}

		ledgerService.SetNextGetCheckpointResponse(resp)
		_, _ = grpcClient.GetLatestCheckpoint(context.Background(), []string{CheckpointFieldSequenceNumber})

		ledgerService.SetNextGetCheckpointResponse(nil)

	})
}

func FuzzSuiGrpcClientGetTransactionNoEvents(f *testing.F) {
	// The mock responses don't include events in the transaction. This is deliberate, since
	// the datatypes become unnecessarily complicated, and fuzzing the event parsing can be
	// done separately.

	ledgerService := &MockLedgerServiceClient{}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	// Add a seed input for each property that can be nil
	f.Add(true, false, false, "random string")
	f.Add(false, true, false, "random string")
	f.Add(false, false, true, "random string")

	f.Fuzz(func(t *testing.T,
		respNilOrNot bool,
		transactionNilOrNot bool,
		transaction_digestNilOrNot bool,
		transaction_digest string,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetTransactionResponse{}

		if respNilOrNot {
			resp = nil
		} else {
			if transactionNilOrNot {
				resp.Transaction = nil
			} else {
				// The successful status keeps the post-status-check conversion path
				// exercised; the status check itself is covered by unit tests.
				resp.Transaction = &pb.ExecutedTransaction{Effects: successfulEffects()}

				if !transaction_digestNilOrNot {
					resp.Transaction.Digest = &transaction_digest
				}
			}
		}

		ledgerService.SetNextGetTransactionResponse(resp)
		_, _ = grpcClient.GetTransaction(context.Background(), "some digest", []string{
			TransactionFieldDigest,
			TransactionFieldEvents,
		})
		ledgerService.SetNextGetTransactionResponse(nil)
	})
}

func FuzzExecutedTransactionToSuiTransaction(f *testing.F) {

	// This fuzzer accepts a number for each property within a pb.Event. The
	// maximum of the set is then used to determine how many events are created,
	// and for each `property` only `numProperty` amount of entries in the list
	// will be non-nil.

	u8_0 := uint8(0)
	u8_1 := uint8(1)

	// Add seed inputs for each case where a single property exists, but the others are nil.
	f.Add(u8_1, u8_0, u8_0, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_1, u8_0, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_1, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_1, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_1, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_0, u8_1, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_0, u8_0, u8_1)

	txDigest := "0xDigest"

	// Default values for properties
	defaultPackageId := "PackageId"
	defaultModule := "Module"
	defaultSender := "Sender"
	defaultEventType := "EventType"
	defaultContentsName := "Contents.Name"
	defaultContentsValue := []byte{0x13, 0x37}

	f.Fuzz(func(t *testing.T,
		numPackageIds uint8,
		numModules uint8,
		numSenders uint8,
		numEventTypes uint8,
		numContents uint8,
		numContentsName uint8,
		numContentsBcs uint8,
	) {

		entries := max(numPackageIds, numModules, numSenders, numEventTypes, numContents, numContentsName, numContentsBcs)

		grpcTransaction := &pb.ExecutedTransaction{
			Digest: &txDigest,
		}

		grpcTransaction.Events = &pb.TransactionEvents{}

		for idx := range entries {
			grpcEvent := &pb.Event{}

			if idx < numPackageIds {
				grpcEvent.PackageId = &defaultPackageId
			}

			if idx < numModules {
				grpcEvent.Module = &defaultModule
			}

			if idx < numSenders {
				grpcEvent.Sender = &defaultSender
			}

			if idx < numEventTypes {
				grpcEvent.EventType = &defaultEventType
			}

			if idx < numContents {
				grpcEvent.Contents = &pb.Bcs{}

				if idx < numContentsName {
					grpcEvent.Contents.Name = &defaultContentsName
				}

				if idx < numContentsBcs {
					grpcEvent.Contents.Value = defaultContentsValue
				}

			}

			grpcTransaction.Events.Events = append(grpcTransaction.Events.Events, grpcEvent)
		}

		grpcExecutedTransactionToSuiTransaction(grpcTransaction)

	})
}

// MockSubscribeCheckpointsStream is a mock server-streaming client for the Sui
// SubscribeCheckpoints RPC. Recv() returns each queued response in order, and once
// the queue is exhausted it returns recvErr. recvErr must be non-nil so that the
// subscription's background goroutine terminates deterministically during fuzzing.
type MockSubscribeCheckpointsStream struct {
	responses []*pb.SubscribeCheckpointsResponse
	idx       int
	recvErr   error
}

func (m *MockSubscribeCheckpointsStream) Recv() (*pb.SubscribeCheckpointsResponse, error) {
	if m.idx < len(m.responses) {
		resp := m.responses[m.idx]
		m.idx++
		return resp, nil
	}
	return nil, m.recvErr
}

// The remaining methods satisfy the grpc.ClientStream portion of the
// SubscriptionService_SubscribeCheckpointsClient interface. None of them are
// exercised by the subscription logic under test, so they are trivial stubs.
func (m *MockSubscribeCheckpointsStream) Header() (metadata.MD, error) { return nil, nil }
func (m *MockSubscribeCheckpointsStream) Trailer() metadata.MD         { return nil }
func (m *MockSubscribeCheckpointsStream) CloseSend() error             { return nil }
func (m *MockSubscribeCheckpointsStream) Context() context.Context     { return context.Background() }
func (m *MockSubscribeCheckpointsStream) SendMsg(_ any) error          { return nil }
func (m *MockSubscribeCheckpointsStream) RecvMsg(_ any) error          { return nil }

// MockSubscriptionServiceClient is a mock SubscriptionService client. SubscribeCheckpoints
// returns the configured stream and error, which allows both the stream-creation failure
// path and the streaming path of SubscribeToEvents to be exercised.
type MockSubscriptionServiceClient struct {
	nextStream pb.SubscriptionService_SubscribeCheckpointsClient
	nextError  error
}

func (m *MockSubscriptionServiceClient) SubscribeCheckpoints(ctx context.Context, req *pb.SubscribeCheckpointsRequest) (pb.SubscriptionService_SubscribeCheckpointsClient, error) {
	return m.nextStream, m.nextError
}

// releaseStream blocks in Recv until release is closed, then returns recvErr. It lets a
// test control exactly when the subscription goroutine observes a stream error.
type releaseStream struct {
	release chan struct{}
	recvErr error
}

func (m *releaseStream) Recv() (*pb.SubscribeCheckpointsResponse, error) {
	<-m.release
	return nil, m.recvErr
}

func (m *releaseStream) Header() (metadata.MD, error) { return nil, nil }
func (m *releaseStream) Trailer() metadata.MD         { return nil }
func (m *releaseStream) CloseSend() error             { return nil }
func (m *releaseStream) Context() context.Context     { return context.Background() }
func (m *releaseStream) SendMsg(_ any) error          { return nil }
func (m *releaseStream) RecvMsg(_ any) error          { return nil }

func FuzzSuiGrpcClientSubscribeToEvents(f *testing.F) {
	// Default values for event properties.
	txDigest := "0xDigest"
	defaultPackageId := "PackageId"
	defaultModule := "Module"
	defaultSender := "Sender"
	defaultEventType := "EventType"
	defaultContentsName := "Contents.Name"
	defaultContentsValue := []byte{0x13, 0x37}

	// Seed inputs covering: stream-creation failure, a nil checkpoint response, a nil
	// checkpoint, a fully-populated matching event, the single-event Subscribe variant,
	// and early unsubscription with multiple transactions.
	f.Add(true, false, false, false, false, false, uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, true, false, false, uint8(1), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, false, true, false, uint8(1), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, false, false, true, uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1))
	f.Add(false, true, false, false, false, true, uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1))
	f.Add(false, false, true, false, false, true, uint8(3), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2))

	// The structure of the events mirrors FuzzExecutedTransactionToSuiTransaction: the
	// maximum of the `num*` inputs determines how many events each transaction holds, and
	// for each property only `numProperty` of those events have that property set.
	f.Fuzz(func(t *testing.T,
		streamCreationFails bool, // SubscribeCheckpoints returns an error
		useSingleSubscribe bool, // call SubscribeToEvent instead of SubscribeToEvents
		unsubscribeEarly bool, // cancel the subscription context before draining
		respNil bool, // the streamed SubscribeCheckpointsResponse is nil
		checkpointNil bool, // the response's Checkpoint is nil
		matchEventType bool, // the subscribed event type matches the events' type
		numTransactions uint8,
		numPackageIds uint8,
		numModules uint8,
		numSenders uint8,
		numEventTypes uint8,
		numContents uint8,
		numContentsName uint8,
		numContentsBcs uint8,
	) {
		// Bound the work so a single fuzz input cannot create an unbounded number of events.
		const maxTransactions = 8
		const maxEventsPerTx = 16
		txCount := int(min(numTransactions, maxTransactions))
		entries := int(min(max(numPackageIds, numModules, numSenders, numEventTypes, numContents, numContentsName, numContentsBcs), maxEventsPerTx))

		// Build the checkpoint response that the mock stream will emit once.
		var resp *pb.SubscribeCheckpointsResponse
		if !respNil {
			resp = &pb.SubscribeCheckpointsResponse{}
			if !checkpointNil {
				checkpoint := &pb.Checkpoint{}
				for range txCount {
					grpcTx := &pb.ExecutedTransaction{
						Digest: &txDigest,
						Events: &pb.TransactionEvents{},
						// The successful status keeps the event-matching path exercised;
						// the status check itself is covered by unit tests.
						Effects: successfulEffects(),
					}
					for idx := range entries {
						grpcEvent := &pb.Event{}

						if idx < int(numPackageIds) {
							grpcEvent.PackageId = &defaultPackageId
						}
						if idx < int(numModules) {
							grpcEvent.Module = &defaultModule
						}
						if idx < int(numSenders) {
							grpcEvent.Sender = &defaultSender
						}
						if idx < int(numEventTypes) {
							grpcEvent.EventType = &defaultEventType
						}
						if idx < int(numContents) {
							grpcEvent.Contents = &pb.Bcs{}

							if idx < int(numContentsName) {
								grpcEvent.Contents.Name = &defaultContentsName
							}
							if idx < int(numContentsBcs) {
								grpcEvent.Contents.Value = defaultContentsValue
							}
						}

						grpcTx.Events.Events = append(grpcTx.Events.Events, grpcEvent)
					}
					checkpoint.Transactions = append(checkpoint.Transactions, grpcTx)
				}
				resp.Checkpoint = checkpoint
			}
		}

		// Configure the mock subscription service.
		subscriptionService := &MockSubscriptionServiceClient{}
		if streamCreationFails {
			subscriptionService.nextError = errors.New("stream creation failed")
		} else {
			subscriptionService.nextStream = &MockSubscribeCheckpointsStream{
				responses: []*pb.SubscribeCheckpointsResponse{resp},
				// io.EOF terminates the subscription goroutine after the single response.
				recvErr: io.EOF,
			}
		}

		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)

		// Buffer the channel generously so the subscription goroutine never blocks while
		// writing events. At most maxTransactions*maxEventsPerTx events can be produced.
		eventChan := make(chan SuiTransactionEvent, maxTransactions*maxEventsPerTx+1)

		eventTypes := []string{"non-matching-event-type"}
		if matchEventType {
			eventTypes = []string{defaultEventType}
		}

		var subscription SuiSubscription
		var err error
		if useSingleSubscribe {
			subscription, err = grpcClient.SubscribeToTransactionEvent(context.Background(), eventTypes[0], eventChan)
		} else {
			subscription, err = grpcClient.SubscribeToTransactionEvents(context.Background(), eventTypes, eventChan)
		}

		// When stream creation fails there is no background goroutine to wait on.
		if err != nil {
			return
		}

		if unsubscribeEarly {
			subscription.Unsubscribe()
		}

		// Wait for the subscription's background goroutine to fully exit. The error channel
		// is buffered, so the goroutine never blocks even though it is not drained here.
		// The wait is bounded so a mutant that never terminates the goroutine fails the
		// test instead of hanging the whole run.
		done := subscription.Done()
		require.NotNil(t, done)
		select {
		case <-done:
		case <-time.After(subscriptionExitTimeout):
			t.Fatal("subscription goroutine did not exit")
		}

		// Unsubscribe again to confirm it is safe to call after the goroutine has exited.
		subscription.Unsubscribe()
	})
}

// TestGetTransactionEnforcesExecutionStatus verifies that GetTransaction returns an error for
// any transaction that does not carry an explicit successful execution status, including all
// the partially-populated status shapes (fail closed).
func TestGetTransactionEnforcesExecutionStatus(t *testing.T) {
	digest := "0xDigest"
	successFalse := false

	cases := []struct {
		name    string
		effects *pb.TransactionEffects
		wantErr bool
	}{
		{name: "success", effects: successfulEffects(), wantErr: false},
		{name: "failed", effects: &pb.TransactionEffects{Status: &pb.ExecutionStatus{Success: &successFalse}}, wantErr: true},
		{name: "success field missing", effects: &pb.TransactionEffects{Status: &pb.ExecutionStatus{}}, wantErr: true},
		{name: "status missing", effects: &pb.TransactionEffects{}, wantErr: true},
		{name: "effects missing", effects: nil, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ledgerService := &MockLedgerServiceClient{}
			grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

			ledgerService.SetNextGetTransactionResponse(&pb.GetTransactionResponse{
				Transaction: &pb.ExecutedTransaction{
					Digest:  &digest,
					Effects: tc.effects,
				},
			})

			tx, err := grpcClient.GetTransaction(context.Background(), digest, []string{TransactionFieldDigest})

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tx.TxDigest)
			}
		})
	}
}

// TestSubscribeDropsEventsFromFailedTransactions verifies that the event subscription only
// forwards events from transactions with an explicit successful execution status. The streamed
// checkpoint contains one successful and one failed transaction, each carrying an otherwise
// identical matching event; only the successful transaction's event must be delivered.
func TestSubscribeDropsEventsFromFailedTransactions(t *testing.T) {
	successDigest := "0xSuccess"
	failedDigest := "0xFailed"
	packageId := "PackageId"
	module := "Module"
	sender := "Sender"
	eventType := "EventType"
	contentsName := "Contents.Name"
	contentsValue := []byte{0x13, 0x37}
	successFalse := false

	makeEvent := func() *pb.Event {
		return &pb.Event{
			PackageId: &packageId,
			Module:    &module,
			Sender:    &sender,
			EventType: &eventType,
			Contents: &pb.Bcs{
				Name:  &contentsName,
				Value: contentsValue,
			},
		}
	}

	resp := &pb.SubscribeCheckpointsResponse{
		Checkpoint: &pb.Checkpoint{
			Transactions: []*pb.ExecutedTransaction{
				{
					Digest:  &successDigest,
					Events:  &pb.TransactionEvents{Events: []*pb.Event{makeEvent()}},
					Effects: successfulEffects(),
				},
				{
					Digest:  &failedDigest,
					Events:  &pb.TransactionEvents{Events: []*pb.Event{makeEvent()}},
					Effects: &pb.TransactionEffects{Status: &pb.ExecutionStatus{Success: &successFalse}},
				},
			},
		},
	}

	subscriptionService := &MockSubscriptionServiceClient{
		nextStream: &MockSubscribeCheckpointsStream{
			responses: []*pb.SubscribeCheckpointsResponse{resp},
			recvErr:   io.EOF,
		},
	}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)

	eventChan := make(chan SuiTransactionEvent, 4)
	subscription, err := grpcClient.SubscribeToTransactionEvent(context.Background(), eventType, eventChan)
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	// Wait for the subscription goroutine to process the single response and exit. The
	// wait is bounded so a mutant that never terminates the goroutine fails the test
	// instead of hanging the whole run.
	select {
	case <-subscription.Done():
	case <-time.After(subscriptionExitTimeout):
		t.Fatal("subscription goroutine did not exit")
	}

	var received []SuiTransactionEvent
	for len(eventChan) > 0 {
		received = append(received, <-eventChan)
	}

	require.Len(t, received, 1)
	require.Equal(t, successDigest, received[0].TxDigest)
}

// TestGetObjectErrorPaths verifies that GetObject/GetObjectAtVersion reject an empty
// field list, propagate transport errors, and reject nil top-level responses.
//
// The valid-response cases deliberately pair a good response with the failure condition:
// a mutant that ignores the guard would then return success, so require.Error discriminates.
func TestGetObjectErrorPaths(t *testing.T) {
	objectID := "0xObject"
	version := uint64(7)
	fields := []string{ObjectFieldObjectID}
	validResponse := &pb.GetObjectResponse{Object: &pb.Object{ObjectId: &objectID}}

	t.Run("empty fields", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetObjectResponse(validResponse)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetObject(context.Background(), objectID, nil)
		require.Error(t, err)

		_, err = grpcClient.GetObjectAtVersion(context.Background(), objectID, &version, nil)
		require.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetObjectResponse(validResponse)
		ledgerService.SetNextGetObjectError(errors.New("transport failed"))
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetObject(context.Background(), objectID, fields)
		require.Error(t, err)
	})

	t.Run("nil response", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetObjectResponse(nil)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetObject(context.Background(), objectID, fields)
		require.Error(t, err)
	})

	t.Run("nil object", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetObjectResponse(&pb.GetObjectResponse{Object: nil})
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetObject(context.Background(), objectID, fields)
		require.Error(t, err)
	})
}

// TestGetLatestCheckpointErrorPaths verifies that GetLatestCheckpoint rejects an empty
// field list, propagates transport errors, and rejects nil top-level responses.
func TestGetLatestCheckpointErrorPaths(t *testing.T) {
	fields := []string{CheckpointFieldSequenceNumber}
	seq := uint64(1)
	validResponse := &pb.GetCheckpointResponse{Checkpoint: &pb.Checkpoint{SequenceNumber: &seq}}

	t.Run("empty fields", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetCheckpointResponse(validResponse)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetLatestCheckpoint(context.Background(), nil)
		require.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetCheckpointResponse(validResponse)
		ledgerService.SetNextGetCheckpointError(errors.New("transport failed"))
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetLatestCheckpoint(context.Background(), fields)
		require.Error(t, err)
	})

	t.Run("nil response", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetCheckpointResponse(nil)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetLatestCheckpoint(context.Background(), fields)
		require.Error(t, err)
	})

	t.Run("nil checkpoint", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetCheckpointResponse(&pb.GetCheckpointResponse{Checkpoint: nil})
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetLatestCheckpoint(context.Background(), fields)
		require.Error(t, err)
	})
}

// TestGetTransactionErrorPaths verifies that GetTransaction rejects an empty field list,
// propagates transport errors, and rejects nil top-level responses.
func TestGetTransactionErrorPaths(t *testing.T) {
	digest := "0xDigest"
	fields := []string{TransactionFieldDigest}
	validResponse := &pb.GetTransactionResponse{
		Transaction: &pb.ExecutedTransaction{Digest: &digest, Effects: successfulEffects()},
	}

	t.Run("empty fields", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetTransactionResponse(validResponse)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetTransaction(context.Background(), digest, nil)
		require.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetTransactionResponse(validResponse)
		ledgerService.SetNextGetTransactionError(errors.New("transport failed"))
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetTransaction(context.Background(), digest, fields)
		require.Error(t, err)
	})

	t.Run("nil response", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetTransactionResponse(nil)
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetTransaction(context.Background(), digest, fields)
		require.Error(t, err)
	})

	t.Run("nil transaction", func(t *testing.T) {
		ledgerService := &MockLedgerServiceClient{}
		ledgerService.SetNextGetTransactionResponse(&pb.GetTransactionResponse{Transaction: nil})
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

		_, err := grpcClient.GetTransaction(context.Background(), digest, fields)
		require.Error(t, err)
	})
}

// TestSubscribeToTransactionEventsValidation verifies the argument validation and the
// stream-creation failure path of SubscribeToTransactionEvents.
func TestSubscribeToTransactionEventsValidation(t *testing.T) {
	t.Run("empty event types", func(t *testing.T) {
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, &MockSubscriptionServiceClient{})
		_, err := grpcClient.SubscribeToTransactionEvents(context.Background(), nil, make(chan SuiTransactionEvent, 1))
		require.Error(t, err)
	})

	t.Run("nil channel", func(t *testing.T) {
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, &MockSubscriptionServiceClient{})
		_, err := grpcClient.SubscribeToTransactionEvents(context.Background(), []string{"EventType"}, nil)
		require.Error(t, err)
	})

	t.Run("stream creation error", func(t *testing.T) {
		subscriptionService := &MockSubscriptionServiceClient{nextError: errors.New("stream creation failed")}
		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)
		_, err := grpcClient.SubscribeToTransactionEvents(context.Background(), []string{"EventType"}, make(chan SuiTransactionEvent, 1))
		require.Error(t, err)
	})
}

// TestGetTransactionDoesNotDuplicateStatusField verifies that GetTransaction appends the
// execution-status field only when the caller did not already request it, and never mutates
// the caller's slice.
func TestGetTransactionDoesNotDuplicateStatusField(t *testing.T) {
	digest := "0xDigest"
	ledgerService := &MockLedgerServiceClient{}
	ledgerService.SetNextGetTransactionResponse(&pb.GetTransactionResponse{
		Transaction: &pb.ExecutedTransaction{Digest: &digest, Effects: successfulEffects()},
	})
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	fields := []string{TransactionFieldDigest, TransactionFieldStatus}
	_, err := grpcClient.GetTransaction(context.Background(), digest, fields)
	require.NoError(t, err)

	require.Equal(t, []string{TransactionFieldDigest, TransactionFieldStatus}, ledgerService.lastGetTransactionRequest.ReadMask.Paths)
	require.Equal(t, []string{TransactionFieldDigest, TransactionFieldStatus}, fields)
}

// TestGrpcEventToSuiEvent verifies that grpcEventToSuiEvent rejects an event missing any
// required field and maps a fully-populated event into the wrapper struct.
func TestGrpcEventToSuiEvent(t *testing.T) {
	packageID := "PackageId"
	module := "Module"
	sender := "Sender"
	eventType := "EventType"
	contentsName := "Contents.Name"
	contentsValue := []byte{0x13, 0x37}

	full := func() *pb.Event {
		return &pb.Event{
			PackageId: &packageID,
			Module:    &module,
			Sender:    &sender,
			EventType: &eventType,
			Contents:  &pb.Bcs{Name: &contentsName, Value: contentsValue},
		}
	}

	t.Run("nil event", func(t *testing.T) {
		require.Nil(t, grpcEventToSuiEvent(nil))
	})

	// Each case drops exactly one required field; the result must be nil.
	dropField := map[string]func(*pb.Event){
		"package id":    func(e *pb.Event) { e.PackageId = nil },
		"module":        func(e *pb.Event) { e.Module = nil },
		"sender":        func(e *pb.Event) { e.Sender = nil },
		"event type":    func(e *pb.Event) { e.EventType = nil },
		"contents":      func(e *pb.Event) { e.Contents = nil },
		"contents name": func(e *pb.Event) { e.Contents.Name = nil },
		"contents value": func(e *pb.Event) {
			e.Contents.Value = nil
		},
	}
	for name, drop := range dropField {
		t.Run("missing "+name, func(t *testing.T) {
			event := full()
			drop(event)
			require.Nil(t, grpcEventToSuiEvent(event))
		})
	}

	t.Run("fully populated", func(t *testing.T) {
		got := grpcEventToSuiEvent(full())
		require.NotNil(t, got)
		require.Equal(t, packageID, got.PackageID)
		require.Equal(t, module, got.TransactionModule)
		require.Equal(t, sender, got.Sender)
		require.Equal(t, eventType, got.EventType)
		require.Equal(t, contentsName, got.BcsType)
		require.Equal(t, contentsValue, got.BcsBytes)
	})
}

// TestGrpcExecutedTransactionToSuiTransaction verifies the nil guard, timestamp mapping,
// event filtering, and the nil-element guard in the changed-objects loop.
func TestGrpcExecutedTransactionToSuiTransaction(t *testing.T) {
	t.Run("nil transaction", func(t *testing.T) {
		require.Equal(t, SuiTransaction{}, grpcExecutedTransactionToSuiTransaction(nil))
	})

	t.Run("timestamp mapped", func(t *testing.T) {
		ts := timestamppb.New(time.Unix(1700000000, 0))
		tx := &pb.ExecutedTransaction{Timestamp: ts}
		got := grpcExecutedTransactionToSuiTransaction(tx)
		require.NotNil(t, got.Timestamp)
		require.Equal(t, ts.AsTime(), *got.Timestamp)
	})

	t.Run("nil changed object skipped", func(t *testing.T) {
		objectID := "0xObject"
		tx := &pb.ExecutedTransaction{
			Effects: &pb.TransactionEffects{
				ChangedObjects: []*pb.ChangedObject{
					nil,
					{ObjectId: &objectID},
				},
			},
		}
		got := grpcExecutedTransactionToSuiTransaction(tx)
		require.Len(t, got.ObjectChanges, 1)
		require.Equal(t, objectID, *got.ObjectChanges[0].ObjectID)
	})

	t.Run("malformed event skipped", func(t *testing.T) {
		packageID := "PackageId"
		module := "Module"
		sender := "Sender"
		eventType := "EventType"
		contentsName := "Contents.Name"
		tx := &pb.ExecutedTransaction{
			Events: &pb.TransactionEvents{
				Events: []*pb.Event{
					{PackageId: &packageID}, // missing required fields
					{
						PackageId: &packageID,
						Module:    &module,
						Sender:    &sender,
						EventType: &eventType,
						Contents:  &pb.Bcs{Name: &contentsName, Value: []byte{0x01}},
					},
				},
			},
		}
		got := grpcExecutedTransactionToSuiTransaction(tx)
		require.Len(t, got.Events, 1)
		require.Equal(t, eventType, got.Events[0].EventType)
	})
}

// TestGrpcObjectToSuiObject verifies the nil guard and the Bcs/Contents flattening.
func TestGrpcObjectToSuiObject(t *testing.T) {
	t.Run("nil object", func(t *testing.T) {
		require.Equal(t, SuiObject{}, grpcObjectToSuiObject(nil))
	})

	t.Run("flattened", func(t *testing.T) {
		objectID := "0xObject"
		version := uint64(3)
		digest := "0xDigest"
		objectType := "0x2::coin::Coin"
		previousTransaction := "0xPrev"
		storageRebate := uint64(10)
		balance := uint64(99)
		bcsName := "bcs.name"
		bcsValue := []byte{0xaa}
		contentsName := "contents.name"
		contentsValue := []byte{0xbb}

		got := grpcObjectToSuiObject(&pb.Object{
			ObjectId:            &objectID,
			Version:             &version,
			Digest:              &digest,
			ObjectType:          &objectType,
			PreviousTransaction: &previousTransaction,
			StorageRebate:       &storageRebate,
			Balance:             &balance,
			Bcs:                 &pb.Bcs{Name: &bcsName, Value: bcsValue},
			Contents:            &pb.Bcs{Name: &contentsName, Value: contentsValue},
		})

		require.Equal(t, objectID, *got.ObjectID)
		require.Equal(t, version, *got.Version)
		require.Equal(t, digest, *got.Digest)
		require.Equal(t, objectType, *got.ObjectType)
		require.Equal(t, previousTransaction, *got.PreviousTransaction)
		require.Equal(t, storageRebate, *got.StorageRebate)
		require.Equal(t, balance, *got.Balance)
		require.Equal(t, bcsName, *got.BcsType)
		require.Equal(t, bcsValue, got.BcsBytes)
		require.Equal(t, contentsName, *got.ContentsType)
		require.Equal(t, contentsValue, got.ContentsBytes)
	})
}

// TestGrpcCheckpointToSuiCheckpoint verifies the nil guard and field mapping.
func TestGrpcCheckpointToSuiCheckpoint(t *testing.T) {
	t.Run("nil checkpoint", func(t *testing.T) {
		require.Equal(t, SuiCheckpoint{}, grpcCheckpointToSuiCheckpoint(nil))
	})

	t.Run("mapped", func(t *testing.T) {
		seq := uint64(42)
		digest := "0xDigest"
		got := grpcCheckpointToSuiCheckpoint(&pb.Checkpoint{SequenceNumber: &seq, Digest: &digest})
		require.Equal(t, seq, *got.SequenceNumber)
		require.Equal(t, digest, *got.Digest)
	})
}

// TestFieldMask verifies that fieldMask preserves the requested paths.
func TestFieldMask(t *testing.T) {
	fields := []string{"a", "b.c", "d"}
	mask := fieldMask(fields)
	require.NotNil(t, mask)
	require.Equal(t, fields, mask.Paths)
}

// TestDecodeBcs verifies that DecodeBcs round-trips a valid value and returns an error
// (with a nil result) for malformed input.
func TestDecodeBcs(t *testing.T) {
	type bcsTestStruct struct {
		Value uint64
	}

	t.Run("valid", func(t *testing.T) {
		encoded, err := mystenbcs.Marshal(bcsTestStruct{Value: 7})
		require.NoError(t, err)

		got, err := DecodeBcs[bcsTestStruct](encoded)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, uint64(7), got.Value)
	})

	t.Run("invalid", func(t *testing.T) {
		got, err := DecodeBcs[bcsTestStruct]([]byte{})
		require.Error(t, err)
		require.Nil(t, got)
	})
}

// TestSuiSubscriptionAccessors verifies that Err and Done expose the subscription's
// channels rather than nil.
func TestSuiSubscriptionAccessors(t *testing.T) {
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})
	sub := SuiSubscription{err: errChan, done: doneChan, ctxCancel: func() {}}

	require.NotNil(t, sub.Err())
	require.NotNil(t, sub.Done())

	sub.Unsubscribe()
}

// TestSubscribeContextCancellationDoesNotReportError verifies that a stream error observed
// after the subscription context is cancelled is treated as a clean shutdown: the error
// channel is closed without delivering an error.
func TestSubscribeContextCancellationDoesNotReportError(t *testing.T) {
	stream := &releaseStream{release: make(chan struct{}), recvErr: errors.New("recv failed")}
	subscriptionService := &MockSubscriptionServiceClient{nextStream: stream}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)

	ctx, cancel := context.WithCancel(context.Background())
	subscription, err := grpcClient.SubscribeToTransactionEvent(ctx, "EventType", make(chan SuiTransactionEvent, 1))
	require.NoError(t, err)

	// Cancel first, then let Recv return its error: the goroutine must observe a cancelled
	// context and exit without reporting the stream error.
	cancel()
	close(stream.release)

	select {
	case <-subscription.Done():
	case <-time.After(subscriptionExitTimeout):
		t.Fatal("subscription goroutine did not exit")
	}

	select {
	case err, ok := <-subscription.Err():
		if ok {
			t.Fatalf("unexpected error delivered after cancellation: %v", err)
		}
	default:
	}
}

// TestSubscribeSkipsTransactionsWithoutEvents verifies that a transaction with a nil Events
// message is skipped rather than dereferenced.
func TestSubscribeSkipsTransactionsWithoutEvents(t *testing.T) {
	digest := "0xDigest"
	resp := &pb.SubscribeCheckpointsResponse{
		Checkpoint: &pb.Checkpoint{
			Transactions: []*pb.ExecutedTransaction{
				{Digest: &digest, Events: nil, Effects: successfulEffects()},
			},
		},
	}
	subscriptionService := &MockSubscriptionServiceClient{
		nextStream: &MockSubscribeCheckpointsStream{
			responses: []*pb.SubscribeCheckpointsResponse{resp},
			recvErr:   io.EOF,
		},
	}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)

	eventChan := make(chan SuiTransactionEvent, 1)
	subscription, err := grpcClient.SubscribeToTransactionEvent(context.Background(), "EventType", eventChan)
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	select {
	case <-subscription.Done():
	case <-time.After(subscriptionExitTimeout):
		t.Fatal("subscription goroutine did not exit")
	}

	require.Empty(t, eventChan)
}

// TestNewSuiGrpcClientNilLogger verifies that a nil logger is replaced with a usable one.
func TestNewSuiGrpcClientNilLogger(t *testing.T) {
	client, err := NewSuiGrpcClient("localhost:443", nil)
	require.NoError(t, err)
	defer client.Close() //nolint:errcheck // The Close error is not relevant for this test

	grpcClient, ok := client.(*SuiGrpcClient)
	require.True(t, ok)
	require.NotNil(t, grpcClient.logger)
}

// TestNewSuiGrpcClientCreationError verifies that a target grpc.NewClient rejects surfaces
// as an error rather than a nil-error, nil-client return.
func TestNewSuiGrpcClientCreationError(t *testing.T) {
	client, err := NewSuiGrpcClient("\x00", zap.NewNop())
	require.Error(t, err)
	require.Nil(t, client)
}

// TestCloseNilConn verifies that Close is safe on a client whose connection was never set.
func TestCloseNilConn(t *testing.T) {
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, nil)
	require.NoError(t, grpcClient.Close())
}

// TestCloseClosesConnection verifies that Close actually shuts the underlying connection
// down rather than merely returning nil.
func TestCloseClosesConnection(t *testing.T) {
	client, err := NewSuiGrpcClient("localhost:443", zap.NewNop())
	require.NoError(t, err)

	grpcClient, ok := client.(*SuiGrpcClient)
	require.True(t, ok)

	require.NoError(t, grpcClient.Close())
	require.Equal(t, connectivity.Shutdown, grpcClient.conn.GetState())
}

// TestNewSuiGrpcClientExtraOptions verifies that caller-supplied dial options are accepted
// and appended after the defaults. Passing more options than the defaults also guards the
// capacity arithmetic in NewSuiGrpcClient against a negative capacity.
func TestNewSuiGrpcClientExtraOptions(t *testing.T) {
	extraOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024)),
		grpc.WithDisableRetry(),
	}

	client, err := NewSuiGrpcClient("localhost:443", zap.NewNop(), extraOpts...)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func FuzzNewSuiGrpcClient(f *testing.F) {
	// grpc.NewClient is lazy: it validates the target and constructs the client without
	// dialing, so this exercises NewSuiGrpcClient and Close() with no network access.
	f.Add("fullnode.mainnet.sui.io:443")
	f.Add("localhost:443")
	f.Add("")
	f.Add(":::::")
	f.Add("dns:///example.com:443")

	f.Fuzz(func(t *testing.T, rpcURL string) {
		client, err := NewSuiGrpcClient(rpcURL, zap.NewNop())
		if err != nil {
			return
		}
		client.Close() //nolint:errcheck // The Close error is not relevant for the fuzz harness
	})
}
